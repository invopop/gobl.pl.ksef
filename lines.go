package ksef

import (
	"regexp"
	"strings"

	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/gobl/uuid"
)

// metaKeyUnitLabel holds the original KSeF unit of measure when it does not
// match a value accepted by GOBL (neither a defined unit nor a UN/ECE code).
// The "-label" suffix marks it as a free-form human-readable label rather
// than a canonical unit code.
const metaKeyUnitLabel cbc.Key = "unit-label"

// regexpUNECEUnit matches the UN/ECE Recommendation 20/21 unit codes accepted
// by the `untdid-unit` extension.
var regexpUNECEUnit = regexp.MustCompile(org.UnitPatternUNECE)

// Line defines the XML structure for KSeF item line (element type FaWiersz, for VAT and KOR type invoices)
type Line struct {
	LineNumber              int    `xml:"NrWierszaFa"`
	UniqueID                string `xml:"UU_ID,omitempty"`
	CompletionDate          string `xml:"P_6A,omitempty"`
	Name                    string `xml:"P_7,omitempty"`
	InternalCode            string `xml:"Indeks,omitempty"`
	GTIN                    string `xml:"GTIN,omitempty"`
	PKWiU                   string `xml:"PKWiU,omitempty"`
	CN                      string `xml:"CN,omitempty"`
	PKOB                    string `xml:"PKOB,omitempty"`
	Measure                 string `xml:"P_8A,omitempty"`
	Quantity                string `xml:"P_8B,omitempty"`
	NetUnitPrice            string `xml:"P_9A,omitempty"`
	GrossUnitPrice          string `xml:"P_9B,omitempty"`
	Discount                string `xml:"P_10,omitempty"`
	NetPriceTotal           string `xml:"P_11,omitempty"`
	GrossPriceTotal         string `xml:"P_11A,omitempty"`
	VATAmount               string `xml:"P_11Vat,omitempty"`
	VATRate                 string `xml:"P_12,omitempty"`
	OSSTaxRate              string `xml:"P_12_XII,omitempty"` // one stop shop
	Attachment15GoodsMarker int    `xml:"P_12_Zal_15,omitempty"`
	ExciseDuty              string `xml:"KwotaAkcyzy,omitempty"`
	SpecialGoodsCode        string `xml:"GTU,omitempty"` // values GTU_01 to GTU_13
	Procedure               string `xml:"Procedura,omitempty"`
	CurrencyRate            string `xml:"KursWaluty,omitempty"`
	BeforeCorrectionMarker  int    `xml:"StanPrzed,omitempty"`
}

// NewLines generates lines for the KSeF invoice assuming net-priced lines
// (P_9A/P_11).
//
// Deprecated: prefer NewLinesForInvoice, which honors tax.prices_include=VAT
// and emits gross fields (P_9B/P_11A, art. 106e(7)-(8)) for gross-priced
// invoices.
func NewLines(lines []*bill.Line) []*Line {
	return newLines(lines, false)
}

// NewLinesForInvoice generates KSeF lines from a GOBL invoice, choosing
// between net (P_9A/P_11) and gross (P_9B/P_11A) line fields based on
// invoice.Tax.PricesInclude.
func NewLinesForInvoice(invoice *bill.Invoice) []*Line {
	return newLines(invoice.Lines, invoicePricesIncludeVAT(invoice))
}

func newLines(lines []*bill.Line, pricesIncludeVAT bool) []*Line {
	var Lines []*Line

	for _, line := range lines {
		Lines = append(Lines, newLine(line, pricesIncludeVAT))
	}

	return Lines
}

func newLine(line *bill.Line, pricesIncludeVAT bool) *Line {
	l := &Line{
		LineNumber: line.Index,
		UniqueID:   string(line.UUID),
		Name:       line.Item.Name,
		Measure:    lineMeasure(line),
		Quantity:   line.Quantity.String(),
		Discount:   lineDiscount(line),
	}

	if line.Period != nil && line.Period.End != nil {
		l.CompletionDate = line.Period.End.String()
	}

	if pricesIncludeVAT {
		l.GrossUnitPrice = line.Item.Price.String()
		l.GrossPriceTotal = line.Total.String()
	} else {
		l.NetUnitPrice = line.Item.Price.String()
		l.NetPriceTotal = line.Total.String()
	}

	if tc := line.Taxes.Get(tax.CategoryVAT); tc != nil {
		if tc.Ext.Get(favat.ExtKeyTaxCategory) == "5" {
			if tc.Percent != nil {
				l.OSSTaxRate = tc.Percent.Amount().MinimalString()
			}
		} else {
			l.VATRate = vatRate(tc)
		}
	}

	return l
}

// vatRate returns the VAT rate string and OSS tax rate string for a tax combo
// based on the tax category extension
func vatRate(tc *tax.Combo) string {

	// For non-zero percentages, use the percentage value
	if tc.Percent != nil && !tc.Percent.IsZero() {
		return tc.Percent.Amount().MinimalString()
	}

	// For zero/nil percentage, determine from tax category extension
	switch tc.Ext.Get(favat.ExtKeyTaxCategory) {
	case "6.1": // zero-rated goods and services in the country
		return "0 KR"
	case "6.2": // intra-community supply
		return "0 WDT"
	case "6.3": // export supply
		return "0 EX"
	case "7": // tax exempt supply
		return "zw"
	case "8": // outside scope supply
		return "np I"
	case "9": // reverse charge supply
		return "np II"
	case "10": // domestic reverse charge supply
		return "oo"
	default:
		return ""
	}
}

// lineMeasure resolves the KSeF P_8A unit of measure. When the GOBL item has
// an Item.Meta["unit-label"] entry — typically set by ToGOBL for KSeF units
// that do not match a GOBL unit key or UN/ECE code — that original value is
// used so KSeF round-trips preserve the supplier's wording.
//
// Otherwise the item's unit decides, as it does everywhere in GOBL: every unit
// it defines has an exact UNTDID code, so the `untdid-unit` extension is only
// consulted for an item with no unit, which is how a code GOBL has no key for
// is carried. An item with neither leaves P_8A empty.
func lineMeasure(line *bill.Line) string {
	if u, ok := line.Item.Meta[metaKeyUnitLabel]; ok && u != "" {
		return u
	}
	if code := untdid.UnitCode(line.Item.Unit); code != cbc.CodeEmpty {
		return code.String()
	}
	return line.Item.Ext.Get(untdid.ExtKeyUnit).String()
}

func lineDiscount(line *bill.Line) string {
	if len(line.Discounts) == 0 {
		return ""
	}

	amount := num.MakeAmount(0, 2)

	for _, discount := range line.Discounts {
		amount = amount.Add(discount.Amount)
	}

	return amount.String()
}

// ToGOBL converts a KSEF Line to a GOBL Line.
func (l *Line) ToGOBL() (*bill.Line, error) {
	line := &bill.Line{
		Item: &org.Item{
			Name: l.Name,
		},
	}

	if l.UniqueID != "" {
		if id, err := uuid.Parse(l.UniqueID); err == nil {
			line.UUID = id
		}
	}

	if l.CompletionDate != "" {
		d, err := parseDate(l.CompletionDate)
		if err != nil {
			return nil, err
		}
		end := d
		line.Period = &cal.Period{Start: &d, End: &end}
	}

	// Parse quantity
	if l.Quantity != "" {
		qty, err := parseAmount(l.Quantity)
		if err != nil {
			return nil, err
		}
		line.Quantity = qty
	}

	// Parse unit price
	if l.NetUnitPrice != "" {
		price, err := parseAmount(l.NetUnitPrice)
		if err != nil {
			return nil, err
		}
		line.Item.Price = &price
	} else if l.GrossUnitPrice != "" {
		price, err := parseAmount(l.GrossUnitPrice)
		if err != nil {
			return nil, err
		}
		line.Item.Price = &price
	}

	// If no unit price was provided, calculate from total price and quantity.
	// Rescale the total to at least 4 decimal places before dividing to
	// avoid precision loss (num.Amount.Divide preserves the numerator's scale).
	if line.Item.Price == nil && !line.Quantity.IsZero() {
		if l.NetPriceTotal != "" {
			total, err := parseAmount(l.NetPriceTotal)
			if err != nil {
				return nil, err
			}
			total = total.RescaleUp(4)
			price := total.Divide(line.Quantity)
			line.Item.Price = &price
		} else if l.GrossPriceTotal != "" {
			total, err := parseAmount(l.GrossPriceTotal)
			if err != nil {
				return nil, err
			}
			total = total.RescaleUp(4)
			price := total.Divide(line.Quantity)
			line.Item.Price = &price
		}
	}

	// Parse unit of measure. KSeF accepts free-form unit strings (e.g. "kilo",
	// "pcs."), but GOBL only accepts its own defined unit keys, with UN/ECE
	// codes held in the `untdid-unit` extension. Trim surrounding whitespace
	// before matching so that user-entered values like " KGM " are still
	// recognized as canonical. When the measure is neither, preserve the
	// trimmed value under Item.Meta["unit-label"] so the information is not
	// lost while keeping the resulting invoice valid.
	if measure := strings.TrimSpace(l.Measure); measure != "" {
		switch {
		case org.HasValidUnitKey.Check(cbc.Key(measure)):
			line.Item.Unit = cbc.Key(measure)
		case regexpUNECEUnit.MatchString(measure):
			// A code GOBL has a unit for is held as that unit, which implies
			// the code again on the way back out. Only a code with no
			// equivalent needs the extension to carry it.
			code := cbc.Code(measure)
			if unit := untdid.UnitKey(code); unit != cbc.KeyEmpty {
				line.Item.Unit = unit
			} else {
				line.Item.Ext = line.Item.Ext.Merge(tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyUnit: code,
				}))
			}
		default:
			if line.Item.Meta == nil {
				line.Item.Meta = cbc.Meta{}
			}
			line.Item.Meta[metaKeyUnitLabel] = measure
		}
	}

	// Parse discount — P_10 is the total line discount per the KSeF spec,
	// map it directly.
	if l.Discount != "" {
		discount, err := parseAmount(l.Discount)
		if err != nil {
			return nil, err
		}
		if !discount.IsZero() {
			line.Discounts = []*bill.LineDiscount{
				{
					Amount: discount,
				},
			}
		}
	}

	// Parse VAT rate and create tax combo
	var rateStr string
	if l.OSSTaxRate != "" {
		rateStr = l.OSSTaxRate
	} else if l.VATRate != "" {
		rateStr = l.VATRate
	}

	if rateStr != "" {
		taxInfo := parseVATRate(rateStr)
		taxCombo := &tax.Combo{
			Category: tax.CategoryVAT,
			Key:      taxInfo.Key,
			Rate:     taxInfo.Rate,
			Percent:  taxInfo.Percent,
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				favat.ExtKeyTaxCategory: taxInfo.TaxCategory,
			}),
		}
		line.Taxes = tax.Set{taxCombo}
	}

	return line, nil
}

// parseAmount parses a string amount to num.Amount
func parseAmount(s string) (num.Amount, error) {
	amt, err := num.AmountFromString(strings.TrimSpace(s))
	if err != nil {
		return num.Amount{}, err
	}
	return amt, nil
}

// TaxRateInfo contains the parsed tax rate information
type TaxRateInfo struct {
	Key         cbc.Key
	Rate        cbc.Key
	Percent     *num.Percentage
	TaxCategory cbc.Code
}

// parseVATRate converts KSEF VAT rate string to GOBL tax information.
// KSEF uses various formats:
// - "23", "8", "5" for standard rates
// - "0 KR" for zero-rated (6.1)
// - "0 WDT" for intra-community (6.2)
// - "0 EX" for export (6.3)
// - "zw" for exempt (7)
// - "np I" for outside scope (8)
// - "np II" for reverse charge (9)
// - "oo" for domestic reverse charge (10)
func parseVATRate(rateStr string) *TaxRateInfo {
	rateStr = strings.TrimSpace(rateStr)

	info := &TaxRateInfo{}

	switch rateStr {
	case "23":
		info.Key = tax.KeyStandard
		info.Rate = tax.RateGeneral
		pct := num.MakePercentage(230, 3)
		info.Percent = &pct
		info.TaxCategory = "1"
	case "22":
		info.Key = tax.KeyStandard
		pct := num.MakePercentage(220, 3)
		info.Percent = &pct
		info.TaxCategory = "1"
	case "8":
		info.Key = tax.KeyStandard
		info.Rate = tax.RateReduced
		pct := num.MakePercentage(80, 3)
		info.Percent = &pct
		info.TaxCategory = "2"
	case "7":
		info.Key = tax.KeyStandard
		pct := num.MakePercentage(70, 3)
		info.Percent = &pct
		info.TaxCategory = "2"
	case "5":
		info.Key = tax.KeyStandard
		info.Rate = tax.RateSuperReduced
		pct := num.MakePercentage(50, 3)
		info.Percent = &pct
		info.TaxCategory = "3"
	case "4":
		info.Key = tax.KeyStandard
		pct := num.MakePercentage(40, 3)
		info.Percent = &pct
		info.TaxCategory = "4"
	case "3":
		info.Key = tax.KeyStandard
		pct := num.MakePercentage(30, 3)
		info.Percent = &pct
		info.TaxCategory = "3"
	case "0 KR":
		info.Key = tax.KeyZero
		pct := num.MakePercentage(0, 3)
		info.Percent = &pct
		info.TaxCategory = "6.1"
	case "0 WDT":
		info.Key = tax.KeyIntraCommunity
		pct := num.MakePercentage(0, 3)
		info.Percent = &pct
		info.TaxCategory = "6.2"
	case "0 EX":
		info.Key = tax.KeyExport
		pct := num.MakePercentage(0, 3)
		info.Percent = &pct
		info.TaxCategory = "6.3"
	case "zw":
		info.Key = tax.KeyExempt
		info.TaxCategory = "7"
	case "np I":
		info.Key = tax.KeyOutsideScope
		info.TaxCategory = "8"
	case "np II":
		info.Key = tax.KeyReverseCharge
		info.TaxCategory = "9"
	case "oo":
		info.Key = tax.KeyReverseCharge
		info.TaxCategory = "10"
	}

	return info
}
