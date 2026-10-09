package ksef_test

import (
	"testing"

	"github.com/invopop/gobl"
	ksef "github.com/invopop/gobl.pl.ksef"
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl.pl.ksef/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFAVAT(t *testing.T) {
	t.Run("should return a Document with KSeF data", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-standard.json")
		require.NoError(t, err)

		assert.Equal(t, "Faktura", doc.XMLName.Local)
		assert.Equal(t, "http://crd.gov.pl/wzor/2025/06/25/13775/", doc.XMLNamespace)
		assert.Equal(t, "http://www.w3.org/2001/XMLSchema", doc.XSDNamespace)
		assert.Equal(t, "http://www.w3.org/2001/XMLSchema-instance", doc.XSINamespace)
		assert.NotNil(t, doc.Header)
		assert.NotNil(t, doc.Buyer)
		assert.NotNil(t, doc.Seller)
		assert.NotNil(t, doc.Inv)
	})

	t.Run("should return bytes of the KSeF document", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-standard.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		output, err := test.LoadOutputFile("invoice-standard.xml")
		require.NoError(t, err)

		// Normalize dynamic timestamps before comparison
		assert.Equal(t, test.NormalizeXMLDate(string(output)), test.NormalizeXMLDate(string(data)))
	})

	t.Run("should return bytes of the credit-note invoice", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("credit-note-standard.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		output, err := test.LoadOutputFile("credit-note-standard.xml")
		require.NoError(t, err)

		// Normalize dynamic timestamps before comparison
		assert.Equal(t, test.NormalizeXMLDate(string(output)), test.NormalizeXMLDate(string(data)))
	})

	t.Run("should generate valid KSeF document", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-standard.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)
	})

	t.Run("should generate valid credit-note", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("credit-note-standard.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)
	})

	t.Run("should generate valid simplified invoice", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-simplified.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)
	})

	t.Run("should generate valid self-billed invoice", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-self-billed.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)

		output, err := test.LoadOutputFile("invoice-self-billed.xml")
		require.NoError(t, err)

		// Normalize dynamic timestamps before comparison
		assert.Equal(t, test.NormalizeXMLDate(string(output)), test.NormalizeXMLDate(string(data)))
	})

	t.Run("should generate valid exempt invoice", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-exempt.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)
	})

	t.Run("should generate valid reverse-charge invoice", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-reverse-charge.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)
	})

	t.Run("should generate valid prepayment invoice", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-prepayment.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)
	})

	t.Run("should generate valid settlement invoice", func(t *testing.T) {
		doc, err := test.BuildFAVATFrom("invoice-settlement.json")
		require.NoError(t, err)

		data, err := doc.Bytes()
		require.NoError(t, err)

		test.ValidateAgainstFA3Schema(t, data)
	})
}

// TestBuildFAVATCurrencyPrecision covers the FA(3) limit of two decimal places
// on every amount, which a currency with finer subunits cannot satisfy.
func TestBuildFAVATCurrencyPrecision(t *testing.T) {
	newInvoice := func(cur currency.Code) *bill.Invoice {
		inv := &bill.Invoice{
			Addons:    tax.WithAddons(favat.V3),
			IssueDate: cal.MakeDate(2024, 6, 15),
			Code:      "TEST-1",
			Currency:  cur,
			Supplier: &org.Party{
				Name:  "Supplier",
				TaxID: &tax.Identity{Country: l10n.PL.Tax(), Code: "9876543210"},
			},
			Customer: &org.Party{
				Name:  "Customer",
				TaxID: &tax.Identity{Country: l10n.PL.Tax(), Code: "1111111111"},
			},
			Lines: []*bill.Line{
				{
					Quantity: num.MakeAmount(3, 0),
					Item:     &org.Item{Name: "Item", Price: num.NewAmount(3333, 3)},
					Taxes:    tax.Set{{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			},
		}
		if cur != currency.PLN {
			inv.ExchangeRates = []*currency.ExchangeRate{
				{From: cur, To: currency.PLN, Amount: num.MakeAmount(106, 1)},
			}
		}
		return inv
	}

	t.Run("rejects a currency with more than two decimal places", func(t *testing.T) {
		// Bahraini dinar has three subunits, and FA(3) lists it, so rounding
		// to the currency would still leave P_11 and P_15 unrepresentable.
		env, err := gobl.Envelop(newInvoice(currency.BHD))
		require.NoError(t, err)

		_, err = ksef.BuildFavat(env)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "BHD has 3 decimal places")
	})

	t.Run("accepts a currency with fewer decimal places", func(t *testing.T) {
		// Japanese yen has none, and TKwotowy makes the fraction optional.
		env, err := gobl.Envelop(newInvoice(currency.JPY))
		require.NoError(t, err)

		doc, err := ksef.BuildFavat(env)
		require.NoError(t, err)
		require.Len(t, doc.Inv.Lines, 1)
		assert.Equal(t, "10", doc.Inv.Lines[0].NetPriceTotal)
	})
}

// TestBuildFAVATRounding covers the recalculation to the `currency` rounding
// rule. FA(3) amounts (TKwotowy) allow at most two decimal places, so line
// totals carrying the extra precision of the `precise` rule would not pass the
// schema.
func TestBuildFAVATRounding(t *testing.T) {
	// Three units at 3.333 sums to 9.999, which the `precise` rule keeps as
	// is and the `currency` rule rounds to 10.00.
	newInvoice := func(rounding cbc.Key) *bill.Invoice {
		return &bill.Invoice{
			Addons:    tax.WithAddons(favat.V3),
			IssueDate: cal.MakeDate(2024, 6, 15),
			Code:      "TEST-1",
			Currency:  currency.PLN,
			Tax:       &bill.Tax{Rounding: rounding},
			Supplier: &org.Party{
				Name:  "Supplier",
				TaxID: &tax.Identity{Country: l10n.PL.Tax(), Code: "9876543210"},
			},
			Customer: &org.Party{
				Name:  "Customer",
				TaxID: &tax.Identity{Country: l10n.PL.Tax(), Code: "1111111111"},
			},
			Lines: []*bill.Line{
				{
					Quantity: num.MakeAmount(3, 0),
					Item:     &org.Item{Name: "Item", Price: num.NewAmount(3333, 3)},
					Taxes:    tax.Set{{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			},
		}
	}

	t.Run("rounds precise line totals to the currency", func(t *testing.T) {
		env, err := gobl.Envelop(newInvoice(tax.RoundingRulePrecise))
		require.NoError(t, err)

		doc, err := ksef.BuildFavat(env)
		require.NoError(t, err)

		require.Len(t, doc.Inv.Lines, 1)
		assert.Equal(t, "10.00", doc.Inv.Lines[0].NetPriceTotal)
	})

	t.Run("leaves currency-rounded invoices untouched", func(t *testing.T) {
		env, err := gobl.Envelop(newInvoice(tax.RoundingRuleCurrency))
		require.NoError(t, err)

		doc, err := ksef.BuildFavat(env)
		require.NoError(t, err)

		require.Len(t, doc.Inv.Lines, 1)
		assert.Equal(t, "10.00", doc.Inv.Lines[0].NetPriceTotal)
	})
}

func TestBuildFAVATCreditNote(t *testing.T) {
	load := func(t *testing.T) (*gobl.Envelope, *bill.Invoice) {
		t.Helper()
		env, err := test.LoadTestEnvelope("credit-note-standard.json")
		require.NoError(t, err)
		return env, env.Extract().(*bill.Invoice)
	}

	t.Run("should negate amounts without modifying the envelope", func(t *testing.T) {
		env, inv := load(t)

		doc, err := ksef.BuildFavat(env)
		require.NoError(t, err)

		assert.Equal(t, "-10", doc.Inv.Lines[0].Quantity)
		assert.Equal(t, "-100.00", doc.Inv.Lines[0].NetPriceTotal)
		assert.Equal(t, "-123.00", doc.Inv.TotalAmountDue)
		assert.Equal(t, "10", inv.Lines[0].Quantity.String())
		assert.Equal(t, "123.00", inv.Totals.Payable.String())
	})

	t.Run("should handle credit notes with bypass tag", func(t *testing.T) {
		env, inv := load(t)
		inv.SetTags(tax.TagBypass)
		// Totals that would not survive a recalculation
		rate := inv.Totals.Taxes.Categories[0].Rates[0]
		rate.Amount = num.MakeAmount(2301, 2)
		inv.Totals.Taxes.Categories[0].Amount = rate.Amount
		inv.Totals.Taxes.Sum = rate.Amount
		inv.Totals.Tax = rate.Amount
		inv.Totals.TotalWithTax = num.MakeAmount(12301, 2)
		inv.Totals.Payable = inv.Totals.TotalWithTax

		doc, err := ksef.BuildFavat(env)
		require.NoError(t, err)

		assert.Equal(t, "KOR", doc.Inv.InvoiceType)
		assert.Equal(t, "-10", doc.Inv.Lines[0].Quantity)
		assert.Equal(t, "-100.00", doc.Inv.Lines[0].NetPriceTotal)
		assert.Equal(t, "-100.00", doc.Inv.StandardRateNetSale)
		assert.Equal(t, "-23.01", doc.Inv.StandardRateTax)
		assert.Equal(t, "-123.01", doc.Inv.TotalAmountDue)
		assert.Equal(t, "123.01", inv.Totals.Payable.String())

		data, err := doc.Bytes()
		require.NoError(t, err)
		test.ValidateAgainstFA3Schema(t, data)
	})

	t.Run("should handle credit notes with rounding", func(t *testing.T) {
		env, inv := load(t)
		rounding := num.MakeAmount(-2, 2)
		inv.Totals.Rounding = &rounding
		require.NoError(t, env.Calculate())
		require.Equal(t, "122.98", inv.Totals.Payable.String())

		doc, err := ksef.BuildFavat(env)
		require.NoError(t, err)

		assert.Equal(t, "-23.00", doc.Inv.StandardRateTax)
		assert.Equal(t, "-122.98", doc.Inv.TotalAmountDue)
	})
}
