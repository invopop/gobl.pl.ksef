package ksef_test

import (
	"testing"
	"time"

	ksef "github.com/invopop/gobl.ksef"
	"github.com/invopop/gobl/addons/pl/favat"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
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

func TestNewFavatInv(t *testing.T) {
	baseInvoice := func() *bill.Invoice {
		return &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
				},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
			Totals: &bill.Totals{
				Taxes: &tax.Total{},
			},
		}
	}

	t.Run("sets preceding invoice", func(t *testing.T) {
		inv := baseInvoice()
		inv.Preceding = []*org.DocumentRef{
			{},
		}

		invoice := ksef.NewFavatInv(inv)

		assert.NotNil(t, invoice.CorrectedInv)
	})

	t.Run("sets correction reason", func(t *testing.T) {
		reason := "example reason"

		inv := baseInvoice()
		inv.Preceding = []*org.DocumentRef{
			{
				Reason: reason,
			},
		}

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, reason, invoice.CorrectionReason)
	})

	t.Run("sets correction type", func(t *testing.T) {
		inv := baseInvoice()
		inv.Preceding = []*org.DocumentRef{
			{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyEffectiveDate: "1",
				}),
			},
		}

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.CorrectionType)
	})

	t.Run("sets the self-billing annotation to false in non-self-billed invoices", func(t *testing.T) {
		inv := baseInvoice()

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "2", invoice.Annotations.SelfBilling)
	})

	t.Run("sets the self-billing annotation to true in self-billed invoices", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeySelfBilling, "1")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.SelfBilling)
	})

	t.Run("sets reverse charge annotation", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyReverseCharge, "1")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.ReverseCharge)
	})

	t.Run("sets cash accounting annotation", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyCashAccounting, "1")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.CashAccounting)
	})

	t.Run("sets split payment annotation", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeySplitPayment, "1")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.SplitPaymentMechanism)
	})

	t.Run("sets tax exemption annotation with marker", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyExemption, "A")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.TaxExemption.Marker)
	})

	t.Run("sets margin scheme travel agency", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyMarginScheme, "2")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.MarginScheme.Marker)
		assert.Equal(t, "1", invoice.Annotations.MarginScheme.TravelAgencyMargin)
	})

	t.Run("sets margin scheme used goods", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyMarginScheme, "3.1")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.MarginScheme.Marker)
		assert.Equal(t, "1", invoice.Annotations.MarginScheme.UsedGoodsMargin)
	})

	t.Run("sets margin scheme art works", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyMarginScheme, "3.2")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.MarginScheme.Marker)
		assert.Equal(t, "1", invoice.Annotations.MarginScheme.ArtWorksMargin)
	})

	t.Run("sets margin scheme collectibles and antiques", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyMarginScheme, "3.3")

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "1", invoice.Annotations.MarginScheme.Marker)
		assert.Equal(t, "1", invoice.Annotations.MarginScheme.CollectiblesAndAntiquesMargin)
	})

	t.Run("sets additional description from notes", func(t *testing.T) {
		inv := baseInvoice()
		inv.Notes = []*org.Note{
			{
				Key:  "general",
				Text: "Test note text",
			},
		}

		invoice := ksef.NewFavatInv(inv)

		assert.Len(t, invoice.AdditionalDescription, 1)
		assert.Equal(t, "general", invoice.AdditionalDescription[0].Key)
		assert.Equal(t, "Test note text", invoice.AdditionalDescription[0].Value)
	})

	t.Run("excludes footer notes from additional description", func(t *testing.T) {
		inv := baseInvoice()
		inv.Notes = []*org.Note{
			{
				Key:  "general",
				Text: "Regular note",
			},
			{
				Src:  ksef.NoteSourceFooter,
				Text: "Footer note",
			},
		}

		invoice := ksef.NewFavatInv(inv)

		assert.Len(t, invoice.AdditionalDescription, 1)
		assert.Equal(t, "Regular note", invoice.AdditionalDescription[0].Value)
	})

	t.Run("sets invoice number with series", func(t *testing.T) {
		inv := baseInvoice()
		inv.Series = "INV"
		inv.Code = "001"

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "INV-001", invoice.SequentialNumber)
	})

	t.Run("sets invoice number without series", func(t *testing.T) {
		inv := baseInvoice()
		inv.Code = "001"

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "001", invoice.SequentialNumber)
	})

	t.Run("sets total amount due from payable for VAT invoices ignoring due", func(t *testing.T) {
		inv := baseInvoice()
		payable, err := num.AmountFromString("25.00")
		require.NoError(t, err)
		due, err := num.AmountFromString("10.00")
		require.NoError(t, err)
		inv.Totals.Payable = payable
		inv.Totals.Due = &due

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "25.00", invoice.TotalAmountDue)
	})

	t.Run("sets total amount due from payable for fully-prepaid VAT invoices", func(t *testing.T) {
		// Regression: a standard VAT invoice fully paid via tracked
		// payment.advances must still report the full gross total in P_15.
		// The advance is tracking info, not a prior ZAL invoice, so it does
		// not reduce P_15 (would otherwise render Brutto=0 in KSeF UI).
		inv := baseInvoice()
		payable, err := num.AmountFromString("205.00")
		require.NoError(t, err)
		zero, err := num.AmountFromString("0.00")
		require.NoError(t, err)
		inv.Totals.Payable = payable
		inv.Totals.Due = &zero

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "205.00", invoice.TotalAmountDue)
	})

	t.Run("sets total amount due from payable for VAT invoices when due is nil", func(t *testing.T) {
		inv := baseInvoice()
		payable, err := num.AmountFromString("25.00")
		require.NoError(t, err)
		inv.Totals.Payable = payable

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "25.00", invoice.TotalAmountDue)
	})

	t.Run("sets total amount due from due for ZAL prepayment invoices", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyInvoiceType, "ZAL")
		payable, err := num.AmountFromString("100.00")
		require.NoError(t, err)
		due, err := num.AmountFromString("40.00")
		require.NoError(t, err)
		inv.Totals.Payable = payable
		inv.Totals.Due = &due

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "40.00", invoice.TotalAmountDue)
	})

	t.Run("sets total amount due from due for ROZ settlement invoices", func(t *testing.T) {
		inv := baseInvoice()
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyInvoiceType, "ROZ")
		payable, err := num.AmountFromString("100.00")
		require.NoError(t, err)
		due, err := num.AmountFromString("40.00")
		require.NoError(t, err)
		inv.Totals.Payable = payable
		inv.Totals.Due = &due

		invoice := ksef.NewFavatInv(inv)

		assert.Equal(t, "40.00", invoice.TotalAmountDue)
	})
}

func TestParseLinesSetsPricesInclude(t *testing.T) {
	t.Run("sets PricesInclude when lines use gross pricing", func(t *testing.T) {
		doc := &ksef.Invoice{
			Seller: &ksef.Seller{
				NIP:  "1234567890",
				Name: "Test Supplier",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 1 00-001 Warszawa",
				},
			},
			Buyer: &ksef.Buyer{
				NIP:  "9876543210",
				Name: "Test Buyer",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 2 00-002 Warszawa",
				},
				JST: "2",
				GV:  "2",
			},
			Inv: &ksef.Inv{
				CurrencyCode:        "PLN",
				IssueDate:           "2024-06-15",
				SequentialNumber:    "FV-001",
				TotalAmountDue:      "123.00",
				InvoiceType:         "VAT",
				StandardRateNetSale: "100.00",
				StandardRateTax:     "23.00",
				Annotations: &ksef.Annotations{
					CashAccounting:                      "2",
					SelfBilling:                         "2",
					ReverseCharge:                       "2",
					SplitPaymentMechanism:               "2",
					SimplifiedProcedureBySecondTaxpayer: "2",
					TaxExemption: &ksef.TaxExemption{
						NoExemption: "1",
					},
					NewTransportMeans: &ksef.NewTransportMeans{
						NoNewTransportMeans: "1",
					},
					MarginScheme: &ksef.MarginScheme{
						NoMarginScheme: "1",
					},
				},
				Lines: []*ksef.Line{
					{
						LineNumber:     1,
						Name:           "Gross Price Item",
						Quantity:       "1",
						GrossUnitPrice: "123.00",
						Measure:        "HUR",
						VATRate:        "23",
					},
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Tax)
		assert.Equal(t, tax.CategoryVAT, inv.Tax.PricesInclude)
	})

	t.Run("does not set PricesInclude for net pricing", func(t *testing.T) {
		doc := &ksef.Invoice{
			Seller: &ksef.Seller{
				NIP:  "1234567890",
				Name: "Test Supplier",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 1 00-001 Warszawa",
				},
			},
			Buyer: &ksef.Buyer{
				NIP:  "9876543210",
				Name: "Test Buyer",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 2 00-002 Warszawa",
				},
				JST: "2",
				GV:  "2",
			},
			Inv: &ksef.Inv{
				CurrencyCode:        "PLN",
				IssueDate:           "2024-06-15",
				SequentialNumber:    "FV-001",
				TotalAmountDue:      "123.00",
				InvoiceType:         "VAT",
				StandardRateNetSale: "100.00",
				StandardRateTax:     "23.00",
				Annotations: &ksef.Annotations{
					CashAccounting:                      "2",
					SelfBilling:                         "2",
					ReverseCharge:                       "2",
					SplitPaymentMechanism:               "2",
					SimplifiedProcedureBySecondTaxpayer: "2",
					TaxExemption: &ksef.TaxExemption{
						NoExemption: "1",
					},
					NewTransportMeans: &ksef.NewTransportMeans{
						NoNewTransportMeans: "1",
					},
					MarginScheme: &ksef.MarginScheme{
						NoMarginScheme: "1",
					},
				},
				Lines: []*ksef.Line{
					{
						LineNumber:    1,
						Name:          "Net Price Item",
						Quantity:      "1",
						NetUnitPrice:  "100.00",
						Measure:       "HUR",
						VATRate:       "23",
						NetPriceTotal: "100.00",
					},
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Tax)
		assert.Empty(t, inv.Tax.PricesInclude)
	})
}

// testCreditNoteDoc creates a KSeF Invoice document for credit note tests.
func testCreditNoteDoc(lines []*ksef.Line, totalDue string) *ksef.Invoice {
	return &ksef.Invoice{
		Seller: &ksef.Seller{
			NIP:  "1234567890",
			Name: "Test Supplier",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Testowa 1 00-001 Warszawa",
			},
		},
		Buyer: &ksef.Buyer{
			NIP:  "9876543210",
			Name: "Test Buyer",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Testowa 2 00-002 Warszawa",
			},
			JST: "2",
			GV:  "2",
		},
		Inv: &ksef.Inv{
			CurrencyCode:        "PLN",
			IssueDate:           "2026-01-20",
			SequentialNumber:    "KOR-TEST",
			TotalAmountDue:      totalDue,
			InvoiceType:         "KOR",
			StandardRateNetSale: "-100.00",
			StandardRateTax:     "-23.00",
			CorrectionReason:    "Test correction",
			CorrectedInv: []*ksef.CorrectedInv{
				{IssueDate: "2026-01-15", SequentialNumber: "INV-001"},
			},
			Annotations: &ksef.Annotations{
				CashAccounting:                      "2",
				SelfBilling:                         "2",
				ReverseCharge:                       "2",
				SplitPaymentMechanism:               "2",
				SimplifiedProcedureBySecondTaxpayer: "2",
				TaxExemption: &ksef.TaxExemption{
					NoExemption: "1",
				},
				NewTransportMeans: &ksef.NewTransportMeans{
					NoNewTransportMeans: "1",
				},
				MarginScheme: &ksef.MarginScheme{
					NoMarginScheme: "1",
				},
			},
			Lines: lines,
		},
	}
}

func TestCreditNoteLineInversion(t *testing.T) {
	t.Run("inverts negative qty to positive for differences method", func(t *testing.T) {
		doc := testCreditNoteDoc([]*ksef.Line{
			{
				LineNumber:    1,
				Name:          "Software Services",
				Quantity:      "-10",
				NetUnitPrice:  "10.00",
				Measure:       "HUR",
				VATRate:       "23",
				NetPriceTotal: "-100.00",
			},
		}, "-123.00")

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.Len(t, inv.Lines, 1)

		// Quantity should be inverted from -10 to 10 (GOBL positive convention)
		assert.Equal(t, "10", inv.Lines[0].Quantity.String())
		// Payable should be positive
		assert.Equal(t, "123.00", inv.Totals.Payable.String())
	})

	t.Run("keeps StanPrzed lines positive", func(t *testing.T) {
		doc := testCreditNoteDoc([]*ksef.Line{
			// "After" line (no StanPrzed) — will be inverted
			{
				LineNumber:    1,
				Name:          "Item A",
				Quantity:      "0",
				NetUnitPrice:  "50.00",
				Measure:       "HUR",
				VATRate:       "23",
				NetPriceTotal: "0.00",
			},
			// "Before" line (StanPrzed=1) — stays positive
			{
				LineNumber:             1,
				Name:                   "Item A",
				Quantity:               "2",
				NetUnitPrice:           "50.00",
				Measure:                "HUR",
				VATRate:                "23",
				NetPriceTotal:          "100.00",
				BeforeCorrectionMarker: 1,
			},
		}, "-123.00")

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.Len(t, inv.Lines, 2)

		// "After" line: qty=0 inverted stays 0, no note
		assert.Equal(t, "0", inv.Lines[0].Quantity.String())
		assert.Empty(t, inv.Lines[0].Notes)
		// "Before" (StanPrzed) line: qty=2 stays positive, has note
		assert.Equal(t, "2", inv.Lines[1].Quantity.String())
		require.Len(t, inv.Lines[1].Notes, 1)
		assert.Contains(t, inv.Lines[1].Notes[0].Text, "Before correction")
	})

	t.Run("keeps discount positive for non-StanPrzed lines", func(t *testing.T) {
		// In KSeF credit notes (differences method), qty is negative but
		// P_10 is always a positive total discount. parseLines inverts qty
		// to positive; the discount stays positive as-is.
		doc := testCreditNoteDoc([]*ksef.Line{
			{
				LineNumber:    1,
				Name:          "Discounted Item",
				Quantity:      "-2",
				NetUnitPrice:  "100.00",
				Discount:      "20.00",
				Measure:       "HUR",
				VATRate:       "23",
				NetPriceTotal: "-180.00",
			},
		}, "-221.40")

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.Len(t, inv.Lines, 1)

		// Quantity inverted from -2 to 2
		assert.Equal(t, "2", inv.Lines[0].Quantity.String())
		// P_10 = 20.00 total discount, stays positive
		require.Len(t, inv.Lines[0].Discounts, 1)
		assert.Equal(t, "20.00", inv.Lines[0].Discounts[0].Amount.String())
	})

	t.Run("does not invert lines for standard invoices", func(t *testing.T) {
		doc := &ksef.Invoice{
			Seller: &ksef.Seller{
				NIP:  "1234567890",
				Name: "Test Supplier",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 1 00-001 Warszawa",
				},
			},
			Buyer: &ksef.Buyer{
				NIP:  "9876543210",
				Name: "Test Buyer",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 2 00-002 Warszawa",
				},
				JST: "2",
				GV:  "2",
			},
			Inv: &ksef.Inv{
				CurrencyCode:        "PLN",
				IssueDate:           "2026-01-20",
				SequentialNumber:    "FV-001",
				TotalAmountDue:      "123.00",
				InvoiceType:         "VAT",
				StandardRateNetSale: "100.00",
				StandardRateTax:     "23.00",
				Annotations: &ksef.Annotations{
					CashAccounting:                      "2",
					SelfBilling:                         "2",
					ReverseCharge:                       "2",
					SplitPaymentMechanism:               "2",
					SimplifiedProcedureBySecondTaxpayer: "2",
					TaxExemption: &ksef.TaxExemption{
						NoExemption: "1",
					},
					NewTransportMeans: &ksef.NewTransportMeans{
						NoNewTransportMeans: "1",
					},
					MarginScheme: &ksef.MarginScheme{
						NoMarginScheme: "1",
					},
				},
				Lines: []*ksef.Line{
					{
						LineNumber:    1,
						Name:          "Normal Item",
						Quantity:      "10",
						NetUnitPrice:  "10.00",
						Measure:       "HUR",
						VATRate:       "23",
						NetPriceTotal: "100.00",
					},
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.Len(t, inv.Lines, 1)

		// Quantity should stay as-is for standard invoices
		assert.Equal(t, "10", inv.Lines[0].Quantity.String())
	})
}

// helper to build a minimal KSeF prepayment document for testing.
func testPrepaymentDoc() *ksef.Invoice {
	return &ksef.Invoice{
		Seller: &ksef.Seller{
			NIP:  "1234567890",
			Name: "Test Supplier",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Testowa 1 00-001 Warszawa",
			},
		},
		Buyer: &ksef.Buyer{
			NIP:  "9876543210",
			Name: "Test Buyer",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Testowa 2 00-002 Warszawa",
			},
			JST: "2",
			GV:  "2",
		},
		Inv: &ksef.Inv{
			CurrencyCode:        "PLN",
			IssueDate:           "2026-01-15",
			SequentialNumber:    "ZAL-001",
			InvoiceType:         "ZAL",
			StandardRateNetSale: "1000.00",
			StandardRateTax:     "230.00",
			TotalAmountDue:      "1230.00",
			Annotations: &ksef.Annotations{
				CashAccounting:                      "2",
				SelfBilling:                         "2",
				ReverseCharge:                       "2",
				SplitPaymentMechanism:               "2",
				SimplifiedProcedureBySecondTaxpayer: "2",
				TaxExemption:                        &ksef.TaxExemption{NoExemption: "1"},
				NewTransportMeans:                   &ksef.NewTransportMeans{NoNewTransportMeans: "1"},
				MarginScheme:                        &ksef.MarginScheme{NoMarginScheme: "1"},
			},
			Order: &ksef.Order{
				OrderAmount: "5000.00",
				LineItems: []*ksef.OrderLine{
					{
						LineNumber:    1,
						Name:          "Widget A",
						NetPriceTotal: "3000.00",
						TaxValue:      "690.00",
						VATRate:       "23",
					},
					{
						LineNumber:    2,
						Name:          "Widget B",
						NetPriceTotal: "2000.00",
						TaxValue:      "460.00",
						VATRate:       "23",
					},
				},
			},
			TransactionConditions: &ksef.TransactionConditions{
				Orders: []*ksef.OrderRef{
					{Date: "2026-01-10", Number: "PO-12345"},
				},
			},
			Payment: &ksef.Payment{
				PaidMarker:  "1",
				PaymentDate: "2026-01-15",
				PaymentMean: "6",
				DueDates: []*ksef.DueDate{
					{Date: "2026-01-15"},
				},
				BankAccounts: []*ksef.BankAccount{
					{AccountNumber: "PL61109010140000071219812874", SWIFT: "WBKPPLPP"},
				},
			},
		},
	}
}

func TestPrepaymentBypass(t *testing.T) {
	t.Run("sets bypass tag for ZAL without lines", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.True(t, inv.HasTags(tax.TagBypass))
		assert.True(t, inv.HasTags(tax.TagPartial))
		assert.Empty(t, inv.Lines)
	})

	t.Run("does not set bypass for ZAL with FaWiersz lines", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Lines = []*ksef.Line{
			{
				LineNumber:   1,
				Name:         "Line item",
				Quantity:     "1",
				NetUnitPrice: "1000.00",
				VATRate:      "23",
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.False(t, inv.HasTags(tax.TagBypass))
		assert.Len(t, inv.Lines, 1)
	})

	t.Run("does not set bypass for standard VAT invoice with lines", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.InvoiceType = "VAT"
		doc.Inv.Order = nil
		doc.Inv.Lines = []*ksef.Line{
			{LineNumber: 1, Name: "Item", Quantity: "1", NetUnitPrice: "1000.00", VATRate: "23"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.False(t, inv.HasTags(tax.TagBypass))
	})
}

func TestParsePrepaymentTotals(t *testing.T) {
	t.Run("builds totals from standard rate fields", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Totals)

		assert.Equal(t, "1000.00", inv.Totals.Sum.String())
		assert.Equal(t, "1000.00", inv.Totals.Total.String())
		assert.Equal(t, "230.00", inv.Totals.Tax.String())
		assert.Equal(t, "1230.00", inv.Totals.TotalWithTax.String())
		assert.Equal(t, "1230.00", inv.Totals.Payable.String())
	})

	t.Run("sets tax categories with correct rate", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Totals.Taxes)
		require.Len(t, inv.Totals.Taxes.Categories, 1)

		cat := inv.Totals.Taxes.Categories[0]
		assert.Equal(t, tax.CategoryVAT, cat.Code)
		require.Len(t, cat.Rates, 1)

		rate := cat.Rates[0]
		assert.Equal(t, "1000.00", rate.Base.String())
		assert.Equal(t, "230.00", rate.Amount.String())
		assert.Equal(t, cbc.Code("1"), rate.Ext.Get(favat.ExtKeyTaxCategory))
		require.NotNil(t, rate.Percent)
		assert.Equal(t, "23.0%", rate.Percent.String())
	})

	t.Run("handles multiple tax categories", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.ReducedRateNetSale = "500.00"
		doc.Inv.ReducedRateTax = "40.00"
		doc.Inv.TotalAmountDue = "1770.00"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Totals.Taxes)

		cat := inv.Totals.Taxes.Categories[0]
		require.Len(t, cat.Rates, 2)

		assert.Equal(t, "1500.00", inv.Totals.Sum.String())
		assert.Equal(t, "270.00", inv.Totals.Tax.String())
		assert.Equal(t, "1770.00", inv.Totals.TotalWithTax.String())
		assert.Equal(t, "1770.00", inv.Totals.Payable.String())
	})

	t.Run("handles exempt category without percent", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.StandardRateNetSale = ""
		doc.Inv.StandardRateTax = ""
		doc.Inv.TaxExemptNetSale = "1000.00"
		doc.Inv.TotalAmountDue = "1000.00"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Totals.Taxes)

		cat := inv.Totals.Taxes.Categories[0]
		require.Len(t, cat.Rates, 1)

		rate := cat.Rates[0]
		assert.Equal(t, tax.KeyExempt, rate.Key)
		assert.Nil(t, rate.Percent)
		assert.True(t, rate.Amount.IsZero())
		assert.Equal(t, "1000.00", rate.Base.String())
	})

	t.Run("includes advances and due from payment", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Totals.Advances)
		require.NotNil(t, inv.Totals.Due)

		assert.Equal(t, "1230.00", inv.Totals.Advances.String())
		assert.Equal(t, "0.00", inv.Totals.Due.String())
	})

	t.Run("no advances when no payment", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Payment = nil

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.Nil(t, inv.Totals.Advances)
		assert.Nil(t, inv.Totals.Due)
	})
}

func TestParseOrderingLines(t *testing.T) {
	t.Run("skips when no order data", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Order = nil
		doc.Inv.TransactionConditions = nil

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// Ordering may still exist from period parsing, but no purchases
		if inv.Ordering != nil {
			assert.Empty(t, inv.Ordering.Purchases)
		}
	})

	t.Run("parses Zamowienia without Zamowienie", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Order = nil // no Zamowienie block

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Ordering)
		require.Len(t, inv.Ordering.Purchases, 1)

		ref := inv.Ordering.Purchases[0]
		assert.Equal(t, cbc.Code("PO-12345"), ref.Code)
		require.NotNil(t, ref.IssueDate)
		assert.Equal(t, "2026-01-10", ref.IssueDate.String())
		// No payable or tax since there's no Zamowienie order block
		assert.Nil(t, ref.Payable)
		assert.Nil(t, ref.Tax)
	})

	t.Run("creates purchase ref with order number and date", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Ordering)
		require.Len(t, inv.Ordering.Purchases, 1)

		ref := inv.Ordering.Purchases[0]
		assert.Equal(t, cbc.Code("PO-12345"), ref.Code)
		require.NotNil(t, ref.IssueDate)
		assert.Equal(t, "2026-01-10", ref.IssueDate.String())
	})

	t.Run("sets code to unknown when no order number", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.TransactionConditions.Orders[0].Number = ""

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		ref := inv.Ordering.Purchases[0]
		assert.Equal(t, cbc.Code("unknown"), ref.Code)
	})

	t.Run("sets payable from order amount", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		ref := inv.Ordering.Purchases[0]
		require.NotNil(t, ref.Payable)
		assert.Equal(t, "5000.00", ref.Payable.String())
	})

	t.Run("builds one tax rate per order line", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		ref := inv.Ordering.Purchases[0]
		require.NotNil(t, ref.Tax)
		require.Len(t, ref.Tax.Categories, 1)
		require.Len(t, ref.Tax.Categories[0].Rates, 2)

		rate0 := ref.Tax.Categories[0].Rates[0]
		assert.Equal(t, "3000.00", rate0.Base.String())
		assert.Equal(t, "690.00", rate0.Amount.String())
		require.NotNil(t, rate0.Percent)
		assert.Equal(t, "23.0%", rate0.Percent.String())

		rate1 := ref.Tax.Categories[0].Rates[1]
		assert.Equal(t, "2000.00", rate1.Base.String())
		assert.Equal(t, "460.00", rate1.Amount.String())

		assert.Equal(t, "1150.00", ref.Tax.Sum.String())
	})

	t.Run("concatenates order line descriptions", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		ref := inv.Ordering.Purchases[0]
		assert.Equal(t, "Widget A, Widget B", ref.Description)
	})

	t.Run("handles order line without VAT rate", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Order.LineItems = []*ksef.OrderLine{
			{
				LineNumber:    1,
				Name:          "Exempt Item",
				NetPriceTotal: "100.00",
				TaxValue:      "",
				VATRate:       "",
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		ref := inv.Ordering.Purchases[0]
		require.NotNil(t, ref.Tax)

		rate := ref.Tax.Categories[0].Rates[0]
		assert.Equal(t, "100.00", rate.Base.String())
		assert.Nil(t, rate.Percent)
		assert.True(t, rate.Amount.IsZero())
	})
}

func TestParsePaidInFull(t *testing.T) {
	t.Run("creates advance when Zaplacono=1", func(t *testing.T) {
		doc := testPrepaymentDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Payment)
		require.Len(t, inv.Payment.Advances, 1)

		adv := inv.Payment.Advances[0]
		assert.Equal(t, "1230.00", adv.Amount.String())
		assert.Equal(t, "Advance payment", adv.Description)
		assert.Equal(t, pay.MeansKeyCreditTransfer, adv.Key)
		require.NotNil(t, adv.Date)
		assert.Equal(t, "2026-01-15", adv.Date.String())
		assert.Equal(t, cbc.Code("6"), adv.Ext.Get(favat.ExtKeyPaymentMeans))
	})

	t.Run("does not create advance when Zaplacono is not 1", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Payment.PaidMarker = ""

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// No advances since PaidMarker is empty and no AdvancePayments
		if inv.Payment != nil {
			assert.Empty(t, inv.Payment.Advances)
		}
	})

	t.Run("skips Zaplacono advance when AdvancePayments exist", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Payment.AdvancePayments = []*ksef.AdvancePayment{
			{PaymentAmount: "500.00", PaymentDate: "2026-01-10"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// Should use the AdvancePayments, not create from Zaplacono
		require.Len(t, inv.Payment.Advances, 1)
		assert.Equal(t, "500.00", inv.Payment.Advances[0].Amount.String())
	})
}

func TestParseTermDescription(t *testing.T) {
	t.Run("stores TerminOpis as notes without due date", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Payment.DueDates = []*ksef.DueDate{
			{
				TermDescription: &ksef.TermDescription{
					Quantity:      14,
					Unit:          "DNI",
					StartingEvent: "od daty wystawienia faktury",
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Payment)
		require.NotNil(t, inv.Payment.Terms)
		assert.Empty(t, inv.Payment.Terms.DueDates)
		assert.Equal(t, "14 DNI od daty wystawienia faktury", inv.Payment.Terms.Notes)
	})

	t.Run("explicit Termin date sets due date without percent", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Payment.DueDates = []*ksef.DueDate{
			{Date: "2026-03-01"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Payment)
		require.NotNil(t, inv.Payment.Terms)
		require.Len(t, inv.Payment.Terms.DueDates, 1)

		assert.Equal(t, "2026-03-01", inv.Payment.Terms.DueDates[0].Date.String())
		assert.Equal(t, "100%", inv.Payment.Terms.DueDates[0].Percent.String())
	})

	t.Run("both Termin and TerminOpis sets due date and notes", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Payment.DueDates = []*ksef.DueDate{
			{
				Date: "2026-03-13",
				TermDescription: &ksef.TermDescription{
					Quantity:      14,
					Unit:          "DNI",
					StartingEvent: "od daty wystawienia faktury",
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotNil(t, inv.Payment)
		require.NotNil(t, inv.Payment.Terms)
		require.Len(t, inv.Payment.Terms.DueDates, 1)

		assert.Equal(t, "2026-03-13", inv.Payment.Terms.DueDates[0].Date.String())
		assert.Equal(t, "100%", inv.Payment.Terms.DueDates[0].Percent.String())
		assert.Equal(t, "14 DNI od daty wystawienia faktury", inv.Payment.Terms.Notes)
	})

	t.Run("no date and no TerminOpis does not set terms", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.Payment.DueDates = []*ksef.DueDate{
			{}, // empty due date entry
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		// Payment may exist (due to advances), but terms should be nil
		if inv.Payment != nil {
			assert.Nil(t, inv.Payment.Terms)
		}
	})
}

func TestAdditionalDescriptionCodeValidation(t *testing.T) {
	t.Run("valid code is used as note code", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.AdditionalDescription = []*ksef.AdditionalDescriptionLine{
			{Key: "general", Value: "Some note"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotEmpty(t, inv.Notes)

		assert.Equal(t, cbc.Code("general"), inv.Notes[0].Code)
		assert.Equal(t, "Some note", inv.Notes[0].Text)
	})

	t.Run("invalid code is merged into text", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.AdditionalDescription = []*ksef.AdditionalDescriptionLine{
			{Key: "Numer wewnętrzny zamówienia", Value: "12345"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		require.NotEmpty(t, inv.Notes)

		assert.Equal(t, cbc.Code(""), inv.Notes[0].Code)
		assert.Equal(t, "Numer wewnętrzny zamówienia: 12345", inv.Notes[0].Text)
	})
}

func TestParseAdditionalDescriptions(t *testing.T) {
	// Helper: standard VAT invoice with 2 lines.
	stdDoc := func() *ksef.Invoice {
		return &ksef.Invoice{
			Seller: &ksef.Seller{
				NIP:  "1234567890",
				Name: "Test Supplier",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 1 00-001 Warszawa",
				},
			},
			Buyer: &ksef.Buyer{
				NIP:  "9876543210",
				Name: "Test Buyer",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 2 00-002 Warszawa",
				},
				JST: "2",
				GV:  "2",
			},
			Inv: &ksef.Inv{
				CurrencyCode:        "PLN",
				IssueDate:           "2026-01-20",
				SequentialNumber:    "FV-001",
				TotalAmountDue:      "246.00",
				InvoiceType:         "VAT",
				StandardRateNetSale: "200.00",
				StandardRateTax:     "46.00",
				Annotations: &ksef.Annotations{
					CashAccounting:                      "2",
					SelfBilling:                         "2",
					ReverseCharge:                       "2",
					SplitPaymentMechanism:               "2",
					SimplifiedProcedureBySecondTaxpayer: "2",
					TaxExemption:                        &ksef.TaxExemption{NoExemption: "1"},
					NewTransportMeans:                   &ksef.NewTransportMeans{NoNewTransportMeans: "1"},
					MarginScheme:                        &ksef.MarginScheme{NoMarginScheme: "1"},
				},
				Lines: []*ksef.Line{
					{LineNumber: 1, Name: "Item A", Quantity: "1", NetUnitPrice: "100.00", VATRate: "23"},
					{LineNumber: 2, Name: "Item B", Quantity: "1", NetUnitPrice: "100.00", VATRate: "23"},
				},
			},
		}
	}

	t.Run("NrWiersza routes note to correct line", func(t *testing.T) {
		doc := stdDoc()
		doc.Inv.AdditionalDescription = []*ksef.AdditionalDescriptionLine{
			{LineNumber: "1", Key: "batch", Value: "LOT-001"},
			{LineNumber: "2", Key: "batch", Value: "LOT-002"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// No invoice-level notes
		assert.Empty(t, inv.Notes)

		// Line 1 should have a note
		require.Len(t, inv.Lines, 2)
		require.Len(t, inv.Lines[0].Notes, 1)
		assert.Equal(t, cbc.Code("batch"), inv.Lines[0].Notes[0].Code)
		assert.Equal(t, "LOT-001", inv.Lines[0].Notes[0].Text)

		// Line 2 should have a note
		require.Len(t, inv.Lines[1].Notes, 1)
		assert.Equal(t, cbc.Code("batch"), inv.Lines[1].Notes[0].Code)
		assert.Equal(t, "LOT-002", inv.Lines[1].Notes[0].Text)
	})

	t.Run("without NrWiersza stays as invoice note", func(t *testing.T) {
		doc := stdDoc()
		doc.Inv.AdditionalDescription = []*ksef.AdditionalDescriptionLine{
			{Key: "general", Value: "Invoice-level note"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		require.Len(t, inv.Notes, 1)
		assert.Equal(t, cbc.Code("general"), inv.Notes[0].Code)
		assert.Equal(t, "Invoice-level note", inv.Notes[0].Text)

		// No line notes
		for _, line := range inv.Lines {
			assert.Empty(t, line.Notes)
		}
	})

	t.Run("NrWiersza referencing non-existent line falls back to invoice", func(t *testing.T) {
		doc := stdDoc()
		doc.Inv.AdditionalDescription = []*ksef.AdditionalDescriptionLine{
			{LineNumber: "99", Key: "orphan", Value: "No such line"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// Should fall back to invoice-level note
		require.Len(t, inv.Notes, 1)
		assert.Equal(t, cbc.Code("orphan"), inv.Notes[0].Code)

		// No line notes
		for _, line := range inv.Lines {
			assert.Empty(t, line.Notes)
		}
	})

	t.Run("mixed NrWiersza and invoice-level notes", func(t *testing.T) {
		doc := stdDoc()
		doc.Inv.AdditionalDescription = []*ksef.AdditionalDescriptionLine{
			{Key: "general", Value: "Top-level note"},
			{LineNumber: "1", Key: "detail", Value: "Line 1 detail"},
			{LineNumber: "2", Key: "detail", Value: "Line 2 detail"},
			{Key: "footer", Value: "Another top-level"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// Invoice-level: 2 notes
		require.Len(t, inv.Notes, 2)
		assert.Equal(t, cbc.Code("general"), inv.Notes[0].Code)
		assert.Equal(t, cbc.Code("footer"), inv.Notes[1].Code)

		// Each line: 1 note
		require.Len(t, inv.Lines[0].Notes, 1)
		assert.Equal(t, "Line 1 detail", inv.Lines[0].Notes[0].Text)
		require.Len(t, inv.Lines[1].Notes, 1)
		assert.Equal(t, "Line 2 detail", inv.Lines[1].Notes[0].Text)
	})
}

func TestNewFavatInvLineNotes(t *testing.T) {
	baseInvoice := func() *bill.Invoice {
		pct23 := num.MakePercentage(230, 3)
		return &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{Country: l10n.PL.Tax()},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: num.MakeAmount(1, 0),
					Item: &org.Item{
						Name:  "Item A",
						Price: num.NewAmount(10000, 2),
					},
					Total: num.NewAmount(10000, 2),
					Taxes: tax.Set{
						{
							Category: tax.CategoryVAT,
							Percent:  &pct23,
							Ext:      tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyTaxCategory: "1"}),
						},
					},
				},
			},
			Totals: &bill.Totals{
				Taxes: &tax.Total{},
			},
		}
	}

	t.Run("line notes exported as DodatkowyOpis with NrWiersza", func(t *testing.T) {
		inv := baseInvoice()
		inv.Lines[0].Notes = []*org.Note{
			{Key: "batch", Text: "LOT-001"},
		}

		ksefInv := ksef.NewFavatInv(inv)

		require.Len(t, ksefInv.AdditionalDescription, 1)
		assert.Equal(t, "1", ksefInv.AdditionalDescription[0].LineNumber)
		assert.Equal(t, "batch", ksefInv.AdditionalDescription[0].Key)
		assert.Equal(t, "LOT-001", ksefInv.AdditionalDescription[0].Value)
	})

	t.Run("invoice and line notes combined", func(t *testing.T) {
		inv := baseInvoice()
		inv.Notes = []*org.Note{
			{Key: "general", Text: "Invoice note"},
		}
		inv.Lines[0].Notes = []*org.Note{
			{Key: "detail", Text: "Line note"},
		}

		ksefInv := ksef.NewFavatInv(inv)

		require.Len(t, ksefInv.AdditionalDescription, 2)
		// Invoice notes come first (no NrWiersza)
		assert.Empty(t, ksefInv.AdditionalDescription[0].LineNumber)
		assert.Equal(t, "general", ksefInv.AdditionalDescription[0].Key)
		// Line notes come after (with NrWiersza)
		assert.Equal(t, "1", ksefInv.AdditionalDescription[1].LineNumber)
		assert.Equal(t, "detail", ksefInv.AdditionalDescription[1].Key)
	})
}

func TestIsPrepaymentType(t *testing.T) {
	// Prepayment types: test via ToGOBL that bypass is set
	t.Run("ZAL sets bypass", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.InvoiceType = "ZAL"
		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		assert.True(t, inv.HasTags(tax.TagBypass))
	})

	t.Run("KOR_ZAL sets bypass", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.InvoiceType = "KOR_ZAL"
		doc.Inv.CorrectedInv = []*ksef.CorrectedInv{
			{SequentialNumber: "ZAL-001", IssueDate: "2026-01-15"},
		}
		inv, err := doc.ToGOBL()
		require.NoError(t, err)
		assert.True(t, inv.HasTags(tax.TagBypass))
	})

	// Non-prepayment types: test via ToGOBL with lines (so totals calculate)
	nonPrepayment := []struct {
		name    string
		invType string
	}{
		{"VAT", "VAT"},
		{"KOR", "KOR"}, // credit note needs matching TotalAmountDue
		{"ROZ", "ROZ"},
	}
	for _, tt := range nonPrepayment {
		t.Run(tt.name+" does not set bypass", func(t *testing.T) {
			doc := testPrepaymentDoc()
			doc.Inv.InvoiceType = tt.invType
			doc.Inv.StandardRateNetSale = "1000.00"
			doc.Inv.StandardRateTax = "230.00"
			doc.Inv.TotalAmountDue = "1230.00"
			doc.Inv.Order = nil
			doc.Inv.Payment = nil
			if tt.invType == "KOR" {
				doc.Inv.Lines = []*ksef.Line{
					{LineNumber: 1, Name: "Item", Quantity: "-1", NetUnitPrice: "1000.00", VATRate: "23"},
				}
				doc.Inv.TotalAmountDue = "-1230.00"
				doc.Inv.CorrectedInv = []*ksef.CorrectedInv{
					{SequentialNumber: "FV-001", IssueDate: "2026-01-01"},
				}
			} else {
				doc.Inv.Lines = []*ksef.Line{
					{LineNumber: 1, Name: "Item", Quantity: "1", NetUnitPrice: "1000.00", VATRate: "23"},
				}
			}
			inv, err := doc.ToGOBL()
			require.NoError(t, err)
			assert.False(t, inv.HasTags(tax.TagBypass))
		})
	}
}

// Verify the full prepayment parse flow via ToGOBL.
func TestPrepaymentEndToEnd(t *testing.T) {
	t.Run("KOR_ZAL sets bypass and credit note type", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.InvoiceType = "KOR_ZAL"
		doc.Inv.CorrectedInv = []*ksef.CorrectedInv{
			{SequentialNumber: "ZAL-001", IssueDate: "2026-01-15"},
		}
		doc.Inv.CorrectionReason = "Wrong amount"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.True(t, inv.HasTags(tax.TagBypass))
		assert.True(t, inv.HasTags(tax.TagPartial))
		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		require.Len(t, inv.Preceding, 1)
		assert.Equal(t, cbc.Code("ZAL-001"), inv.Preceding[0].Code)
	})

	t.Run("KOR_ZAL inverts negative totals for credit note", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.InvoiceType = "KOR_ZAL"
		doc.Inv.CorrectedInv = []*ksef.CorrectedInv{
			{SequentialNumber: "ZAL-001", IssueDate: "2026-01-15"},
		}
		// KSeF credit notes have negative P_13/P_14/P_15 values
		doc.Inv.StandardRateNetSale = "-1000.00"
		doc.Inv.StandardRateTax = "-230.00"
		doc.Inv.TotalAmountDue = "-1230.00"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.True(t, inv.HasTags(tax.TagBypass))
		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)

		// Totals should be positive (GOBL convention)
		assert.Equal(t, "1000.00", inv.Totals.Sum.String())
		assert.Equal(t, "230.00", inv.Totals.Tax.String())
		assert.Equal(t, "1230.00", inv.Totals.TotalWithTax.String())
		assert.Equal(t, "1230.00", inv.Totals.Payable.String())

		// Tax rate amounts should also be positive
		require.NotNil(t, inv.Totals.Taxes)
		rate := inv.Totals.Taxes.Categories[0].Rates[0]
		assert.Equal(t, "1000.00", rate.Base.String())
		assert.Equal(t, "230.00", rate.Amount.String())
	})

	t.Run("zero rate prepayment sets correct totals", func(t *testing.T) {
		doc := testPrepaymentDoc()
		doc.Inv.StandardRateNetSale = ""
		doc.Inv.StandardRateTax = ""
		doc.Inv.ZeroTaxExceptIntraCommunityNetSale = "1000.00"
		doc.Inv.TotalAmountDue = "1000.00"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.Equal(t, "1000.00", inv.Totals.Sum.String())
		assert.True(t, inv.Totals.Tax.IsZero())
		assert.Equal(t, "1000.00", inv.Totals.Payable.String())

		rate := inv.Totals.Taxes.Categories[0].Rates[0]
		assert.Equal(t, tax.KeyZero, rate.Key)
		require.NotNil(t, rate.Percent)
		assert.Equal(t, "0.0%", rate.Percent.String())
	})
}

// testSettlementDoc builds a minimal ROZ settlement invoice for testing.
func testSettlementDoc() *ksef.Invoice {
	return &ksef.Invoice{
		Seller: &ksef.Seller{
			NIP:  "1234567890",
			Name: "Test Supplier",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Testowa 1 00-001 Warszawa",
			},
		},
		Buyer: &ksef.Buyer{
			NIP:  "9876543210",
			Name: "Test Buyer",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Testowa 2 00-002 Warszawa",
			},
			JST: "2",
			GV:  "2",
		},
		Inv: &ksef.Inv{
			CurrencyCode:     "PLN",
			IssueDate:        "2026-01-20",
			SequentialNumber: "ROZ-001",
			InvoiceType:      "ROZ",
			// Lines sum to 12300.00 (10000 net + 2300 VAT)
			StandardRateNetSale: "8000.00",
			StandardRateTax:     "1840.00",
			TotalAmountDue:      "6150.00", // P_15: remaining after advance
			Annotations: &ksef.Annotations{
				CashAccounting:                      "2",
				SelfBilling:                         "2",
				ReverseCharge:                       "2",
				SplitPaymentMechanism:               "2",
				SimplifiedProcedureBySecondTaxpayer: "2",
				TaxExemption:                        &ksef.TaxExemption{NoExemption: "1"},
				NewTransportMeans:                   &ksef.NewTransportMeans{NoNewTransportMeans: "1"},
				MarginScheme:                        &ksef.MarginScheme{NoMarginScheme: "1"},
			},
			AdvanceInvoices: []*ksef.AdvanceInvoiceRef{
				{KSeFAdvanceInvoiceNo: "1234567890-20260101-ABC123-01"},
			},
			Lines: []*ksef.Line{
				{
					LineNumber:   1,
					Name:         "Project Completion",
					Quantity:     "1",
					NetUnitPrice: "10000.00",
					VATRate:      "23",
				},
			},
		},
	}
}

func TestDeriveSettlementAdvances(t *testing.T) {
	t.Run("ROZ with advance refs derives advance from Payable - P_15", func(t *testing.T) {
		doc := testSettlementDoc()
		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// Should have an advance
		require.NotNil(t, inv.Payment)
		require.Len(t, inv.Payment.Advances, 1)

		// Advance = Payable(12300) - P_15(6150) = 6150
		assert.Equal(t, "6150.00", inv.Payment.Advances[0].Amount.String())
		assert.Equal(t, "Payment 1234567890-20260101-ABC123-01", inv.Payment.Advances[0].Description)
		assert.Equal(t, "1234567890-20260101-ABC123-01", inv.Payment.Advances[0].Ref)

		// Due should equal P_15
		require.NotNil(t, inv.Totals.Due)
		assert.Equal(t, "6150.00", inv.Totals.Due.String())
	})

	t.Run("fully prepaid settlement (P_15=0)", func(t *testing.T) {
		doc := testSettlementDoc()
		doc.Inv.TotalAmountDue = "0.00"
		doc.Inv.StandardRateNetSale = "0"
		doc.Inv.StandardRateTax = "0"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		require.NotNil(t, inv.Payment)
		require.Len(t, inv.Payment.Advances, 1)

		// Advance should equal the full payable
		assert.Equal(t, "12300.00", inv.Payment.Advances[0].Amount.String())

		// Due should be 0
		require.NotNil(t, inv.Totals.Due)
		assert.Equal(t, "0.00", inv.Totals.Due.String())
	})

	t.Run("settlement with ZaplataCzesciowa is no-op", func(t *testing.T) {
		doc := testSettlementDoc()
		doc.Inv.Payment = &ksef.Payment{
			AdvancePayments: []*ksef.AdvancePayment{
				{
					PaymentAmount: "6150.00",
					PaymentDate:   "2026-01-20",
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// Should use the explicit ZaplataCzesciowa advance, not derive
		require.NotNil(t, inv.Payment)
		require.Len(t, inv.Payment.Advances, 1)
		assert.Equal(t, "6150.00", inv.Payment.Advances[0].Amount.String())
		// Ref should be empty (came from ZaplataCzesciowa, not derived)
		assert.Empty(t, inv.Payment.Advances[0].Ref)
	})

	t.Run("non-settlement invoice is no-op", func(t *testing.T) {
		doc := testSettlementDoc()
		doc.Inv.InvoiceType = "VAT"
		// For VAT type, P_15 = Payable (no advance deduction)
		doc.Inv.TotalAmountDue = "12300.00"
		doc.Inv.StandardRateNetSale = "10000.00"
		doc.Inv.StandardRateTax = "2300.00"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// Should not derive advances for VAT type
		if inv.Payment != nil {
			assert.Empty(t, inv.Payment.Advances)
		}
	})

	t.Run("settlement without advance refs is no-op", func(t *testing.T) {
		doc := testSettlementDoc()
		doc.Inv.AdvanceInvoices = nil
		// Without advance refs, P_15 must equal Payable
		doc.Inv.TotalAmountDue = "12300.00"
		doc.Inv.StandardRateNetSale = "10000.00"
		doc.Inv.StandardRateTax = "2300.00"

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		// No advance refs → no derived advances
		if inv.Payment != nil {
			assert.Empty(t, inv.Payment.Advances)
		}
	})

	t.Run("multiple advance refs create separate advances", func(t *testing.T) {
		doc := testSettlementDoc()
		doc.Inv.AdvanceInvoices = []*ksef.AdvanceInvoiceRef{
			{KSeFAdvanceInvoiceNo: "1234567890-20260101-AAA111-01"},
			{KSeFAdvanceInvoiceNo: "1234567890-20260105-BBB222-02"},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		require.NotNil(t, inv.Payment)
		require.Len(t, inv.Payment.Advances, 2)

		// First advance carries the total amount
		assert.Equal(t, "6150.00", inv.Payment.Advances[0].Amount.String())
		assert.Equal(t, "1234567890-20260101-AAA111-01", inv.Payment.Advances[0].Ref)
		assert.Equal(t, "Payment 1234567890-20260101-AAA111-01", inv.Payment.Advances[0].Description)

		// Second advance has zero amount
		assert.Equal(t, "0.00", inv.Payment.Advances[1].Amount.String())
		assert.Equal(t, "1234567890-20260105-BBB222-02", inv.Payment.Advances[1].Ref)
		assert.Equal(t, "Payment 1234567890-20260105-BBB222-02", inv.Payment.Advances[1].Description)
	})
}

func TestAdjustSettlementTotals(t *testing.T) {
	// Helper to build a GOBL settlement invoice for GOBL→KSeF tests.
	// Fields are pre-populated as if Calculate() had already run.
	pct23 := num.MakePercentage(230, 3)
	advDate := cal.DateOf(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	baseSettlementInvoice := func() *bill.Invoice {
		due := num.MakeAmount(615000, 2)     // 6150.00
		advance := num.MakeAmount(615000, 2) // 6150.00
		return &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{
				TaxID: &tax.Identity{Country: l10n.PL.Tax()},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "ROZ",
				}),
			},
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: num.MakeAmount(1, 0),
					Item: &org.Item{
						Name:  "Full order item",
						Price: num.NewAmount(1000000, 2),
					},
					Total: num.NewAmount(1000000, 2),
					Taxes: tax.Set{
						{
							Category: tax.CategoryVAT,
							Percent:  &pct23,
							Ext:      tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyTaxCategory: "1"}),
						},
					},
				},
			},
			Payment: &bill.PaymentDetails{
				Advances: []*pay.Record{
					{
						Date:        &advDate,
						Ref:         "1234567890-20260101-AAA000BBB111-01",
						Description: "Advance payment",
						Amount:      num.MakeAmount(615000, 2),
					},
				},
			},
			Totals: &bill.Totals{
				Sum:   num.MakeAmount(1000000, 2),
				Total: num.MakeAmount(1000000, 2),
				Taxes: &tax.Total{
					Categories: []*tax.CategoryTotal{
						{
							Code: tax.CategoryVAT,
							Rates: []*tax.RateTotal{
								{
									Key:     tax.KeyStandard,
									Base:    num.MakeAmount(1000000, 2),
									Percent: &pct23,
									Amount:  num.MakeAmount(230000, 2),
									Ext:     tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyTaxCategory: "1"}),
								},
							},
							Amount: num.MakeAmount(230000, 2),
						},
					},
					Sum: num.MakeAmount(230000, 2),
				},
				Tax:          num.MakeAmount(230000, 2),
				TotalWithTax: num.MakeAmount(1230000, 2),
				Payable:      num.MakeAmount(1230000, 2),
				Advances:     &advance,
				Due:          &due,
			},
		}
	}

	t.Run("prorates P_13_X/P_14_X to remaining amounts", func(t *testing.T) {
		inv := baseSettlementInvoice()
		inv.Totals.Taxes.Categories[0].Rates[0].Amount = num.MakeAmount(230000, 2)
		ksefInv := ksef.NewFavatInv(inv)

		// P_15 = Due = 6150.00
		assert.Equal(t, "6150.00", ksefInv.TotalAmountDue)

		// P_13_1 should be prorated: 10000 * (6150/12300) = 5000.00
		assert.Equal(t, "5000.00", ksefInv.StandardRateNetSale)

		// P_14_1 should be prorated: 2300 * (6150/12300) = 1150.00
		assert.Equal(t, "1150.00", ksefInv.StandardRateTax)

		// FakturaZaliczkowa should be mapped from advance ref
		require.Len(t, ksefInv.AdvanceInvoices, 1)
		assert.Equal(t, "1234567890-20260101-AAA000BBB111-01", ksefInv.AdvanceInvoices[0].KSeFAdvanceInvoiceNo)

		// Lines should still show full amount
		require.Len(t, ksefInv.Lines, 1)
		assert.Equal(t, "10000.00", ksefInv.Lines[0].NetPriceTotal)
	})

	t.Run("fully prepaid sets tax totals to zero", func(t *testing.T) {
		inv := baseSettlementInvoice()
		inv.Totals.Taxes.Categories[0].Rates[0].Amount = num.MakeAmount(230000, 2)
		zero := num.MakeAmount(0, 2)
		fullPayable := num.MakeAmount(1230000, 2)
		inv.Totals.Due = &zero
		inv.Totals.Advances = &fullPayable

		ksefInv := ksef.NewFavatInv(inv)

		assert.Equal(t, "0.00", ksefInv.TotalAmountDue)
		assert.Equal(t, "0.00", ksefInv.StandardRateNetSale)
		assert.Equal(t, "0.00", ksefInv.StandardRateTax)
	})

	t.Run("no advances does not adjust tax totals", func(t *testing.T) {
		inv := baseSettlementInvoice()
		inv.Totals.Taxes.Categories[0].Rates[0].Amount = num.MakeAmount(230000, 2)
		inv.Payment = nil
		inv.Totals.Due = nil
		inv.Totals.Advances = nil

		ksefInv := ksef.NewFavatInv(inv)

		// Should use Payable since no Due
		assert.Equal(t, "12300.00", ksefInv.TotalAmountDue)
		// Tax totals should be unchanged (full order amounts)
		assert.Equal(t, "10000.00", ksefInv.StandardRateNetSale)
		assert.Equal(t, "2300.00", ksefInv.StandardRateTax)
	})

	t.Run("advance without ref does not emit FakturaZaliczkowa", func(t *testing.T) {
		inv := baseSettlementInvoice()
		inv.Totals.Taxes.Categories[0].Rates[0].Amount = num.MakeAmount(230000, 2)
		inv.Payment.Advances[0].Ref = "" // no KSeF number

		ksefInv := ksef.NewFavatInv(inv)

		// No FakturaZaliczkowa
		assert.Empty(t, ksefInv.AdvanceInvoices)
		// But tax totals should still be prorated
		assert.Equal(t, "5000.00", ksefInv.StandardRateNetSale)
	})

	t.Run("non-settlement with advances does not adjust", func(t *testing.T) {
		inv := baseSettlementInvoice()
		inv.Totals.Taxes.Categories[0].Rates[0].Amount = num.MakeAmount(230000, 2)
		inv.Tax.Ext = inv.Tax.Ext.Set(favat.ExtKeyInvoiceType, "VAT")

		ksefInv := ksef.NewFavatInv(inv)

		// Tax totals should be unchanged for non-settlement
		assert.Equal(t, "10000.00", ksefInv.StandardRateNetSale)
		assert.Equal(t, "2300.00", ksefInv.StandardRateTax)
	})
}

// Ensure ordering is parsed for non-prepayment invoices too.
func TestOrderingOnStandardInvoice(t *testing.T) {
	doc := testPrepaymentDoc()
	doc.Inv.InvoiceType = "VAT"
	doc.Inv.Lines = []*ksef.Line{
		{
			LineNumber:   1,
			Name:         "Item",
			Quantity:     "1",
			NetUnitPrice: "1000.00",
			VATRate:      "23",
		},
	}

	inv, err := doc.ToGOBL()
	require.NoError(t, err)

	// Should still have ordering from Zamowienie
	require.NotNil(t, inv.Ordering)
	require.Len(t, inv.Ordering.Purchases, 1)
	assert.Equal(t, cbc.Code("PO-12345"), inv.Ordering.Purchases[0].Code)

	// But should NOT have bypass tag
	assert.False(t, inv.HasTags(tax.TagBypass))
	// And should have lines
	require.Len(t, inv.Lines, 1)
}

func TestForeignCurrencyExchangeRate(t *testing.T) {
	pct23 := num.MakePercentage(230, 3)
	pct8 := num.MakePercentage(80, 3)

	baseInvoice := func() *bill.Invoice {
		return &bill.Invoice{
			Currency: currency.EUR,
			Supplier: &org.Party{
				TaxID: &tax.Identity{Country: l10n.PL.Tax()},
			},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyInvoiceType: "VAT",
				}),
			},
			ExchangeRates: []*currency.ExchangeRate{
				{
					From:   currency.EUR,
					To:     currency.PLN,
					Amount: num.MakeAmount(43120, 4), // 4.3120
				},
			},
			Totals: &bill.Totals{
				Payable: num.MakeAmount(204000, 2),
				Taxes: &tax.Total{
					Categories: []*tax.CategoryTotal{
						{
							Code: tax.CategoryVAT,
							Rates: []*tax.RateTotal{
								{
									Key:     tax.KeyStandard,
									Base:    num.MakeAmount(100000, 2),
									Percent: &pct23,
									Amount:  num.MakeAmount(23000, 2),
									Ext:     tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyTaxCategory: "1"}),
								},
								{
									Key:     tax.KeyStandard,
									Base:    num.MakeAmount(75000, 2),
									Percent: &pct8,
									Amount:  num.MakeAmount(6000, 2),
									Ext:     tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyTaxCategory: "2"}),
								},
							},
						},
					},
				},
			},
		}
	}

	t.Run("sets KursWalutyZ from exchange rates", func(t *testing.T) {
		inv := baseInvoice()
		ksefInv := ksef.NewFavatInv(inv)

		assert.Equal(t, "4.3120", ksefInv.ExchangeRate)
	})

	t.Run("converts standard rate tax to PLN", func(t *testing.T) {
		inv := baseInvoice()
		ksefInv := ksef.NewFavatInv(inv)

		// 230.00 EUR * 4.3120 = 991.76 PLN
		assert.Equal(t, "991.76", ksefInv.StandardRateTaxConvertedToPln)
	})

	t.Run("converts reduced rate tax to PLN", func(t *testing.T) {
		inv := baseInvoice()
		ksefInv := ksef.NewFavatInv(inv)

		// 60.00 EUR * 4.3120 = 258.72 PLN
		assert.Equal(t, "258.72", ksefInv.ReducedRateTaxConvertedToPln)
	})

	t.Run("does not set PLN conversion for PLN invoices", func(t *testing.T) {
		inv := baseInvoice()
		inv.Currency = currency.PLN
		inv.ExchangeRates = nil

		ksefInv := ksef.NewFavatInv(inv)

		assert.Empty(t, ksefInv.ExchangeRate)
		assert.Empty(t, ksefInv.StandardRateTaxConvertedToPln)
		assert.Empty(t, ksefInv.ReducedRateTaxConvertedToPln)
	})

	t.Run("does not set PLN conversion without exchange rate", func(t *testing.T) {
		inv := baseInvoice()
		inv.ExchangeRates = nil

		ksefInv := ksef.NewFavatInv(inv)

		assert.Empty(t, ksefInv.ExchangeRate)
		assert.Empty(t, ksefInv.StandardRateTaxConvertedToPln)
		assert.Empty(t, ksefInv.ReducedRateTaxConvertedToPln)
	})

	t.Run("does not set P_14_5W for OSS rate", func(t *testing.T) {
		inv := baseInvoice()
		inv.Totals.Taxes.Categories[0].Rates = []*tax.RateTotal{
			{
				Key:    tax.KeyStandard,
				Base:   num.MakeAmount(100000, 2),
				Amount: num.MakeAmount(21000, 2),
				Ext:    tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyTaxCategory: "5"}),
			},
		}

		ksefInv := ksef.NewFavatInv(inv)

		assert.Equal(t, "1000.00", ksefInv.OSSNetSale)
		assert.Equal(t, "210.00", ksefInv.OSSTax)
		// No P_14_5W field exists in FA3 schema
	})
}

func TestParseExchangeRate(t *testing.T) {
	t.Run("KursWalutyZ mapped to ExchangeRates for foreign currency", func(t *testing.T) {
		doc := &ksef.Invoice{
			Seller: &ksef.Seller{
				NIP:  "1234567890",
				Name: "Test Supplier",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 1 00-001 Warszawa",
				},
			},
			Buyer: &ksef.Buyer{
				NIP:  "9876543210",
				Name: "Test Buyer",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 2 00-002 Warszawa",
				},
				JST: "2",
				GV:  "2",
			},
			Inv: &ksef.Inv{
				CurrencyCode:        "EUR",
				IssueDate:           "2026-01-20",
				SequentialNumber:    "FV-001",
				InvoiceType:         "VAT",
				ExchangeRate:        "4.3120",
				StandardRateNetSale: "1000.00",
				StandardRateTax:     "230.00",
				TotalAmountDue:      "1230.00",
				Annotations: &ksef.Annotations{
					CashAccounting:                      "2",
					SelfBilling:                         "2",
					ReverseCharge:                       "2",
					SplitPaymentMechanism:               "2",
					SimplifiedProcedureBySecondTaxpayer: "2",
					TaxExemption:                        &ksef.TaxExemption{NoExemption: "1"},
					NewTransportMeans:                   &ksef.NewTransportMeans{NoNewTransportMeans: "1"},
					MarginScheme:                        &ksef.MarginScheme{NoMarginScheme: "1"},
				},
				Lines: []*ksef.Line{
					{
						LineNumber:   1,
						Name:         "Item",
						Quantity:     "1",
						NetUnitPrice: "1000.00",
						VATRate:      "23",
					},
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.Equal(t, currency.EUR, inv.Currency)
		require.Len(t, inv.ExchangeRates, 1)
		assert.Equal(t, currency.EUR, inv.ExchangeRates[0].From)
		assert.Equal(t, currency.PLN, inv.ExchangeRates[0].To)
		assert.Equal(t, "4.3120", inv.ExchangeRates[0].Amount.String())
	})

	t.Run("no ExchangeRates for PLN invoice", func(t *testing.T) {
		doc := &ksef.Invoice{
			Seller: &ksef.Seller{
				NIP:  "1234567890",
				Name: "Test Supplier",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 1 00-001 Warszawa",
				},
			},
			Buyer: &ksef.Buyer{
				NIP:  "9876543210",
				Name: "Test Buyer",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 2 00-002 Warszawa",
				},
				JST: "2",
				GV:  "2",
			},
			Inv: &ksef.Inv{
				CurrencyCode:        "PLN",
				IssueDate:           "2026-01-20",
				SequentialNumber:    "FV-001",
				InvoiceType:         "VAT",
				ExchangeRate:        "4.3120", // should be ignored for PLN
				StandardRateNetSale: "1000.00",
				StandardRateTax:     "230.00",
				TotalAmountDue:      "1230.00",
				Annotations: &ksef.Annotations{
					CashAccounting:                      "2",
					SelfBilling:                         "2",
					ReverseCharge:                       "2",
					SplitPaymentMechanism:               "2",
					SimplifiedProcedureBySecondTaxpayer: "2",
					TaxExemption:                        &ksef.TaxExemption{NoExemption: "1"},
					NewTransportMeans:                   &ksef.NewTransportMeans{NoNewTransportMeans: "1"},
					MarginScheme:                        &ksef.MarginScheme{NoMarginScheme: "1"},
				},
				Lines: []*ksef.Line{
					{
						LineNumber:   1,
						Name:         "Item",
						Quantity:     "1",
						NetUnitPrice: "1000.00",
						VATRate:      "23",
					},
				},
			},
		}

		inv, err := doc.ToGOBL()
		require.NoError(t, err)

		assert.Equal(t, currency.PLN, inv.Currency)
		assert.Empty(t, inv.ExchangeRates)
	})

	t.Run("does not return error for EUR invoice missing KursWalutyZ", func(t *testing.T) {
		doc := &ksef.Invoice{
			Seller: &ksef.Seller{
				NIP:  "1234567890",
				Name: "Test Supplier",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 1 00-001 Warszawa",
				},
			},
			Buyer: &ksef.Buyer{
				NIP:  "9876543210",
				Name: "Test Buyer",
				Address: &ksef.Address{
					CountryCode: "PL",
					AddressL1:   "ul. Testowa 2 00-002 Warszawa",
				},
				JST: "2",
				GV:  "2",
			},
			Inv: &ksef.Inv{
				CurrencyCode:        "EUR",
				IssueDate:           "2026-01-20",
				SequentialNumber:    "FV-001",
				InvoiceType:         "VAT",
				StandardRateNetSale: "1000.00",
				StandardRateTax:     "230.00",
				TotalAmountDue:      "1230.00",
				Annotations: &ksef.Annotations{
					CashAccounting:                      "2",
					SelfBilling:                         "2",
					ReverseCharge:                       "2",
					SplitPaymentMechanism:               "2",
					SimplifiedProcedureBySecondTaxpayer: "2",
					TaxExemption:                        &ksef.TaxExemption{NoExemption: "1"},
					NewTransportMeans:                   &ksef.NewTransportMeans{NoNewTransportMeans: "1"},
					MarginScheme:                        &ksef.MarginScheme{NoMarginScheme: "1"},
				},
				Lines: []*ksef.Line{
					{
						LineNumber:   1,
						Name:         "Item",
						Quantity:     "1",
						NetUnitPrice: "1000.00",
						VATRate:      "23",
					},
				},
			},
		}

		_, err := doc.ToGOBL()
		require.NoError(t, err)
	})
}

func TestNoteCodeAsKey(t *testing.T) {
	inv := &bill.Invoice{
		Currency: currency.PLN,
		Supplier: &org.Party{
			TaxID: &tax.Identity{Country: l10n.PL.Tax()},
		},
		Tax: &bill.Tax{
			Ext: tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyInvoiceType: "VAT"}),
		},
		Totals: &bill.Totals{Taxes: &tax.Total{}},
		Notes: []*org.Note{
			{Code: "ABC123", Text: "Note with code only"},
		},
	}

	result := ksef.NewFavatInv(inv)

	require.Len(t, result.AdditionalDescription, 1)
	assert.Equal(t, "ABC123", result.AdditionalDescription[0].Key)
}

func TestNewFavatFooter(t *testing.T) {
	baseInvoice := func() *bill.Invoice {
		pct23 := num.MakePercentage(230, 3)
		return &bill.Invoice{
			Currency: currency.PLN,
			Addons:   tax.WithAddons(favat.V3),
			Supplier: &org.Party{
				TaxID: &tax.Identity{Country: l10n.PL.Tax(), Code: "1234567890"},
			},
			IssueDate: cal.MakeDate(2024, 1, 15),
			Lines: []*bill.Line{
				{
					Index:    1,
					Quantity: num.MakeAmount(1, 0),
					Item:     &org.Item{Name: "Test", Price: num.NewAmount(10000, 2)},
					Taxes:    tax.Set{{Category: "VAT", Percent: &pct23}},
				},
			},
		}
	}

	t.Run("returns nil when no footer notes", func(t *testing.T) {
		inv := baseInvoice()
		inv.Notes = []*org.Note{
			{Key: "general", Text: "Regular note"},
		}
		assert.Nil(t, ksef.NewFavatFooter(inv))
	})

	t.Run("returns nil when notes is empty", func(t *testing.T) {
		inv := baseInvoice()
		assert.Nil(t, ksef.NewFavatFooter(inv))
	})

	t.Run("maps footer notes to Stopka", func(t *testing.T) {
		inv := baseInvoice()
		inv.Notes = []*org.Note{
			{Src: ksef.NoteSourceFooter, Text: "Footer line 1"},
			{Src: ksef.NoteSourceFooter, Text: "Footer line 2"},
		}

		footer := ksef.NewFavatFooter(inv)

		require.NotNil(t, footer)
		require.Len(t, footer.Informacje, 2)
		assert.Equal(t, "Footer line 1", footer.Informacje[0].StopkaFaktury)
		assert.Equal(t, "Footer line 2", footer.Informacje[1].StopkaFaktury)
	})

	t.Run("ignores non-footer notes", func(t *testing.T) {
		inv := baseInvoice()
		inv.Notes = []*org.Note{
			{Key: "general", Text: "Regular note"},
			{Src: ksef.NoteSourceFooter, Text: "Footer only"},
		}

		footer := ksef.NewFavatFooter(inv)

		require.NotNil(t, footer)
		require.Len(t, footer.Informacje, 1)
		assert.Equal(t, "Footer only", footer.Informacje[0].StopkaFaktury)
	})
}
