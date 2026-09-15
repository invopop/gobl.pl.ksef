// Package ksef implements conversion between GOBL documents and KSeF formats,
// including the Polish FA_VAT XML invoice document.
package ksef

import (
	"encoding/xml"
	"fmt"

	"github.com/invopop/gobl"
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/tax"
)

// Constants for KSeF XML
const (
	XSINamespace    = "http://www.w3.org/2001/XMLSchema-instance"
	XSDNamespace    = "http://www.w3.org/2001/XMLSchema"
	XMLNamespace    = "http://crd.gov.pl/wzor/2025/06/25/13775/"
	RootElementName = "Faktura"
)

// NoteSourceFooter identifies notes that should be mapped to KSeF's Stopka/Informacje/StopkaFaktury.
const NoteSourceFooter cbc.Key = "footer"

// Invoice is a pseudo-model for containing the XML document being created
type Invoice struct {
	XMLName      xml.Name
	XSINamespace string        `xml:"xmlns:xsi,attr"`
	XSDNamespace string        `xml:"xmlns:xsd,attr"`
	XMLNamespace string        `xml:"xmlns,attr"`
	Header       *Header       `xml:"Naglowek"`
	Seller       *Seller       `xml:"Podmiot1"`
	Buyer        *Buyer        `xml:"Podmiot2"`
	ThirdParties []*ThirdParty `xml:"Podmiot3,omitempty"` // third party (up to 100)
	Inv          *Inv          `xml:"Fa"`
	Footer       *Stopka       `xml:"Stopka,omitempty"`
}

// favatMaxSubunits is the number of decimal places the FA(3) TKwotowy type
// allows on every monetary amount, regardless of the document's currency.
const favatMaxSubunits uint32 = 2

// BuildFavat converts a GOBL envelope into a KSeF FA_VAT invoice document.
func BuildFavat(env *gobl.Envelope) (*Invoice, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	if !favat.V3.In(inv.GetAddons()...) {
		return nil, fmt.Errorf("invoice does not have the FA_VAT v3 addon")
	}

	// FA(3) fixes every amount (TKwotowy) at two decimal places, whatever the
	// currency, so a currency with finer subunits cannot be represented even
	// after rounding to its own precision. Report that rather than emit XML
	// the schema rejects.
	if cd := inv.Currency.Def(); cd != nil && cd.Subunits > favatMaxSubunits {
		return nil, fmt.Errorf(
			"currency %s has %d decimal places, more than the %d FA(3) allows",
			inv.Currency, cd.Subunits, favatMaxSubunits,
		)
	}

	// Documents calculated with the `precise` rounding rule keep extra
	// decimals on line totals, which the schema rejects, so recalculate them
	// with the `currency` rule and carry any change to the amount payable in
	// the totals' rounding. Documents already within the currency's precision
	// are left untouched.
	if err := inv.RoundToCurrency(); err != nil {
		return nil, fmt.Errorf("rounding invoice to currency precision: %w", err)
	}

	if inv.Type == bill.InvoiceTypeCreditNote {
		// In KSEF credit notes become corrective invoices,
		// which require negative totals.
		if err := inv.Invert(); err != nil {
			return nil, err
		}
	}

	invoice := &Invoice{
		XMLName:      xml.Name{Local: RootElementName},
		XSINamespace: XSINamespace,
		XSDNamespace: XSDNamespace,
		XMLNamespace: XMLNamespace,

		Header:       NewFavatHeader(),
		Seller:       NewFavatSeller(inv.Supplier),
		Buyer:        NewFavatBuyer(inv.Customer),
		ThirdParties: NewThirdParties(inv),
		Inv:          NewFavatInv(inv),
		Footer:       NewFavatFooter(inv),
	}

	return invoice, nil
}

// Bytes returns the XML representation of the document in bytes
func (d *Invoice) Bytes() ([]byte, error) {
	data, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), data...), nil
}

// ParseKSeF converts a KSeF FA_VAT XML document into a GOBL envelope.
func ParseKSeF(xmlData []byte) (*gobl.Envelope, error) {
	var doc Invoice
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		return nil, fmt.Errorf("unmarshaling XML: %w", err)
	}

	inv, err := doc.ToGOBL()
	if err != nil {
		return nil, fmt.Errorf("converting to GOBL: %w", err)
	}

	env, err := gobl.Envelop(inv)
	if err != nil {
		return nil, fmt.Errorf("creating envelope: %w", err)
	}

	return env, nil
}

// ToGOBL converts the KSeF Invoice to a GOBL invoice. If an error occurs
// during parsing, the invoice is still returned with any successfully parsed
// fields populated, allowing callers to access partial data.
func (d *Invoice) ToGOBL() (*bill.Invoice, error) {
	if d.Inv == nil {
		return nil, fmt.Errorf("missing invoice data")
	}

	inv := &bill.Invoice{}
	inv.Addons = tax.WithAddons(favat.V3)
	inv.Currency = currency.Code(parseCurrency(d.Inv.CurrencyCode))

	// Parse invoice data
	if err := d.Inv.parseInvoiceData(inv); err != nil {
		return inv, err
	}

	// Parse parties
	d.parseParties(inv)

	// Parse lines
	if err := d.Inv.parseLines(inv); err != nil {
		return inv, err
	}

	// Parse additional descriptions (must be after parseLines so line-level
	// notes can be attached to the correct lines)
	d.Inv.parseAdditionalDescriptions(inv)

	// Parse ordering lines (Zamowienie → Ordering.Purchases)
	if err := d.Inv.parseOrderingLines(inv); err != nil {
		return inv, err
	}

	// Parse payment
	if err := d.Inv.parsePayment(inv); err != nil {
		return inv, err
	}

	// Parse Rozliczenie charges/deductions to invoice-level Charges/Discounts
	// before AdjustRounding, so Calculate sees them when reconciling P_15.
	if err := d.Inv.parseSettlement(inv); err != nil {
		return inv, err
	}

	// For bypass invoices (prepayment without lines), set totals directly
	// from the invoice-level tax fields. Otherwise calculate and adjust rounding.
	if inv.HasTags(tax.TagBypass) {
		if err := d.Inv.parsePrepaymentTotals(inv); err != nil {
			return inv, err
		}
	} else {
		// For settlement invoices with advance invoice references,
		// derive the advance payment amount from the difference between
		// calculated payable (from lines) and P_15 (remaining amount).
		if d.Inv.isSettlementType() && len(d.Inv.AdvanceInvoices) > 0 {
			if err := d.Inv.deriveSettlementAdvances(inv, d.Inv.TotalAmountDue); err != nil {
				return inv, err
			}
		}

		// Calculate totals and adjust for rounding if needed.
		// For credit notes, line totals are now positive (GOBL convention),
		// so we must invert the KSeF P_15 total to match.
		totalDue := d.Inv.TotalAmountDue
		if inv.Type == bill.InvoiceTypeCreditNote {
			amt, err := parseAmount(totalDue)
			if err != nil {
				return inv, fmt.Errorf("parsing total amount due: %w", err)
			}
			totalDue = amt.Invert().String()
		}
		if err := AdjustRounding(inv, totalDue); err != nil {
			return inv, err
		}
	}

	return inv, nil
}

func parseCurrency(code string) cbc.Code {
	if code == "" {
		return "PLN"
	}
	return cbc.Code(code)
}

// parseParties converts KSEF parties to GOBL parties.
func (d *Invoice) parseParties(inv *bill.Invoice) {
	// Parse supplier (Podmiot1)
	if d.Seller != nil {
		inv.Supplier = d.Seller.ToGOBL()
	}

	// Parse customer (Podmiot2)
	if d.Buyer != nil {
		inv.Customer = d.Buyer.ToGOBL()
	}

	// Parse third parties (Podmiot3)
	if len(d.ThirdParties) > 0 {
		for _, tp := range d.ThirdParties {
			// Third parties can add identities to supplier or customer
			identity := tp.toIdentity()
			if identity != nil {
				// Determine if this third party belongs to supplier or customer
				// based on role codes
				if tp.Role != "" {
					role := tp.Role
					switch role {
					case "7", "9": // JST issuer, GV issuer
						if inv.Supplier != nil {
							inv.Supplier.Identities = append(inv.Supplier.Identities, identity)
						}
					case "8", "10": // JST recipient, GV recipient
						if inv.Customer != nil {
							inv.Customer.Identities = append(inv.Customer.Identities, identity)
						}
					}
				}
			}
		}
	}

}
