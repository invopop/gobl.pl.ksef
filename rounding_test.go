package ksef_test

import (
	"testing"

	ksef "github.com/invopop/gobl.pl.ksef"
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxRoundingError(t *testing.T) {
	t.Run("calculates max rounding error for single line", func(t *testing.T) {
		inv := &bill.Invoice{
			Currency: currency.PLN,
			Lines: []*bill.Line{
				{},
			},
		}

		maxErr := ksef.MaxRoundingError(inv)

		// For PLN (2 subunits), max error per line is 0.75 of 0.01 = 0.0075
		// With subunits+2 = 4, that's 75 in the 4th decimal place = 0.0075
		assert.Equal(t, "0.01", maxErr.String())
	})

	t.Run("calculates max rounding error for multiple lines", func(t *testing.T) {
		inv := &bill.Invoice{
			Currency: currency.PLN,
			Lines: []*bill.Line{
				{}, {}, {},
			},
		}

		maxErr := ksef.MaxRoundingError(inv)

		// 3 lines * 0.0075 = 0.0225
		assert.Equal(t, "0.03", maxErr.String())
	})

	t.Run("calculates max rounding error for currency with different subunits", func(t *testing.T) {
		inv := &bill.Invoice{
			Currency: currency.EUR,
			Lines: []*bill.Line{
				{},
			},
		}

		maxErr := ksef.MaxRoundingError(inv)

		// For EUR (2 subunits), same as PLN
		assert.Equal(t, "0.01", maxErr.String())
	})
}

func TestAdjustRounding(t *testing.T) {
	baseInvoice := func() *bill.Invoice {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("3")
		total, _ := num.AmountFromString("300.00")

		return &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "1234567890",
				},
			},
			Customer: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "9876543210",
				},
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item: &org.Item{
						Name:  "Test Item",
						Price: &price,
						Unit:  "h",
					},
					Total: &total,
					Taxes: tax.Set{
						&tax.Combo{
							Category: tax.CategoryVAT,
							Percent:  num.NewPercentage(23, 2),
						},
					},
				},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
		}
	}

	t.Run("no adjustment needed when totals match", func(t *testing.T) {
		inv := baseInvoice()

		// Calculate first to know what the total will be
		err := inv.Calculate()
		require.NoError(t, err)

		ksefTotal := inv.Totals.Payable.String()

		// Now test with matching total
		inv2 := baseInvoice()
		err = ksef.AdjustRounding(inv2, ksefTotal)
		require.NoError(t, err)

		assert.Nil(t, inv2.Totals.Rounding)
	})

	t.Run("applies small rounding adjustment", func(t *testing.T) {
		// Create invoice with 2 lines so max error is 0.015 (2 * 0.0075)
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		inv := &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "1234567890",
				},
			},
			Customer: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "9876543210",
				},
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 1", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
				{
					Index:    2,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 2", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
		}

		// Calculate first to know what the total will be
		err := inv.Calculate()
		require.NoError(t, err)

		// Add a small rounding difference (0.01 PLN - within max error of 0.015 for two lines)
		expectedTotal := inv.Totals.Payable.Add(num.MakeAmount(1, 2))

		// Now test with slightly different total
		inv2 := &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "1234567890",
				},
			},
			Customer: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "9876543210",
				},
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 1", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
				{
					Index:    2,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 2", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
		}

		err = ksef.AdjustRounding(inv2, expectedTotal.String())
		require.NoError(t, err)

		assert.NotNil(t, inv2.Totals.Rounding)
		assert.Equal(t, "0.01", inv2.Totals.Rounding.String())
	})

	t.Run("applies negative rounding adjustment", func(t *testing.T) {
		// Create invoice with 2 lines so max error is 0.015 (2 * 0.0075)
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		inv := &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "1234567890",
				},
			},
			Customer: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "9876543210",
				},
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 1", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
				{
					Index:    2,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 2", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
		}

		// Calculate first to know what the total will be
		err := inv.Calculate()
		require.NoError(t, err)

		// Subtract a small rounding difference (0.01 PLN - within max error of 0.015 for two lines)
		expectedTotal := inv.Totals.Payable.Subtract(num.MakeAmount(1, 2))

		// Now test with slightly different total
		inv2 := &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "1234567890",
				},
			},
			Customer: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "9876543210",
				},
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 1", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
				{
					Index:    2,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 2", Price: &price, Unit: "h"},
					Total:    &total,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
		}

		err = ksef.AdjustRounding(inv2, expectedTotal.String())
		require.NoError(t, err)

		assert.NotNil(t, inv2.Totals.Rounding)
		assert.Equal(t, "-0.01", inv2.Totals.Rounding.String())
	})

	t.Run("rejects large rounding error", func(t *testing.T) {
		inv := baseInvoice()

		// Calculate first to know what the total will be
		err := inv.Calculate()
		require.NoError(t, err)

		calculated := inv.Totals.Payable
		// Add a large difference (1.00 PLN - way beyond acceptable rounding)
		expectedTotal := calculated.Add(num.MakeAmount(100, 2))

		// Now test - should fail
		inv2 := baseInvoice()
		err = ksef.AdjustRounding(inv2, expectedTotal.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match KSeF total")

		var rerr *ksef.RoundingError
		require.ErrorAs(t, err, &rerr)
		assert.Equal(t, calculated.String(), rerr.Calculated.String())
		assert.Equal(t, expectedTotal.String(), rerr.Expected.String())
		assert.Equal(t, "1.00", rerr.Diff.String())
		// 1 line × 0.01 subunit tolerance for PLN
		assert.Equal(t, "0.01", rerr.MaxAllowed.String())
	})

	t.Run("handles invoice with advances", func(t *testing.T) {
		inv := baseInvoice()

		// Add advance payment
		advance, _ := num.AmountFromString("100.00")
		inv.Payment = &bill.PaymentDetails{
			Advances: []*pay.Record{
				{
					Amount: advance,
				},
			},
		}

		// Calculate first to know what the due amount will be
		err := inv.Calculate()
		require.NoError(t, err)

		// For standard invoices, P_15 = Payable (not Due)
		ksefTotal := inv.Totals.Payable.String()

		// Now test
		inv2 := baseInvoice()
		inv2.Payment = &bill.PaymentDetails{
			Advances: []*pay.Record{
				{
					Amount: advance,
				},
			},
		}

		err = ksef.AdjustRounding(inv2, ksefTotal)
		require.NoError(t, err)

		assert.Nil(t, inv2.Totals.Rounding)
	})

	t.Run("handles settlement invoice with advances", func(t *testing.T) {
		inv := baseInvoice()
		inv.Addons = tax.WithAddons(favat.V3)
		inv.Tags = tax.Tags{List: []cbc.Key{favat.TagSettlement}}

		// Add advance payment
		advance, _ := num.AmountFromString("100.00")
		inv.Payment = &bill.PaymentDetails{
			Advances: []*pay.Record{
				{
					Amount: advance,
				},
			},
		}

		// Calculate first to know what the due amount will be
		err := inv.Calculate()
		require.NoError(t, err)

		// For settlement invoices, P_15 = Due (the remaining amount after advances)
		ksefTotal := inv.Totals.Due.String()

		// Now test
		inv2 := baseInvoice()
		inv2.Addons = tax.WithAddons(favat.V3)
		inv2.Tags = tax.Tags{List: []cbc.Key{favat.TagSettlement}}
		inv2.Payment = &bill.PaymentDetails{
			Advances: []*pay.Record{
				{
					Amount: advance,
				},
			},
		}

		err = ksef.AdjustRounding(inv2, ksefTotal)
		require.NoError(t, err)

		assert.Nil(t, inv2.Totals.Rounding)
	})

	t.Run("handles multiple lines with rounding", func(t *testing.T) {
		price1, _ := num.AmountFromString("33.33")
		price2, _ := num.AmountFromString("33.33")
		price3, _ := num.AmountFromString("33.33")
		qty, _ := num.AmountFromString("1")
		total1, _ := num.AmountFromString("33.33")
		total2, _ := num.AmountFromString("33.33")
		total3, _ := num.AmountFromString("33.33")

		inv := &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "1234567890",
				},
			},
			Customer: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "9876543210",
				},
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 1", Price: &price1, Unit: "h"},
					Total:    &total1,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
				{
					Index:    2,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 2", Price: &price2, Unit: "h"},
					Total:    &total2,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
				{
					Index:    3,
					Quantity: qty,
					Item:     &org.Item{Name: "Item 3", Price: &price3, Unit: "h"},
					Total:    &total3,
					Taxes:    tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
		}

		err := inv.Calculate()
		require.NoError(t, err)

		// KSEF might round 99.99 instead of the calculated total
		// Add a small difference
		ksefTotal := inv.Totals.Payable.Add(num.MakeAmount(1, 2))

		err = ksef.AdjustRounding(inv, ksefTotal.String())
		require.NoError(t, err)

		// Should have a rounding adjustment
		assert.NotNil(t, inv.Totals.Rounding)
		assert.True(t, inv.Totals.Rounding.Abs().Compare(num.MakeAmount(2, 2)) <= 0)
	})

	t.Run("returns error when totals are nil after calculation", func(t *testing.T) {
		// An empty invoice will fail Calculate() or produce nil Totals
		inv := &bill.Invoice{}

		err := ksef.AdjustRounding(inv, "100.00")
		require.Error(t, err)
	})
}
