package ksef_test

import (
	"testing"

	ksef "github.com/invopop/gobl.pl.ksef"
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLines(t *testing.T) {
	t.Run("converts basic lines", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("2")
		total, _ := num.AmountFromString("200.00")

		lines := []*bill.Line{
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
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Equal(t, 1, result[0].LineNumber)
		assert.Equal(t, "Test Item", result[0].Name)
		assert.Equal(t, "HUR", result[0].Measure)
		assert.Equal(t, "100.00", result[0].NetUnitPrice)
		assert.Equal(t, "2", result[0].Quantity)
		assert.Equal(t, "200.00", result[0].NetPriceTotal)
		assert.Equal(t, "23", result[0].VATRate)
	})

	t.Run("takes the measure from the untdid-unit extension", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Item with a non-standard UN/ECE code",
					Price: &price,
					Ext: tax.ExtensionsOf(cbc.CodeMap{
						untdid.ExtKeyUnit: "SZT",
					}),
				},
				Total: &total,
				Taxes: tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Equal(t, "SZT", result[0].Measure)
	})

	t.Run("prefers the unit key over the untdid-unit extension", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Item",
					Price: &price,
					Unit:  org.UnitPiece,
					Ext: tax.ExtensionsOf(cbc.CodeMap{
						untdid.ExtKeyUnit: "SZT",
					}),
				},
				Total: &total,
				Taxes: tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		// GOBL resolves a conflict in favour of the unit, and normalization
		// realigns the extension to match it, so P_8A follows the unit.
		assert.Equal(t, "H87", result[0].Measure)
	})

	t.Run("maps every GOBL unit key to a measure", func(t *testing.T) {
		// Since GOBL v0.506.1 every defined unit has an exact UNTDID
		// equivalent, so a line carrying a unit always yields a P_8A.
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		for _, def := range org.UnitDefinitions {
			lines := []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item: &org.Item{
						Name:  "Item",
						Price: &price,
						Unit:  def.Key,
					},
					Total: &total,
					Taxes: tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
				},
			}

			result := ksef.NewLines(lines)

			require.Len(t, result, 1)
			assert.NotEmpty(t, result[0].Measure, "unit %s has no UN/ECE code", def.Key)
		}
	})

	t.Run("leaves the measure empty when the item has no unit", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Item",
					Price: &price,
				},
				Total: &total,
				Taxes: tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Empty(t, result[0].Measure)
	})

	t.Run("handles multiple lines", func(t *testing.T) {
		price, _ := num.AmountFromString("50.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("50.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Item 1",
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
			{
				Index:    2,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Item 2",
					Price: &price,
					Unit:  "service",
				},
				Total: &total,
				Taxes: tax.Set{
					&tax.Combo{
						Category: tax.CategoryVAT,
						Percent:  num.NewPercentage(8, 2),
					},
				},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 2)
		assert.Equal(t, "Item 1", result[0].Name)
		assert.Equal(t, "Item 2", result[1].Name)
		assert.Equal(t, 1, result[0].LineNumber)
		assert.Equal(t, 2, result[1].LineNumber)
	})

	t.Run("handles line without VAT percent", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Exempt Item",
					Price: &price,
					Unit:  "h",
				},
				Total: &total,
				Taxes: tax.Set{
					&tax.Combo{
						Category: tax.CategoryVAT,
						// No Percent for exempt items
					},
				},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Equal(t, "", result[0].VATRate)
	})

	t.Run("handles line with discounts", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("2")
		total, _ := num.AmountFromString("180.00")
		discountAmt, _ := num.AmountFromString("20.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Discounted Item",
					Price: &price,
					Unit:  "h",
				},
				Discounts: []*bill.LineDiscount{
					{
						Amount: discountAmt,
						Reason: "Volume discount",
					},
				},
				Total: &total,
				Taxes: tax.Set{
					&tax.Combo{
						Category: tax.CategoryVAT,
						Percent:  num.NewPercentage(23, 2),
					},
				},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Equal(t, "20.00", result[0].Discount) // total line discount, not per-unit
	})

	t.Run("handles multiple discounts on same line", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("4")
		total, _ := num.AmountFromString("360.00")
		discount1, _ := num.AmountFromString("20.00")
		discount2, _ := num.AmountFromString("20.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Multiple Discounts",
					Price: &price,
					Unit:  "h",
				},
				Discounts: []*bill.LineDiscount{
					{Amount: discount1, Reason: "Discount 1"},
					{Amount: discount2, Reason: "Discount 2"},
				},
				Total: &total,
				Taxes: tax.Set{
					&tax.Combo{
						Category: tax.CategoryVAT,
						Percent:  num.NewPercentage(23, 2),
					},
				},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Equal(t, "40.00", result[0].Discount) // total line discount: 20.00 + 20.00 = 40.00
	})

	t.Run("emits gross fields when prices_include=VAT", func(t *testing.T) {
		// Mews-style: GOBL invoice has tax.prices_include=VAT, so item price
		// and line total are gross. KSeF receives them via P_9B/P_11A
		// (art. 106e(7)-(8)) instead of the net P_9A/P_11.
		price, _ := num.AmountFromString("35.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("35.00")

		invoice := &bill.Invoice{
			Tax: &bill.Tax{PricesInclude: tax.CategoryVAT},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: qty,
					Item: &org.Item{
						Name:  "Starogdańskie 0.75l",
						Price: &price,
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
		}

		result := ksef.NewLinesForInvoice(invoice)

		require.Len(t, result, 1)
		assert.Equal(t, "35.00", result[0].GrossUnitPrice)
		assert.Equal(t, "35.00", result[0].GrossPriceTotal)
		assert.Empty(t, result[0].NetUnitPrice)
		assert.Empty(t, result[0].NetPriceTotal)
	})

	t.Run("uses Item.Meta unit-label when set", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Item: &org.Item{
					Name:  "Item with non-GOBL unit",
					Price: &price,
					Meta:  cbc.Meta{"unit-label": "kilo"},
				},
				Total: &total,
				Taxes: tax.Set{
					&tax.Combo{
						Category: tax.CategoryVAT,
						Percent:  num.NewPercentage(23, 2),
					},
				},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Equal(t, "kilo", result[0].Measure)
	})

	t.Run("maps line period end to P_6A", func(t *testing.T) {
		price, _ := num.AmountFromString("100.00")
		qty, _ := num.AmountFromString("1")
		total, _ := num.AmountFromString("100.00")

		lines := []*bill.Line{
			{
				Index:    1,
				Quantity: qty,
				Period: &cal.Period{
					Start: cal.NewDate(2026, 4, 8),
					End:   cal.NewDate(2026, 4, 8),
				},
				Item:  &org.Item{Name: "Room night", Price: &price, Unit: "one"},
				Total: &total,
				Taxes: tax.Set{&tax.Combo{Category: tax.CategoryVAT, Percent: num.NewPercentage(23, 2)}},
			},
		}

		result := ksef.NewLines(lines)

		require.Len(t, result, 1)
		assert.Equal(t, "2026-04-08", result[0].CompletionDate)
	})
}

// TestLineMeasureRoundTrip checks that a KSeF unit of measure survives the
// trip through GOBL and back, whichever of the three representations holds it:
// a unit key, the untdid-unit extension, or the unit-label meta.
func TestLineMeasureRoundTrip(t *testing.T) {
	price, _ := num.AmountFromString("100.00")
	total, _ := num.AmountFromString("100.00")

	for _, measure := range []string{
		"HUR",  // maps to a GOBL unit
		"KGM",  // maps to a GOBL unit
		"H87",  // maps to a GOBL unit
		"13",   // maps to portion, and is numeric
		"SZT",  // UN/ECE-shaped, no GOBL unit
		"D61",  // a real UNTDID code with no GOBL unit
		"PLN",  // not a unit at all, but shaped like one
		"szt.", // free-form
		"kilo", // free-form
	} {
		t.Run(measure, func(t *testing.T) {
			line, err := (&ksef.Line{
				Name:         "Item",
				Quantity:     "1",
				NetUnitPrice: "100.00",
				Measure:      measure,
				VATRate:      "23",
			}).ToGOBL()
			require.NoError(t, err)

			line.Index = 1
			line.Item.Price = &price
			line.Total = &total

			result := ksef.NewLines([]*bill.Line{line})
			require.Len(t, result, 1)
			assert.Equal(t, measure, result[0].Measure)
		})
	}
}

func TestLineToGOBL(t *testing.T) {
	t.Run("converts basic KSEF line to GOBL", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Test Item",
			Quantity:      "2",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "23",
			NetPriceTotal: "200.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Equal(t, "Test Item", line.Item.Name)
		assert.Equal(t, "2", line.Quantity.String())
		assert.Equal(t, "100.00", line.Item.Price.String())
		assert.Equal(t, org.UnitHour, line.Item.Unit)
		// The unit implies the code, so it is not duplicated in the extension.
		assert.Empty(t, line.Item.Ext.Get(untdid.ExtKeyUnit))
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.CategoryVAT, line.Taxes[0].Category)
		assert.Equal(t, "23", line.Taxes[0].Percent.Amount().MinimalString())
	})

	t.Run("handles line with discount", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Discounted Item",
			Quantity:      "2",
			NetUnitPrice:  "100.00",
			Discount:      "20.00",
			Measure:       "HUR",
			VATRate:       "23",
			NetPriceTotal: "180.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Discounts, 1)
		// P_10 mapped directly as total line discount
		assert.Equal(t, "20.00", line.Discounts[0].Amount.String())
	})

	t.Run("handles discount with single unit quantity", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Single Unit Discounted Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Discount:      "15.00",
			Measure:       "HUR",
			VATRate:       "23",
			NetPriceTotal: "85.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Discounts, 1)
		// P_10 mapped directly as total line discount
		assert.Equal(t, "15.00", line.Discounts[0].Amount.String())
	})

	t.Run("skips discount when unit discount is zero", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "No Discount Item",
			Quantity:      "2",
			NetUnitPrice:  "100.00",
			Discount:      "0.00",
			Measure:       "HUR",
			VATRate:       "23",
			NetPriceTotal: "200.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Empty(t, line.Discounts)
	})

	t.Run("handles discount without multiplying when quantity is zero", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:         "Zero Qty Discounted Item",
			Quantity:     "0",
			NetUnitPrice: "100.00",
			Discount:     "10.00",
			Measure:      "HUR",
			VATRate:      "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Discounts, 1)
		// No line total available; P_10 used directly as total discount
		assert.Equal(t, "10.00", line.Discounts[0].Amount.String())
	})

	t.Run("handles exempt line with zero VAT", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Exempt Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "zw",
			NetPriceTotal: "100.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.KeyExempt, line.Taxes[0].Key)
		assert.Equal(t, "7", string(line.Taxes[0].Ext.Get(favat.ExtKeyTaxCategory)))
	})

	t.Run("handles intra-community supply", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "EU Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "0 WDT",
			NetPriceTotal: "100.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.KeyIntraCommunity, line.Taxes[0].Key)
		assert.Equal(t, "6.2", string(line.Taxes[0].Ext.Get(favat.ExtKeyTaxCategory)))
	})

	t.Run("handles export supply", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Export Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "0 EX",
			NetPriceTotal: "100.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.KeyExport, line.Taxes[0].Key)
		assert.Equal(t, "6.3", string(line.Taxes[0].Ext.Get(favat.ExtKeyTaxCategory)))
	})

	t.Run("handles reverse charge", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Reverse Charge Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "np II",
			NetPriceTotal: "100.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.KeyReverseCharge, line.Taxes[0].Key)
		assert.Equal(t, "9", string(line.Taxes[0].Ext.Get(favat.ExtKeyTaxCategory)))
	})

	t.Run("handles domestic reverse charge", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Domestic RC Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "oo",
			NetPriceTotal: "100.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.KeyReverseCharge, line.Taxes[0].Key)
		assert.Equal(t, "10", string(line.Taxes[0].Ext.Get(favat.ExtKeyTaxCategory)))
	})

	t.Run("handles reduced rate 8%", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Reduced Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "8",
			NetPriceTotal: "100.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.KeyStandard, line.Taxes[0].Key)
		assert.Equal(t, tax.RateReduced, line.Taxes[0].Rate)
		assert.Equal(t, "8", line.Taxes[0].Percent.Amount().MinimalString())
	})

	t.Run("handles super reduced rate 5%", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:          "Super Reduced Item",
			Quantity:      "1",
			NetUnitPrice:  "100.00",
			Measure:       "HUR",
			VATRate:       "5",
			NetPriceTotal: "100.00",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Len(t, line.Taxes, 1)
		assert.Equal(t, tax.KeyStandard, line.Taxes[0].Key)
		assert.Equal(t, tax.RateSuperReduced, line.Taxes[0].Rate)
		assert.Equal(t, "5", line.Taxes[0].Percent.Amount().MinimalString())
	})

	t.Run("handles gross unit price (P_9B)", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:           "Gross Price Item",
			Quantity:       "1",
			GrossUnitPrice: "123.00",
			Measure:        "HUR",
			VATRate:        "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Equal(t, "Gross Price Item", line.Item.Name)
		require.NotNil(t, line.Item.Price)
		assert.Equal(t, "123.00", line.Item.Price.String())
		assert.Equal(t, "1", line.Quantity.String())
	})

	t.Run("prefers net unit price over gross", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:           "Both Prices Item",
			Quantity:       "1",
			NetUnitPrice:   "100.00",
			GrossUnitPrice: "123.00",
			Measure:        "HUR",
			VATRate:        "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		require.NotNil(t, line.Item.Price)
		assert.Equal(t, "100.00", line.Item.Price.String())
	})

	t.Run("handles valid UNECE unit codes", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:         "Item with UNECE unit",
			Quantity:     "1",
			NetUnitPrice: "100.00",
			Measure:      "KGM",
			VATRate:      "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		// A code GOBL has a unit for becomes that unit, which implies the
		// same code again on the way back out.
		assert.Equal(t, org.UnitKilogram, line.Item.Unit)
		assert.Empty(t, line.Item.Ext.Get(untdid.ExtKeyUnit))
		assert.Empty(t, line.Item.Meta["unit-label"])
	})

	t.Run("keeps unmapped UNECE unit codes in the extension", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:         "Item with unmapped UNECE unit",
			Quantity:     "1",
			NetUnitPrice: "100.00",
			Measure:      "D61", // minute [unit of angle], no GOBL equivalent
			VATRate:      "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Equal(t, cbc.Code("D61"), line.Item.Ext.Get(untdid.ExtKeyUnit))
		assert.Equal(t, cbc.KeyEmpty, line.Item.Unit)
		assert.Empty(t, line.Item.Meta["unit-label"])
	})

	t.Run("handles valid GOBL unit codes", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:         "Item with GOBL unit",
			Quantity:     "1",
			NetUnitPrice: "100.00",
			Measure:      "h",
			VATRate:      "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Equal(t, org.UnitHour, line.Item.Unit)
		assert.Empty(t, line.Item.Ext.Get(untdid.ExtKeyUnit))
	})

	t.Run("preserves invalid unit under Item.Meta unit-label", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:         "Item with non-GOBL unit",
			Quantity:     "1",
			NetUnitPrice: "100.00",
			Measure:      "szt",
			VATRate:      "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Equal(t, cbc.KeyEmpty, line.Item.Unit)
		assert.Equal(t, "szt", line.Item.Meta["unit-label"])
		assert.Empty(t, line.Item.Ext.Get(untdid.ExtKeyUnit))
	})

	t.Run("trims whitespace around measure", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:         "Padded canonical unit",
			Quantity:     "1",
			NetUnitPrice: "100.00",
			Measure:      "  KGM  ",
			VATRate:      "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Equal(t, org.UnitKilogram, line.Item.Unit)
		assert.Empty(t, line.Item.Meta["unit-label"])
	})

	t.Run("trims whitespace around non-canonical measure", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:         "Padded label unit",
			Quantity:     "1",
			NetUnitPrice: "100.00",
			Measure:      "  szt  ",
			VATRate:      "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		assert.Equal(t, cbc.KeyEmpty, line.Item.Unit)
		assert.Equal(t, "szt", line.Item.Meta["unit-label"])
	})

	t.Run("maps discount directly from P_10", func(t *testing.T) {
		// Auchan-style data: P_9B=5.98, qty=2, P_10=2.40, P_11A=9.56
		ksefLine := &ksef.Line{
			Name:            "Auchan Item",
			Quantity:        "2",
			GrossUnitPrice:  "5.98",
			Discount:        "2.40",
			GrossPriceTotal: "9.56",
			VATRate:         "23",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		require.Len(t, line.Discounts, 1)
		// P_10 mapped directly as total line discount
		assert.Equal(t, "2.40", line.Discounts[0].Amount.String())
	})

	t.Run("maps P_6A to line period", func(t *testing.T) {
		ksefLine := &ksef.Line{
			Name:           "Room night",
			Quantity:       "1",
			NetUnitPrice:   "300.00",
			NetPriceTotal:  "300.00",
			VATRate:        "23",
			CompletionDate: "2026-04-08",
		}

		line, err := ksefLine.ToGOBL()

		require.NoError(t, err)
		require.NotNil(t, line.Period)
		assert.Equal(t, "2026-04-08", line.Period.Start.String())
		assert.Equal(t, "2026-04-08", line.Period.End.String())
	})
}
