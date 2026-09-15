package ksef

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// validCodeRe matches valid cbc.Code values (alphanumeric, hyphens, dots, underscores).
var validCodeRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Inv defines the XML structure for KSeF invoice
type Inv struct {
	CurrencyCode                       string                       `xml:"KodWaluty"`
	IssueDate                          string                       `xml:"P_1"`
	IssuePlace                         string                       `xml:"P_1M,omitempty"`
	SequentialNumber                   string                       `xml:"P_2"`
	WarehouseDocuments                 []string                     `xml:"WZ,omitempty"`
	CompletionDate                     string                       `xml:"P_6,omitempty"`
	Period                             *InvoicePeriod               `xml:"OkresFa,omitempty"`
	StandardRateNetSale                string                       `xml:"P_13_1,omitempty"`
	StandardRateTax                    string                       `xml:"P_14_1,omitempty"`
	StandardRateTaxConvertedToPln      string                       `xml:"P_14_1W,omitempty"`
	ReducedRateNetSale                 string                       `xml:"P_13_2,omitempty"`
	ReducedRateTax                     string                       `xml:"P_14_2,omitempty"`
	ReducedRateTaxConvertedToPln       string                       `xml:"P_14_2W,omitempty"`
	SuperReducedRateNetSale            string                       `xml:"P_13_3,omitempty"`
	SuperReducedRateTax                string                       `xml:"P_14_3,omitempty"`
	SuperReducedRateTaxConvertedToPln  string                       `xml:"P_14_3W,omitempty"`
	TaxiRateNetSale                    string                       `xml:"P_13_4,omitempty"`
	TaxiRateTax                        string                       `xml:"P_14_4,omitempty"`
	TaxiRateTaxConvertedToPln          string                       `xml:"P_14_4W,omitempty"`
	OSSNetSale                         string                       `xml:"P_13_5,omitempty"`
	OSSTax                             string                       `xml:"P_14_5,omitempty"`
	ZeroTaxExceptIntraCommunityNetSale string                       `xml:"P_13_6_1,omitempty"`
	IntraCommunityNetSale              string                       `xml:"P_13_6_2,omitempty"`
	ExportNetSale                      string                       `xml:"P_13_6_3,omitempty"`
	TaxExemptNetSale                   string                       `xml:"P_13_7,omitempty"`
	OutsideScopeNetSale                string                       `xml:"P_13_8,omitempty"`
	ReverseChargeNetSale               string                       `xml:"P_13_9,omitempty"`
	DomesticReverseChargeNetSale       string                       `xml:"P_13_10,omitempty"`
	MarginNetSale                      string                       `xml:"P_13_11,omitempty"`
	TotalAmountDue                     string                       `xml:"P_15"`
	ExchangeRate                       string                       `xml:"KursWalutyZ,omitempty"`
	AmountBeforeCorrection             string                       `xml:"P_15ZK,omitempty"`
	Annotations                        *Annotations                 `xml:"Adnotacje"`
	InvoiceType                        string                       `xml:"RodzajFaktury"`
	CorrectionReason                   string                       `xml:"PrzyczynaKorekty,omitempty"`
	CorrectionType                     string                       `xml:"TypKorekty,omitempty"`
	CorrectedInv                       []*CorrectedInv              `xml:"DaneFaKorygowanej,omitempty"`
	PartialAdvancePayments             []*PartialAdvancePayment     `xml:"ZaliczkaCzesciowa,omitempty"`
	FP                                 int                          `xml:"FP,omitempty"`
	TP                                 int                          `xml:"TP,omitempty"`
	AdditionalDescription              []*AdditionalDescriptionLine `xml:"DodatkowyOpis,omitempty"`
	AdvanceInvoices                    []*AdvanceInvoiceRef         `xml:"FakturaZaliczkowa,omitempty"`
	ExciseTaxRefund                    int                          `xml:"ZwrotAkcyzy,omitempty"`
	Lines                              []*Line                      `xml:"FaWiersz,omitempty"` // empty for ZAL and KOR_ZAL, use Order instead
	Settlement                         *Settlement                  `xml:"Rozliczenie,omitempty"`
	Payment                            *Payment                     `xml:"Platnosc,omitempty"`
	TransactionConditions              *TransactionConditions       `xml:"WarunkiTransakcji,omitempty"`
	Order                              *Order                       `xml:"Zamowienie,omitempty"` // for ZAL and KOR_ZAL types
}

type InvoicePeriod struct {
	StartDate string `xml:"P_6_Od,omitempty"`
	EndDate   string `xml:"P_6_Do,omitempty"`
}

// AdditionalDescriptionLine defines the XML structure for KSeF additional description line (`DodatkowyOpis`)
type AdditionalDescriptionLine struct {
	LineNumber string `xml:"NrWiersza,omitempty"`
	Key        string `xml:"Klucz"`
	Value      string `xml:"Wartosc"`
}

// AdvanceInvoiceRef defines the XML structure for advance invoice reference
type AdvanceInvoiceRef struct {
	KSeFMarker           int    `xml:"NrKSeFZN,omitempty"`
	AdvanceInvoiceNo     string `xml:"NrFaZaliczkowej,omitempty"`
	KSeFAdvanceInvoiceNo string `xml:"NrKSeFFaZaliczkowej,omitempty"`
}

// PartialAdvancePayment defines the XML structure for partial advance payment (ZaliczkaCzesciowa)
type PartialAdvancePayment struct {
	PaymentDate          string `xml:"P_6Z"`
	PaymentAmount        string `xml:"P_15Z"`
	CurrencyExchangeRate string `xml:"KursWalutyZW,omitempty"`
}

// Settlement defines the XML structure for the additional charges and
// deductions block (Rozliczenie). Per the FA(3) schema, each <Obciazenia>
// and <Odliczenia> is itself a single entry (with <Kwota>/<Powod> children,
// maxOccurs=100), and <SumaObciazen>/<SumaOdliczen> are siblings of those
// entries — not nested wrappers.
type Settlement struct {
	Charges         []*ChargeOrDeduction `xml:"Obciazenia,omitempty"`
	TotalCharges    string               `xml:"SumaObciazen,omitempty"`
	Deductions      []*ChargeOrDeduction `xml:"Odliczenia,omitempty"`
	TotalDeductions string               `xml:"SumaOdliczen,omitempty"`
	AmountToPay     string               `xml:"DoZaplaty,omitempty"`
	AmountToSettle  string               `xml:"DoRozliczenia,omitempty"`
}

// ChargeOrDeduction defines the XML structure for a single charge or deduction
type ChargeOrDeduction struct {
	Amount string `xml:"Kwota"`
	Reason string `xml:"Powod"`
}

// TransactionConditions defines the XML structure for transaction conditions
type TransactionConditions struct {
	Contracts         []*Contract  `xml:"Umowy,omitempty"`
	Orders            []*OrderRef  `xml:"Zamowienia,omitempty"`
	BatchNumbers      []string     `xml:"NrPartiiTowaru,omitempty"`
	DeliveryTerms     string       `xml:"WarunkiDostawy,omitempty"`
	ContractRate      string       `xml:"KursUmowny,omitempty"`
	ContractCurrency  string       `xml:"WalutaUmowna,omitempty"`
	Transport         []*Transport `xml:"Transport,omitempty"`
	IntermediaryParty int          `xml:"PodmiotPosredniczacy,omitempty"`
}

// Contract defines the XML structure for contract reference
type Contract struct {
	Date   string `xml:"DataUmowy"`
	Number string `xml:"NrUmowy"`
}

// OrderRef defines the XML structure for order reference
type OrderRef struct {
	Date   string `xml:"DataZamowienia,omitempty"`
	Number string `xml:"NrZamowienia"`
}

// Transport defines the XML structure for transport information
type Transport struct {
	TransportType        string     `xml:"RodzajTransportu,omitempty"`
	OtherTransportType   int        `xml:"TransportInny,omitempty"`
	OtherTransportDesc   string     `xml:"OpisInnegoTransportu,omitempty"`
	Carrier              *Carrier   `xml:"Przewoznik,omitempty"`
	TransportOrderNumber string     `xml:"NrZleceniaTransportu,omitempty"`
	CargoType            string     `xml:"OpisLadunku,omitempty"`
	OtherCargoType       int        `xml:"LadunekInny,omitempty"`
	OtherCargoDesc       string     `xml:"OpisInnegoLadunku,omitempty"`
	PackagingUnit        string     `xml:"JednostkaOpakowania,omitempty"`
	TransportStartTime   string     `xml:"DataGodzRozpTransportu,omitempty"`
	TransportEndTime     string     `xml:"DataGodzZakTransportu,omitempty"`
	ShipFrom             *Address   `xml:"WysylkaZ,omitempty"`
	ShipVia              []*Address `xml:"WysylkaPrzez,omitempty"`
	ShipTo               *Address   `xml:"WysylkaDo,omitempty"`
}

// Carrier defines the XML structure for carrier information
type Carrier struct {
	IdentificationData *Buyer   `xml:"DaneIdentyfikacyjne"`
	Address            *Address `xml:"AdresPrzewoznika"`
}

// Stopka defines the XML structure for the KSeF invoice footer.
type Stopka struct {
	Informacje []*StopkaInformacje `xml:"Informacje,omitempty"`
}

// StopkaInformacje defines the XML structure for free-text footer entries.
type StopkaInformacje struct {
	StopkaFaktury string `xml:"StopkaFaktury,omitempty"`
}

// NewFavatFooter builds a Stopka from GOBL invoice notes with src == NoteSourceFooter.
func NewFavatFooter(inv *bill.Invoice) *Stopka {
	var informacje []*StopkaInformacje
	for _, note := range inv.Notes {
		if note.Src == NoteSourceFooter {
			informacje = append(informacje, &StopkaInformacje{StopkaFaktury: note.Text})
		}
	}
	if len(informacje) == 0 {
		return nil
	}
	return &Stopka{Informacje: informacje}
}

// NewFavatInv gets invoice data from GOBL invoice
func NewFavatInv(invoice *bill.Invoice) *Inv {

	inv := &Inv{
		CurrencyCode:     invoice.Currency.String(),
		IssueDate:        invoice.IssueDate.String(),
		Period:           newInvoicePeriod(invoice.Ordering),
		SequentialNumber: invoiceNumber(invoice.Series, invoice.Code),
		Annotations:      newAnnotations(invoice),
		Lines:            NewLinesForInvoice(invoice),
		Settlement:       newSettlement(invoice),
		Payment:          NewPayment(invoice.Payment, invoice.Totals),
	}

	if invoice.OperationDate != nil {
		inv.CompletionDate = invoice.OperationDate.String()
	}

	if invoice.Tax != nil && !invoice.Tax.Ext.IsZero() {
		inv.InvoiceType = invoice.Tax.Ext.Get(favat.ExtKeyInvoiceType).String()
	}

	// P_15 is "Total amount due". Per the FA(3) spec it varies by invoice type:
	//   - Settlement (ROZ/KOR_ROZ, Art. 106f sec. 3): the amount remaining
	//     after deducting prior advance invoices (Totals.Due).
	//   - Advance (ZAL/KOR_ZAL): the amount of the prepayment documented
	//     by the invoice (Totals.Due, which equals lines − previously-invoiced
	//     advances).
	//   - All other types, including standard VAT invoices that happen to
	//     have been prepaid via tracked payment.advances: the full gross
	//     total of the invoice (Totals.Payable). The prepayment is just
	//     payment-tracking information and does not reduce P_15.
	if (inv.isSettlementType() || inv.isPrepaymentType()) && invoice.Totals.Due != nil {
		inv.TotalAmountDue = invoice.Totals.Due.String()
	} else {
		inv.TotalAmountDue = invoice.Totals.Payable.String()
	}

	taxes := invoice.Totals.Taxes

	// For settlement invoices (ROZ/KOR_ROZ) with advances, KSeF expects
	// P_13_X/P_14_X to contain only the remaining amounts after advance
	// deductions — not the full order totals. We build adjusted tax totals
	// and map advance refs to FakturaZaliczkowa.
	if (inv.InvoiceType == "ROZ" || inv.InvoiceType == "KOR_ROZ") &&
		invoice.Payment != nil && len(invoice.Payment.Advances) > 0 {
		inv.mapSettlementAdvanceRefs(invoice)
		taxes = settlementTaxTotals(invoice.Totals)
	}

	// Look for exchange rate from invoice currency to PLN
	var xr *currency.ExchangeRate
	if invoice.Currency != currency.PLN {
		xr = currency.MatchExchangeRate(invoice.ExchangeRates, invoice.Currency, currency.PLN)
		if xr != nil {
			inv.ExchangeRate = xr.Amount.String()
		}
	}

	inv.setTaxRates(taxes, xr)

	if len(invoice.Notes) > 0 {
		for _, note := range invoice.Notes {
			if note.Src == NoteSourceFooter {
				continue
			}
			inv.AdditionalDescription = append(inv.AdditionalDescription, &AdditionalDescriptionLine{
				Key:   noteKey(note),
				Value: note.Text,
			})
		}
	}

	// Export line-level notes as DodatkowyOpis with NrWiersza
	for _, line := range invoice.Lines {
		for _, note := range line.Notes {
			inv.AdditionalDescription = append(inv.AdditionalDescription, &AdditionalDescriptionLine{
				LineNumber: strconv.Itoa(line.Index),
				Key:        noteKey(note),
				Value:      note.Text,
			})
		}
	}

	inv.Order = newOrder(invoice.Ordering)
	inv.TransactionConditions = newTransactionConditions(invoice.Ordering)

	if len(invoice.Preceding) > 0 {
		if invoice.Preceding[0].Reason != "" {
			inv.CorrectionReason = invoice.Preceding[0].Reason
		}
		if invoice.Preceding[0].Ext.Has(favat.ExtKeyEffectiveDate) {
			inv.CorrectionType = invoice.Preceding[0].Ext.Get(favat.ExtKeyEffectiveDate).String()
		}
		for _, prc := range invoice.Preceding {
			inv.CorrectedInv = append(inv.CorrectedInv, NewCorrectedInv(prc))
		}
	}

	return inv
}

// newTransactionConditions builds TransactionConditions from the invoice's
// ordering data, mapping Purchases to Zamowienia and Contracts to Umowy.
func newTransactionConditions(ordering *bill.Ordering) *TransactionConditions {
	if ordering == nil {
		return nil
	}

	var tc TransactionConditions

	// Add order references from purchases (Zamowienia)
	for _, ref := range ordering.Purchases {
		or := &OrderRef{
			Number: ref.Code.String(),
		}
		if ref.IssueDate != nil {
			or.Date = ref.IssueDate.String()
		}
		tc.Orders = append(tc.Orders, or)
	}

	// Add contract references (Umowy)
	for _, ref := range ordering.Contracts {
		c := &Contract{
			Number: ref.Code.String(),
		}
		if ref.IssueDate != nil {
			c.Date = ref.IssueDate.String()
		}
		tc.Contracts = append(tc.Contracts, c)
	}

	if len(tc.Orders) == 0 && len(tc.Contracts) == 0 {
		return nil
	}

	return &tc
}

func invoiceNumber(series cbc.Code, code cbc.Code) string {
	if series == "" {
		return code.String()
	}
	return fmt.Sprintf("%s-%s", series, code)
}

func invoicePricesIncludeVAT(invoice *bill.Invoice) bool {
	return invoice.Tax != nil && invoice.Tax.PricesInclude == tax.CategoryVAT
}

func (inv *Inv) setTaxRates(taxes *tax.Total, xr *currency.ExchangeRate) {
	if taxes == nil {
		return
	}
	for _, cat := range taxes.Categories {
		if cat.Code != tax.CategoryVAT {
			continue
		}

		for _, rate := range cat.Rates {
			switch rate.Ext.Get(favat.ExtKeyTaxCategory) {
			case "1": // standard rate
				inv.StandardRateNetSale = rate.Base.String()
				inv.StandardRateTax = rate.Amount.String()
				if xr != nil {
					inv.StandardRateTaxConvertedToPln = xr.Convert(rate.Amount).String()
				}
			case "2": // reduced rate
				inv.ReducedRateNetSale = rate.Base.String()
				inv.ReducedRateTax = rate.Amount.String()
				if xr != nil {
					inv.ReducedRateTaxConvertedToPln = xr.Convert(rate.Amount).String()
				}
			case "3": // super reduced rate
				inv.SuperReducedRateNetSale = rate.Base.String()
				inv.SuperReducedRateTax = rate.Amount.String()
				if xr != nil {
					inv.SuperReducedRateTaxConvertedToPln = xr.Convert(rate.Amount).String()
				}
			case "4": // taxi rate
				inv.TaxiRateNetSale = rate.Base.String()
				inv.TaxiRateTax = rate.Amount.String()
				if xr != nil {
					inv.TaxiRateTaxConvertedToPln = xr.Convert(rate.Amount).String()
				}
			case "5": // OSS rate (no PLN-converted variant in FA3 schema)
				inv.OSSNetSale = rate.Base.String()
				inv.OSSTax = rate.Amount.String()
			case "6.1": // zero tax except intra-community supply
				inv.ZeroTaxExceptIntraCommunityNetSale = rate.Base.String()
			case "6.2": // intra-community supply
				inv.IntraCommunityNetSale = rate.Base.String()
			case "6.3": // export supply
				inv.ExportNetSale = rate.Base.String()
			case "7": // tax exempt supply
				inv.TaxExemptNetSale = rate.Base.String()
			case "8": // outside scope supply
				inv.OutsideScopeNetSale = rate.Base.String()
			case "9": // reverse charge supply
				inv.ReverseChargeNetSale = rate.Base.String()
			case "10": // domestic reverse charge supply
				inv.DomesticReverseChargeNetSale = rate.Base.String()
			case "11": // margin supply
				inv.MarginNetSale = rate.Base.String()
			}
		}
	}
}

// newSettlement builds the KSeF Rozliczenie element from invoice-level
// charges and discounts. Returns nil when neither is present so the
// element is omitted from the XML output.
func newSettlement(invoice *bill.Invoice) *Settlement {
	if len(invoice.Charges) == 0 && len(invoice.Discounts) == 0 {
		return nil
	}

	s := &Settlement{}

	var totalCharges num.Amount
	for _, c := range invoice.Charges {
		s.Charges = append(s.Charges, &ChargeOrDeduction{
			Amount: c.Amount.String(),
			Reason: c.Reason,
		})
		totalCharges = totalCharges.MatchPrecision(c.Amount)
		totalCharges = totalCharges.Add(c.Amount)
	}
	if len(s.Charges) > 0 {
		s.TotalCharges = totalCharges.String()
	}

	var totalDeductions num.Amount
	for _, d := range invoice.Discounts {
		s.Deductions = append(s.Deductions, &ChargeOrDeduction{
			Amount: d.Amount.String(),
			Reason: d.Reason,
		})
		totalDeductions = totalDeductions.MatchPrecision(d.Amount)
		totalDeductions = totalDeductions.Add(d.Amount)
	}
	if len(s.Deductions) > 0 {
		s.TotalDeductions = totalDeductions.String()
	}

	return s
}

// parseSettlement maps the KSeF Rozliczenie element back to GOBL invoice-level
// charges and discounts. KSeF's <Obciazenia> and <Odliczenia> entries carry
// no VAT information, so they are mapped without taxes — the amounts flow
// into Totals.Payable via Calculate, which is what the KSeF P_15 reflects.
//
// For corrective invoices (KOR / KOR_ZAL / KOR_ROZ), the KSeF XML carries
// negative monetary amounts (TKwotowy permits sign). parseLines and the
// totalDue passed into AdjustRounding both flip credit-note amounts back
// to GOBL's positive convention; settlement amounts must follow the same
// rule so Calculate reconciles cleanly against the inverted P_15.
func (inv *Inv) parseSettlement(goblInv *bill.Invoice) error {
	if inv.Settlement == nil {
		return nil
	}

	isCreditNote := goblInv.Type == bill.InvoiceTypeCreditNote

	for _, c := range inv.Settlement.Charges {
		amount, err := parseAmount(c.Amount)
		if err != nil {
			return fmt.Errorf("parsing settlement charge amount: %w", err)
		}
		if isCreditNote {
			amount = amount.Invert()
		}
		goblInv.Charges = append(goblInv.Charges, &bill.Charge{
			Amount: amount,
			Reason: c.Reason,
		})
	}

	for _, d := range inv.Settlement.Deductions {
		amount, err := parseAmount(d.Amount)
		if err != nil {
			return fmt.Errorf("parsing settlement deduction amount: %w", err)
		}
		if isCreditNote {
			amount = amount.Invert()
		}
		goblInv.Discounts = append(goblInv.Discounts, &bill.Discount{
			Amount: amount,
			Reason: d.Reason,
		})
	}

	return nil
}

// mapSettlementAdvanceRefs maps GOBL advance payment refs to KSeF
// FakturaZaliczkowa elements for settlement invoices.
func (inv *Inv) mapSettlementAdvanceRefs(invoice *bill.Invoice) {
	for _, adv := range invoice.Payment.Advances {
		if adv.Ref != "" {
			inv.AdvanceInvoices = append(inv.AdvanceInvoices, &AdvanceInvoiceRef{
				KSeFAdvanceInvoiceNo: adv.Ref,
			})
		}
	}
}

// settlementTaxTotals returns a copy of the invoice's tax totals with Base and
// Amount prorated by Due/Payable. This gives the remaining tax amounts after
// advance deductions, as required by KSeF for settlement invoices (ROZ/KOR_ROZ).
// Returns the original totals unchanged when no proration is needed.
func settlementTaxTotals(totals *bill.Totals) *tax.Total {
	if totals.Taxes == nil || totals.Due == nil || totals.Payable.IsZero() {
		return totals.Taxes
	}
	due := *totals.Due
	payable := totals.Payable
	if due.Equals(payable) {
		return totals.Taxes
	}

	prorate := func(full num.Amount) num.Amount {
		if full.IsZero() {
			return full
		}
		return full.Multiply(due).Divide(payable)
	}

	adjusted := &tax.Total{}
	for _, cat := range totals.Taxes.Categories {
		ac := &tax.CategoryTotal{
			Code: cat.Code,
		}
		for _, rate := range cat.Rates {
			ac.Rates = append(ac.Rates, &tax.RateTotal{
				Key:     rate.Key,
				Ext:     rate.Ext,
				Percent: rate.Percent,
				Base:    prorate(rate.Base),
				Amount:  prorate(rate.Amount),
			})
		}
		adjusted.Categories = append(adjusted.Categories, ac)
	}
	return adjusted
}

// parseInvoiceData converts KSEF invoice data to GOBL invoice fields
func (inv *Inv) parseInvoiceData(goblInv *bill.Invoice) error {
	// Parse invoice type and tags
	invType, tags := parseInvoiceType(inv.InvoiceType)
	goblInv.Type = invType
	if len(tags) > 0 {
		goblInv.Tags = tax.Tags{List: tags}
	}

	// Parse issue date
	if inv.IssueDate != "" {
		date, err := parseDate(inv.IssueDate)
		if err != nil {
			return fmt.Errorf("parsing issue date: %w", err)
		}
		goblInv.IssueDate = date
	}

	goblInv.Code = cbc.Code(inv.SequentialNumber)

	// Parse ordering period
	if inv.Period != nil {
		goblInv.Ordering = &bill.Ordering{
			Period: &cal.Period{},
		}
		if inv.Period.StartDate != "" {
			start, err := parseDate(inv.Period.StartDate)
			if err != nil {
				return fmt.Errorf("parsing period start date: %w", err)
			}
			goblInv.Ordering.Period.Start = &start
		}
		if inv.Period.EndDate != "" {
			end, err := parseDate(inv.Period.EndDate)
			if err != nil {
				return fmt.Errorf("parsing period end date: %w", err)
			}
			goblInv.Ordering.Period.End = &end
		}
	}

	// Map KursWalutyZ to ExchangeRates for foreign currency invoices
	if inv.ExchangeRate != "" && goblInv.Currency != currency.PLN {
		rate, err := num.AmountFromString(inv.ExchangeRate)
		if err != nil {
			return fmt.Errorf("parsing exchange rate: %w", err)
		}
		goblInv.ExchangeRates = append(goblInv.ExchangeRates, &currency.ExchangeRate{
			From:   goblInv.Currency,
			To:     currency.PLN,
			Amount: rate,
		})
	}

	// Parse annotations to tax extensions
	inv.parseAnnotations(goblInv)

	// Always set rounding to currency as per EN16931 standard
	if goblInv.Tax == nil {
		goblInv.Tax = &bill.Tax{}
	}
	goblInv.Tax.Rounding = tax.RoundingRuleCurrency

	// NOTE: DodatkowyOpis (additional descriptions) are parsed after lines
	// in parseAdditionalDescriptions, so that line-level notes (NrWiersza)
	// can be attached to the correct GOBL lines.

	// Parse corrected invoices (preceding documents for credit notes)
	if len(inv.CorrectedInv) > 0 {
		goblInv.Preceding = []*org.DocumentRef{}
		for _, corr := range inv.CorrectedInv {
			preceding := &org.DocumentRef{}

			if corr.SequentialNumber != "" {
				preceding.Code = cbc.Code(corr.SequentialNumber)
			}
			if corr.IssueDate != "" {
				date, err := parseDate(corr.IssueDate)
				if err != nil {
					return fmt.Errorf("parsing corrected invoice date: %w", err)
				}
				preceding.IssueDate = &date
			}
			if inv.CorrectionReason != "" {
				preceding.Reason = inv.CorrectionReason
			}
			if inv.CorrectionType != "" {
				preceding.Ext = tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyEffectiveDate: cbc.Code(inv.CorrectionType),
				})
			}
			if corr.KsefNumberPresent == 1 && corr.KsefNumber != "" {
				preceding.Stamps = []*head.Stamp{
					{
						Provider: favat.StampKSeFNumber,
						Value:    corr.KsefNumber,
					},
				}
			}

			goblInv.Preceding = append(goblInv.Preceding, preceding)
		}
	}

	return nil
}

// buildNote creates an org.Note from a KSeF additional description line.
func buildNote(desc *AdditionalDescriptionLine) *org.Note {
	note := &org.Note{}
	if validCodeRe.MatchString(desc.Key) {
		note.Code = cbc.Code(desc.Key)
		note.Text = desc.Value
	} else {
		// Key contains characters not valid for cbc.Code;
		// combine key and value into the text field.
		note.Text = desc.Key + ": " + desc.Value
	}
	return note
}

// parseAdditionalDescriptions converts KSeF DodatkowyOpis entries to GOBL notes.
// Entries with NrWiersza are attached to the corresponding line's notes;
// entries without NrWiersza (or referencing a non-existent line) become
// invoice-level notes. Must be called after parseLines so that GOBL lines
// are already populated.
func (inv *Inv) parseAdditionalDescriptions(goblInv *bill.Invoice) {
	if len(inv.AdditionalDescription) == 0 {
		return
	}

	// Build map from KSeF line number → GOBL line
	lineMap := make(map[int]*bill.Line)
	for i, ksefLine := range inv.Lines {
		if i < len(goblInv.Lines) {
			lineMap[ksefLine.LineNumber] = goblInv.Lines[i]
		}
	}

	for _, desc := range inv.AdditionalDescription {
		note := buildNote(desc)

		if desc.LineNumber != "" {
			lineNum, err := strconv.Atoi(strings.TrimSpace(desc.LineNumber))
			if err == nil {
				if line, ok := lineMap[lineNum]; ok {
					line.Notes = append(line.Notes, note)
					continue
				}
			}
			// Fall through to invoice-level if line not found
		}

		goblInv.Notes = append(goblInv.Notes, note)
	}
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (cal.Date, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return cal.Date{}, fmt.Errorf("invalid date format: %w", err)
	}
	return cal.DateOf(t), nil
}

// parseInvoiceType converts KSEF invoice type to GOBL type and tags.
func parseInvoiceType(invType string) (cbc.Key, []cbc.Key) {
	tags := []cbc.Key{}

	switch invType {
	case "VAT":
		return bill.InvoiceTypeStandard, tags
	case "ZAL":
		tags = append(tags, tax.TagPartial)
		return bill.InvoiceTypeStandard, tags
	case "ROZ":
		tags = append(tags, favat.TagSettlement)
		return bill.InvoiceTypeStandard, tags
	case "UPR":
		tags = append(tags, tax.TagSimplified)
		return bill.InvoiceTypeStandard, tags
	case "KOR":
		return bill.InvoiceTypeCreditNote, tags
	case "KOR_ZAL":
		tags = append(tags, tax.TagPartial)
		return bill.InvoiceTypeCreditNote, tags
	case "KOR_ROZ":
		tags = append(tags, favat.TagSettlement)
		return bill.InvoiceTypeCreditNote, tags
	default:
		return bill.InvoiceTypeStandard, tags
	}
}

// isPrepaymentType returns true for advance invoice types (ZAL or KOR_ZAL).
func (inv *Inv) isPrepaymentType() bool {
	return inv.InvoiceType == "ZAL" || inv.InvoiceType == "KOR_ZAL"
}

// isSettlementType returns true for settlement invoice types (ROZ or KOR_ROZ).
func (inv *Inv) isSettlementType() bool {
	return inv.InvoiceType == "ROZ" || inv.InvoiceType == "KOR_ROZ"
}

// parseLines converts KSEF lines to GOBL lines.
func (inv *Inv) parseLines(goblInv *bill.Invoice) error {
	if len(inv.Lines) == 0 {
		if inv.isPrepaymentType() {
			// Prepayment invoices without FaWiersz lines use bypass mode:
			// totals are set directly from tax summary fields (P_13_X/P_14_X).
			goblInv.Tags.List = append(goblInv.Tags.List, tax.TagBypass)
		}
		return nil
	}

	goblInv.Lines = make([]*bill.Line, 0, len(inv.Lines))

	// NOTE: Mixed P_9A/P_9B invoices are not supported.
	//
	// KSeF allows P_9A (net unit price, art. 106e(2)-(3)) and P_9B (gross unit
	// price, art. 106e(7)-(8)) per line, with no schema-level mutual exclusivity.
	// In practice the Polish VAT act frames this as an invoice-wide choice, but a
	// document with some lines using P_9A and others using P_9B is technically
	// valid XML.
	//
	// The current approach sets PricesInclude = VAT at the invoice level when any
	// line uses gross pricing (P_9B without P_9A). This works for pure-gross
	// invoices but would produce wrong totals on a mixed invoice: GOBL would strip
	// VAT from the already-net prices on P_9A lines.
	//
	// If a mixed invoice is ever encountered, the fix is to drop PricesInclude and
	// instead convert gross→net explicitly inside Line.ToGOBL() using the line's
	// own VAT rate (P_12): net = gross / (1 + rate). This works uniformly for
	// pure-net, pure-gross, and mixed invoices without any invoice-level flag.
	isCreditNote := goblInv.Type == bill.InvoiceTypeCreditNote
	hasGrossPricing := false
	for _, ksefLine := range inv.Lines {
		line, err := ksefLine.ToGOBL()
		if err != nil {
			return fmt.Errorf("parsing line %d: %w", ksefLine.LineNumber, err)
		}

		// For credit notes: invert non-StanPrzed lines to positive GOBL convention.
		// StanPrzed lines represent the before-correction state and stay as-is
		// (they are the amounts being credited back to the customer).
		if isCreditNote {
			if ksefLine.BeforeCorrectionMarker == 1 {
				line.Notes = append(line.Notes, &org.Note{
					Text: "Before correction (stan przed korektą)",
				})
			} else {
				line.Quantity = line.Quantity.Invert()
				// P_10 is always a positive total discount amount in KSeF,
				// no inversion needed — GOBL also expects positive discounts.
			}
		}

		goblInv.Lines = append(goblInv.Lines, line)
		if ksefLine.NetUnitPrice == "" && ksefLine.GrossUnitPrice != "" {
			hasGrossPricing = true
		}
		// Also detect gross pricing when no unit prices exist but gross total is present
		if ksefLine.NetUnitPrice == "" && ksefLine.GrossUnitPrice == "" &&
			ksefLine.NetPriceTotal == "" && ksefLine.GrossPriceTotal != "" {
			hasGrossPricing = true
		}
	}

	if hasGrossPricing {
		if goblInv.Tax == nil {
			goblInv.Tax = &bill.Tax{}
		}
		goblInv.Tax.PricesInclude = tax.CategoryVAT
	}

	return nil
}

// parsePrepaymentTotals builds bill.Totals directly from the invoice-level
// tax summary fields (P_13_X / P_14_X / P_15). The invoice must have the
// bypass tag set so that Calculate() does not overwrite these totals.
// The resulting invoice will not pass GOBL validation (no lines), which
// is acceptable for this edge case.
func (inv *Inv) parsePrepaymentTotals(goblInv *bill.Invoice) error {
	// Each entry maps a P_13_X / P_14_X pair to a tax category.
	// Well-known percentages are set when the rate is unambiguous;
	// otherwise only the amounts are included.
	type taxEntry struct {
		net      string
		tax      string
		category cbc.Code
		key      cbc.Key
		percent  *num.Percentage // nil when rate is unknown or exempt
	}

	pct23 := num.MakePercentage(230, 3)
	pct8 := num.MakePercentage(80, 3)
	pct5 := num.MakePercentage(50, 3)
	pct0 := num.MakePercentage(0, 3)

	entries := []taxEntry{
		{inv.StandardRateNetSale, inv.StandardRateTax, "1", tax.KeyStandard, &pct23},
		{inv.ReducedRateNetSale, inv.ReducedRateTax, "2", tax.KeyStandard, &pct8},
		{inv.SuperReducedRateNetSale, inv.SuperReducedRateTax, "3", tax.KeyStandard, &pct5},
		{inv.TaxiRateNetSale, inv.TaxiRateTax, "4", tax.KeyStandard, nil},
		{inv.OSSNetSale, inv.OSSTax, "5", tax.KeyStandard, nil},
		{inv.ZeroTaxExceptIntraCommunityNetSale, "", "6.1", tax.KeyZero, &pct0},
		{inv.IntraCommunityNetSale, "", "6.2", tax.KeyIntraCommunity, &pct0},
		{inv.ExportNetSale, "", "6.3", tax.KeyExport, &pct0},
		{inv.TaxExemptNetSale, "", "7", tax.KeyExempt, nil},
		{inv.OutsideScopeNetSale, "", "8", tax.KeyOutsideScope, nil},
		{inv.ReverseChargeNetSale, "", "9", tax.KeyReverseCharge, nil},
		{inv.DomesticReverseChargeNetSale, "", "10", tax.KeyReverseCharge, nil},
	}

	// For credit notes (e.g. KOR_ZAL), KSeF P_13/P_14/P_15 values are negative.
	// GOBL expects positive totals, so we invert them. Invoice.Invert() cannot
	// be used with the bypass tag, so we invert each amount individually.
	isCreditNote := goblInv.Type == bill.InvoiceTypeCreditNote

	var rates []*tax.RateTotal
	var netSum, taxSum num.Amount

	for _, e := range entries {
		if e.net == "" {
			continue
		}

		netAmt, err := parseAmount(e.net)
		if err != nil {
			return fmt.Errorf("parsing prepayment net for category %s: %w", e.category, err)
		}
		if isCreditNote {
			netAmt = netAmt.Invert()
		}

		rt := &tax.RateTotal{
			Key:     e.key,
			Base:    netAmt,
			Percent: e.percent,
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				favat.ExtKeyTaxCategory: e.category,
			}),
		}

		if e.tax != "" {
			taxAmt, err := parseAmount(e.tax)
			if err != nil {
				return fmt.Errorf("parsing prepayment tax for category %s: %w", e.category, err)
			}
			if isCreditNote {
				taxAmt = taxAmt.Invert()
			}
			rt.Amount = taxAmt
			taxSum = taxSum.MatchPrecision(taxAmt)
			taxSum = taxSum.Add(taxAmt)
		}

		netSum = netSum.MatchPrecision(netAmt)
		netSum = netSum.Add(netAmt)
		rates = append(rates, rt)
	}

	totals := &bill.Totals{
		Sum:   netSum,
		Total: netSum,
		Tax:   taxSum,
	}

	if len(rates) > 0 {
		totals.Taxes = &tax.Total{
			Categories: []*tax.CategoryTotal{
				{
					Code:   tax.CategoryVAT,
					Rates:  rates,
					Amount: taxSum,
				},
			},
			Sum: taxSum,
		}
	}

	totals.TotalWithTax = netSum.Add(taxSum)

	if inv.TotalAmountDue != "" {
		payable, err := parseAmount(inv.TotalAmountDue)
		if err != nil {
			return fmt.Errorf("parsing total amount due: %w", err)
		}
		if isCreditNote {
			payable = payable.Invert()
		}
		totals.Payable = payable
	} else {
		totals.Payable = totals.TotalWithTax
	}

	// Sum advances from payment details
	if goblInv.Payment != nil && len(goblInv.Payment.Advances) > 0 {
		var advanceSum num.Amount
		for _, adv := range goblInv.Payment.Advances {
			advanceSum = advanceSum.MatchPrecision(adv.Amount)
			advanceSum = advanceSum.Add(adv.Amount)
		}
		totals.Advances = &advanceSum
		due := totals.Payable.Subtract(advanceSum)
		totals.Due = &due
	}

	goblInv.Totals = totals
	return nil
}

func noteKey(note *org.Note) string {
	if note.Key != "" {
		return note.Key.String()
	}
	return note.Code.String()
}
