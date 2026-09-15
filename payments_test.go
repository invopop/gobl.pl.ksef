package ksef_test

import (
	"testing"
	"time"

	ksef "github.com/invopop/gobl.ksef"
	"github.com/invopop/gobl/addons/pl/favat"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPayment(t *testing.T) {
	t.Run("should return nil when no payment data passed", func(t *testing.T) {
		pay := ksef.NewPayment(nil, nil)
		assert.Nil(t, pay)
	})

	t.Run("should return payment if there are payment instructions", func(t *testing.T) {
		payment := &bill.PaymentDetails{
			Instructions: &pay.Instructions{},
		}
		totals := &bill.Totals{}

		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "",
			PaymentDate:            "",
			PartiallyPaidMarker:    "",
			AdvancePayments:        []*ksef.AdvancePayment{},
			DueDates:               []*ksef.DueDate{},
			PaymentMean:            "",
			OtherPaymentMeanMarker: "",
			OtherPaymentMean:       "",
			BankAccounts:           []*ksef.BankAccount{},
			FactorBankAccounts:     []*ksef.BankAccount{},
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})

	t.Run("should return set payment method from payment instructions extension", func(t *testing.T) {
		payment := &bill.PaymentDetails{
			Instructions: &pay.Instructions{
				Key: "credit-transfer",
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyPaymentMeans: "6", // credit transfer code
				}),
			},
		}
		totals := &bill.Totals{}
		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "",
			PaymentDate:            "",
			PartiallyPaidMarker:    "",
			AdvancePayments:        []*ksef.AdvancePayment{},
			DueDates:               []*ksef.DueDate{},
			PaymentMean:            "6", // from extension
			OtherPaymentMeanMarker: "",
			OtherPaymentMean:       "",
			BankAccounts:           []*ksef.BankAccount{},
			FactorBankAccounts:     []*ksef.BankAccount{},
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})

	t.Run("should use other payment marker when key without extension", func(t *testing.T) {
		payment := &bill.PaymentDetails{
			Instructions: &pay.Instructions{
				Key: "credit-transfer",
			},
		}
		totals := &bill.Totals{}
		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "",
			PaymentDate:            "",
			PartiallyPaidMarker:    "",
			AdvancePayments:        []*ksef.AdvancePayment{},
			DueDates:               []*ksef.DueDate{},
			PaymentMean:            "",
			OtherPaymentMeanMarker: "1",
			OtherPaymentMean:       "credit-transfer",
			BankAccounts:           []*ksef.BankAccount{},
			FactorBankAccounts:     []*ksef.BankAccount{},
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})

	t.Run("should populate bank accounts from credit transfer instructions", func(t *testing.T) {
		payment := &bill.PaymentDetails{
			Instructions: &pay.Instructions{
				Key: "credit-transfer",
				CreditTransfer: []*pay.CreditTransfer{
					{
						Number: "12345",
						BIC:    "BICA",
						Name:   "Bank Alpha",
					},
					{
						Number: "67890",
						BIC:    "BICB",
						Name:   "Bank Beta",
					},
				},
			},
		}
		totals := &bill.Totals{}
		pay := ksef.NewPayment(payment, totals)

		expected := []*ksef.BankAccount{
			{
				AccountNumber: "12345",
				SWIFT:         "BICA",
				BankName:      "Bank Alpha",
			},
			{
				AccountNumber: "67890",
				SWIFT:         "BICB",
				BankName:      "Bank Beta",
			},
		}

		assert.Equal(t, expected, pay.BankAccounts)
	})

	t.Run("should set payment terms", func(t *testing.T) {
		x := time.Date(2023, time.July, 28, 0, 0, 0, 0, time.UTC)
		d := cal.DateOf(x)
		num, err := num.AmountFromString("245.890")
		require.NoError(t, err)

		payment := &bill.PaymentDetails{
			Terms: &pay.Terms{
				DueDates: []*pay.DueDate{{Date: &d, Amount: &num}},
			},
		}
		totals := &bill.Totals{}
		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "",
			PaymentDate:            "",
			PartiallyPaidMarker:    "",
			AdvancePayments:        []*ksef.AdvancePayment{},
			DueDates:               []*ksef.DueDate{{Date: d.String()}},
			PaymentMean:            "",
			OtherPaymentMeanMarker: "",
			OtherPaymentMean:       "",
			BankAccounts:           []*ksef.BankAccount(nil),
			FactorBankAccounts:     []*ksef.BankAccount(nil),
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})

	t.Run("advances should set paid marker and date", func(t *testing.T) {
		// Fully paid in advance
		x := time.Date(2023, time.July, 28, 0, 0, 0, 0, time.UTC)
		d := cal.DateOf(x)
		firstNum, err := num.AmountFromString("245.890")
		require.NoError(t, err)
		zero, err := num.AmountFromString("0")
		require.NoError(t, err)

		payment := &bill.PaymentDetails{
			Advances: []*pay.Record{{Date: &d, Amount: firstNum}},
		}
		totals := &bill.Totals{
			Due:      &zero,
			Advances: &firstNum,
		}
		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "1",
			PaymentDate:            d.String(),
			PartiallyPaidMarker:    "",
			AdvancePayments:        []*ksef.AdvancePayment{},
			DueDates:               []*ksef.DueDate{},
			PaymentMean:            "",
			OtherPaymentMeanMarker: "",
			OtherPaymentMean:       "",
			BankAccounts:           []*ksef.BankAccount(nil),
			FactorBankAccounts:     []*ksef.BankAccount(nil),
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})

	t.Run("single advance with payment means should set FormaPlatnosci", func(t *testing.T) {
		x := time.Date(2023, time.July, 28, 0, 0, 0, 0, time.UTC)
		d := cal.DateOf(x)
		amt, err := num.AmountFromString("245.890")
		require.NoError(t, err)
		zero, err := num.AmountFromString("0")
		require.NoError(t, err)

		payment := &bill.PaymentDetails{
			Advances: []*pay.Record{{
				Date:   &d,
				Amount: amt,
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyPaymentMeans: "1", // cash
				}),
			}},
		}
		totals := &bill.Totals{
			Due:      &zero,
			Advances: &amt,
		}
		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "1",
			PaymentDate:            d.String(),
			PartiallyPaidMarker:    "",
			AdvancePayments:        []*ksef.AdvancePayment{},
			DueDates:               []*ksef.DueDate{},
			PaymentMean:            "1",
			OtherPaymentMeanMarker: "",
			OtherPaymentMean:       "",
			BankAccounts:           []*ksef.BankAccount(nil),
			FactorBankAccounts:     []*ksef.BankAccount(nil),
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})

	t.Run("multiple advances sets partially paid marker and advance fields", func(t *testing.T) {
		// Partially paid in advance
		x := time.Date(2023, time.July, 28, 0, 0, 0, 0, time.UTC)
		d := cal.DateOf(x)
		firstNum, err := num.AmountFromString("245.890")
		require.NoError(t, err)
		secondNum, err := num.AmountFromString("45.990")
		require.NoError(t, err)

		payment := &bill.PaymentDetails{
			Advances: []*pay.Record{{Date: &d, Amount: firstNum}, {Date: &d, Amount: secondNum}},
		}
		totals := &bill.Totals{
			Due:      &secondNum,
			Advances: &firstNum,
		}
		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "",
			PaymentDate:            "",
			PartiallyPaidMarker:    "1",
			AdvancePayments:        []*ksef.AdvancePayment{{PaymentAmount: firstNum.String(), PaymentDate: d.String()}, {PaymentAmount: secondNum.String(), PaymentDate: d.String()}},
			DueDates:               []*ksef.DueDate{},
			PaymentMean:            "",
			OtherPaymentMeanMarker: "",
			OtherPaymentMean:       "",
			BankAccounts:           []*ksef.BankAccount(nil),
			FactorBankAccounts:     []*ksef.BankAccount(nil),
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})

	t.Run("multiple advances already paid sets partially paid marker two", func(t *testing.T) {
		// Fully settled using multiple advances
		x := time.Date(2023, time.July, 28, 0, 0, 0, 0, time.UTC)
		d := cal.DateOf(x)
		firstNum, err := num.AmountFromString("245.890")
		require.NoError(t, err)
		secondNum, err := num.AmountFromString("45.990")
		require.NoError(t, err)
		zero, err := num.AmountFromString("0")
		require.NoError(t, err)

		payment := &bill.PaymentDetails{
			Advances: []*pay.Record{{Date: &d, Amount: firstNum}, {Date: &d, Amount: secondNum}},
		}
		totals := &bill.Totals{
			Due:      &zero,
			Advances: &firstNum,
		}
		pay := ksef.NewPayment(payment, totals)
		result := &ksef.Payment{
			PaidMarker:             "",
			PaymentDate:            "",
			PartiallyPaidMarker:    "2", // marks that invoice was fully settled using advances
			AdvancePayments:        []*ksef.AdvancePayment{{PaymentAmount: firstNum.String(), PaymentDate: d.String()}, {PaymentAmount: secondNum.String(), PaymentDate: d.String()}},
			DueDates:               []*ksef.DueDate{},
			PaymentMean:            "",
			OtherPaymentMeanMarker: "",
			OtherPaymentMean:       "",
			BankAccounts:           []*ksef.BankAccount(nil),
			FactorBankAccounts:     []*ksef.BankAccount(nil),
			Discount:               (*ksef.Discount)(nil),
		}

		assert.Equal(t, result, pay)
	})
}

func TestParsePaymentMeansCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{"Cash payment", "1", "cash"},
		{"Card payment", "2", "card"},
		{"Voucher payment", "3", "other+voucher"},
		{"Cheque payment", "4", "cheque"},
		{"Credit payment", "5", "other+credit"},
		{"Credit transfer", "6", "credit-transfer"},
		{"Online payment", "7", "online"},
		{"Unknown code", "99", "any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ksef.ParsePaymentMeansCode(tt.code)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}
