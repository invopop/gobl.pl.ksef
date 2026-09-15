package ksef

import (
	"fmt"

	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
)

// Payment defines the XML structure for KSeF payment
type Payment struct {
	PaidMarker             string            `xml:"Zaplacono,omitempty"`
	PaymentDate            string            `xml:"DataZaplaty,omitempty"`
	PartiallyPaidMarker    string            `xml:"ZnacznikZaplatyCzesciowej,omitempty"`
	AdvancePayments        []*AdvancePayment `xml:"ZaplataCzesciowa,omitempty"`
	DueDates               []*DueDate        `xml:"TerminPlatnosci,omitempty"`
	PaymentMean            string            `xml:"FormaPlatnosci,omitempty"` // enum: 1 = cash, 2 = card etc. (see KSeF documentation)
	OtherPaymentMeanMarker string            `xml:"PlatnoscInna,omitempty"`
	OtherPaymentMean       string            `xml:"OpisPlatnosci,omitempty"`
	BankAccounts           []*BankAccount    `xml:"RachunekBankowy,omitempty"`
	FactorBankAccounts     []*BankAccount    `xml:"RachunekBankowyFaktora,omitempty"`
	Discount               *Discount         `xml:"Skonto,omitempty"`
	PaymentLink            string            `xml:"LinkDoPlatnosci,omitempty"`
	KSeFPaymentID          string            `xml:"IPKSeF,omitempty"`
}

// AdvancePayment defines the XML structure for KSeF advance payments
type AdvancePayment struct {
	PaymentAmount          string `xml:"KwotaZaplatyCzesciowej,omitempty"`
	PaymentDate            string `xml:"DataZaplatyCzesciowej,omitempty"`
	PaymentMean            string `xml:"FormaPlatnosci,omitempty"`
	OtherPaymentMeanMarker int    `xml:"PlatnoscInna,omitempty"`
	OtherPaymentMean       string `xml:"OpisPlatnosci,omitempty"`
}

// DueDate defines the XML structure for KSeF due date
type DueDate struct {
	Date            string           `xml:"Termin,omitempty"`
	TermDescription *TermDescription `xml:"TerminOpis,omitempty"`
}

// TermDescription defines alternative payment term description
type TermDescription struct {
	Quantity      int    `xml:"Ilosc"`
	Unit          string `xml:"Jednostka"`
	StartingEvent string `xml:"ZdarzeniePoczatkowe"`
}

// BankAccount defines the XML structure for KSeF bank accounts
type BankAccount struct {
	AccountNumber         string `xml:"NrRB"`
	SWIFT                 string `xml:"SWIFT,omitempty"`
	BankSelfAccountMarker int    `xml:"RachunekWlasnyBanku,omitempty"` // enum - 1,2,3, not sure what exactly they mean
	BankName              string `xml:"NazwaBanku,omitempty"`
	AccountDescription    string `xml:"OpisRachunku,omitempty"`
}

// Discount defines the XML structure for KSeF early payment discount
type Discount struct {
	Conditions string `xml:"WarunkiSkonta,omitempty"`
	Amount     string `xml:"WysokoscSkonta,omitempty"`
}

// NewPayment gets payment data from GOBL invoice
func NewPayment(pay *bill.PaymentDetails, totals *bill.Totals) *Payment {
	if pay == nil {
		return nil
	}

	var payment = &Payment{
		DueDates:        []*DueDate{},
		AdvancePayments: []*AdvancePayment{},
	}

	if instructions := pay.Instructions; instructions != nil {
		paymentMeansCode := instructions.Ext.Get(favat.ExtKeyPaymentMeans).String()

		if paymentMeansCode == "" && instructions.Key != "" {
			payment.OtherPaymentMeanMarker = "1"
			payment.OtherPaymentMean = instructions.Key.String()
		} else if paymentMeansCode != "" {
			payment.PaymentMean = paymentMeansCode
		}

		payment.BankAccounts = []*BankAccount{}
		payment.FactorBankAccounts = []*BankAccount{}

		for _, account := range instructions.CreditTransfer {
			accountNumber := account.IBAN
			if accountNumber == "" {
				accountNumber = account.Number
			}
			payment.BankAccounts = append(payment.BankAccounts, &BankAccount{
				AccountNumber: accountNumber.String(),
				SWIFT:         account.BIC.String(),
				BankName:      account.Name,
			})
		}
	}

	if terms := pay.Terms; terms != nil {
		for _, dueDate := range pay.Terms.DueDates {
			payment.DueDates = append(payment.DueDates, &DueDate{
				Date: dueDate.Date.String(),
			})
		}
	}

	// According to FA_VAT v3 schema:
	// If an invoice is paid in full in one payment, PaidMarker should be "1"
	// Otherwise, set PartiallyPaidMarker with the following values:
	// 1 = invoice paid partially
	// 2 = paid in full after partial payments, and the last payment is the final one
	// If the invoice is not paid at all, do not add PaidMarker or PartiallyPaidMarker.

	if advances := pay.Advances; advances != nil {
		if len(advances) == 1 && totals.Due.IsZero() {
			// Invoice already paid in full in one payment
			payment.PaidMarker = "1"
			if advances[0].Date != nil {
				payment.PaymentDate = advances[0].Date.String()
			}
			if code := advances[0].Ext.Get(favat.ExtKeyPaymentMeans).String(); code != "" {
				payment.PaymentMean = code
			}
		} else {
			if totals.Due.IsZero() {
				// Invoice already paid in full in multiple payments
				payment.PartiallyPaidMarker = "2"
			}
			if !totals.Due.IsZero() && len(advances) > 0 {
				// Invoice paid partially
				payment.PartiallyPaidMarker = "1"
			}
			// Otherwise, not paid at all - no markers needed

			for _, advance := range advances {
				advancePayment := &AdvancePayment{
					PaymentAmount: advance.Amount.String(),
					PaymentDate:   advance.Date.String(),
				}

				if paymentMeansCode := advance.Ext.Get(favat.ExtKeyPaymentMeans).String(); paymentMeansCode != "" {
					advancePayment.PaymentMean = paymentMeansCode
				}
				payment.AdvancePayments = append(payment.AdvancePayments, advancePayment)
			}
		}
	}

	return payment
}

// parsePayment converts KSEF payment data to GOBL payment.
func (inv *Inv) parsePayment(goblInv *bill.Invoice) error {
	if inv.Payment == nil {
		return nil
	}

	payment := &bill.PaymentDetails{}

	// Parse payment instructions
	if inv.Payment.PaymentMean != "" || len(inv.Payment.BankAccounts) > 0 {
		payment.Instructions = &pay.Instructions{
			Ext: tax.MakeExtensions(),
		}

		// Parse payment means
		if inv.Payment.PaymentMean != "" {
			payment.Instructions.Key = ParsePaymentMeansCode(inv.Payment.PaymentMean)
			payment.Instructions.Ext = payment.Instructions.Ext.Set(favat.ExtKeyPaymentMeans, cbc.Code(inv.Payment.PaymentMean))
		} else if inv.Payment.OtherPaymentMeanMarker == "1" {
			payment.Instructions.Key = cbc.Key(inv.Payment.OtherPaymentMean)
		} else if len(inv.Payment.BankAccounts) > 0 {
			// When bank accounts are present without an explicit payment method,
			// default to credit transfer.
			payment.Instructions.Key = pay.MeansKeyCreditTransfer
		}

		// Parse bank accounts
		if len(inv.Payment.BankAccounts) > 0 {
			payment.Instructions.CreditTransfer = make([]*pay.CreditTransfer, 0, len(inv.Payment.BankAccounts))
			for _, account := range inv.Payment.BankAccounts {
				ct := &pay.CreditTransfer{
					Number: cbc.Code(account.AccountNumber),
					Name:   account.BankName,
				}

				if account.SWIFT != "" {
					ct.BIC = cbc.Code(account.SWIFT)
				}
				payment.Instructions.CreditTransfer = append(payment.Instructions.CreditTransfer, ct)
			}
		}
	}

	// Parse payment terms (due dates).
	// Both Termin (explicit date) and TerminOpis (relative description) can
	// coexist in the same TerminPlatnosci element.
	if len(inv.Payment.DueDates) > 0 {
		lastDueDate := inv.Payment.DueDates[len(inv.Payment.DueDates)-1]

		var terms pay.Terms

		if lastDueDate.Date != "" {
			termDate, err := parseDate(lastDueDate.Date)
			if err != nil {
				return fmt.Errorf("parsing due date: %w", err)
			}
			terms.DueDates = []*pay.DueDate{{Date: &termDate, Percent: num.NewPercentage(100, 2)}}
		}

		if td := lastDueDate.TermDescription; td != nil {
			terms.Notes = fmt.Sprintf("%d %s %s", td.Quantity, td.Unit, td.StartingEvent)
		}

		if terms.DueDates != nil || terms.Notes != "" {
			payment.Terms = &terms
		}
	}

	// For credit notes, KSeF amounts are negative but GOBL expects positive values.
	isCreditNote := goblInv.Type == bill.InvoiceTypeCreditNote

	// Handle paid in full (Zaplacono=1) with no partial advance payments
	if inv.Payment.PaidMarker == "1" && len(inv.Payment.AdvancePayments) == 0 {
		amt, err := parseAmount(inv.TotalAmountDue)
		if err != nil {
			return fmt.Errorf("parsing total amount for advance: %w", err)
		}
		if isCreditNote {
			amt = amt.Invert()
		}
		advance := &pay.Record{
			Description: "Advance payment",
			Amount:      amt,
		}
		if inv.Payment.PaymentDate != "" {
			date, err := parseDate(inv.Payment.PaymentDate)
			if err != nil {
				return fmt.Errorf("parsing advance payment date: %w", err)
			}
			advance.Date = &date
		}
		if inv.Payment.PaymentMean != "" {
			advance.Key = ParsePaymentMeansCode(inv.Payment.PaymentMean)
			advance.Ext = tax.ExtensionsOf(cbc.CodeMap{
				favat.ExtKeyPaymentMeans: cbc.Code(inv.Payment.PaymentMean),
			})
		}
		payment.Advances = []*pay.Record{advance}
	}

	// Parse advance payments
	if len(inv.Payment.AdvancePayments) > 0 {
		payment.Advances = make([]*pay.Record, 0, len(inv.Payment.AdvancePayments))
		for _, adv := range inv.Payment.AdvancePayments {
			advance := &pay.Record{
				Description: "Advance payment", // GOBL requires a description
				Ext:         tax.MakeExtensions(),
			}
			if adv.PaymentAmount != "" {
				amt, err := parseAmount(adv.PaymentAmount)
				if err != nil {
					return fmt.Errorf("parsing advance amount: %w", err)
				}
				if isCreditNote {
					amt = amt.Invert()
				}
				advance.Amount = amt
			}
			if adv.PaymentDate != "" {
				date, err := parseDate(adv.PaymentDate)
				if err != nil {
					return fmt.Errorf("parsing advance date: %w", err)
				}
				advance.Date = &date
			}
			if adv.PaymentMean != "" {
				advance.Ext = advance.Ext.Set(favat.ExtKeyPaymentMeans, cbc.Code(adv.PaymentMean))
			}
			payment.Advances = append(payment.Advances, advance)
		}
	}

	if payment.Instructions != nil || payment.Terms != nil || payment.Advances != nil {
		goblInv.Payment = payment
	}

	return nil
}

// deriveSettlementAdvances reconstructs advance payment data for settlement
// invoices (ROZ/KOR_ROZ) that reference advance invoices (FakturaZaliczkowa)
// but lack explicit ZaplataCzesciowa entries. The advance amount is derived
// from the difference between the calculated payable (from lines) and P_15
// (the remaining amount due).
func (inv *Inv) deriveSettlementAdvances(goblInv *bill.Invoice, totalDueStr string) error {
	// Only act when: settlement type + has advance invoice refs + no existing advances
	if !inv.isSettlementType() || len(inv.AdvanceInvoices) == 0 {
		return nil
	}
	if goblInv.Payment != nil && len(goblInv.Payment.Advances) > 0 {
		return nil
	}

	// Calculate to get Payable from lines
	if err := goblInv.Calculate(); err != nil {
		return fmt.Errorf("calculating totals for settlement advance derivation: %w", err)
	}

	// Parse P_15 (the remaining amount due)
	p15, err := parseAmount(totalDueStr)
	if err != nil {
		return fmt.Errorf("parsing P_15 for settlement advance derivation: %w", err)
	}

	// For credit notes, P_15 is negative in KSeF but GOBL uses positive amounts
	if goblInv.Type == bill.InvoiceTypeCreditNote {
		p15 = p15.Invert()
	}

	payable := goblInv.Totals.Payable
	advanceAmt := payable.Subtract(p15)

	if advanceAmt.IsZero() {
		return nil
	}

	if goblInv.Payment == nil {
		goblInv.Payment = &bill.PaymentDetails{}
	}

	// Create a separate advance for each advance invoice reference.
	// The first advance carries the total amount; subsequent ones use zero.
	for i, ref := range inv.AdvanceInvoices {
		if ref.KSeFAdvanceInvoiceNo == "" {
			continue
		}
		amt := num.MakeAmount(0, 2)
		if i == 0 {
			amt = advanceAmt
		}
		goblInv.Payment.Advances = append(goblInv.Payment.Advances, &pay.Record{
			Description: "Payment " + ref.KSeFAdvanceInvoiceNo,
			Amount:      amt,
			Ref:         ref.KSeFAdvanceInvoiceNo,
		})
	}

	return nil
}

// ParsePaymentMeansCode converts KSEF payment means code to GOBL payment key.
func ParsePaymentMeansCode(code string) cbc.Key {
	switch code {
	case "1":
		return pay.MeansKeyCash
	case "2":
		return pay.MeansKeyCard
	case "3":
		return pay.MeansKeyOther.With(favat.MeansKeyVoucher)
	case "4":
		return pay.MeansKeyCheque
	case "5":
		return pay.MeansKeyOther.With(pay.MeansKeyCredit)
	case "6":
		return pay.MeansKeyCreditTransfer
	case "7":
		return pay.MeansKeyOnline
	default:
		return pay.MeansKeyAny
	}
}
