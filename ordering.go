package ksef

import (
	"fmt"
	"strconv"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// Order defines the XML structure for KSeF "Zamowienie" (order) field, required for ZAL and KOR_ZAL types
type Order struct {
	OrderAmount string       `xml:"WartoscZamowienia"`
	LineItems   []*OrderLine `xml:"ZamowienieWiersz,omitempty"`
}

// OrderLine defines the XML structure for KSeF item line (element type ZamowienieWiersz, for ZAL and KOR_ZAL type invoices)
type OrderLine struct {
	LineNumber              int    `xml:"NrWierszaZam"`
	UniqueID                string `xml:"UU_IDZ,omitempty"`
	Name                    string `xml:"P_7Z,omitempty"`
	InternalCode            string `xml:"IndeksZ,omitempty"`
	GTIN                    string `xml:"GTINZ,omitempty"`
	PKWiU                   string `xml:"PKWiUZ,omitempty"`
	CN                      string `xml:"CNZ,omitempty"`
	PKOB                    string `xml:"PKOBZ,omitempty"`
	Measure                 string `xml:"P_8AZ,omitempty"`
	Quantity                string `xml:"P_8BZ,omitempty"`
	NetUnitPrice            string `xml:"P_9AZ,omitempty"`
	NetPriceTotal           string `xml:"P_11NettoZ,omitempty"`
	TaxValue                string `xml:"P_11VatZ,omitempty"`
	VATRate                 string `xml:"P_12Z,omitempty"`
	OSSTaxRate              string `xml:"P_12Z_XII,omitempty"` // one stop shop
	Attachment15GoodsMarker int    `xml:"P_12Z_Zal_15,omitempty"`
	SpecialGoodsCode        string `xml:"GTUZ,omitempty"` // values GTU_01 to GTU_13
	Procedure               string `xml:"ProceduraZ,omitempty"`
	ExciseDuty              string `xml:"KwotaAkcyzyZ,omitempty"`
	BeforeCorrectionMarker  int    `xml:"StanPrzedZ,omitempty"`
}

// newInvoicePeriod converts GOBL ordering period to KSeF InvoicePeriod.
// OkresFa requires both P_6_Od and P_6_Do, so a GOBL period carrying only one
// of the two — which the schema has allowed since GOBL v0.505 — is omitted
// rather than emitted as an incomplete element.
func newInvoicePeriod(ordering *bill.Ordering) *InvoicePeriod {
	if ordering == nil || ordering.Period == nil {
		return nil
	}
	if ordering.Period.Start == nil || ordering.Period.End == nil {
		return nil
	}

	return &InvoicePeriod{
		StartDate: ordering.Period.Start.String(),
		EndDate:   ordering.Period.End.String(),
	}
}

// newOrder converts ordering purchases to KSeF Order (Zamowienie).
// This is the inverse of parseOrderingLines(). The Zamowienie element is only
// generated when the purchase reference contains order-level data (payable
// amount or tax breakdown), which is the case for ZAL/KOR_ZAL invoices.
// Simple purchase order code references are handled by newTransactionConditions
// as WarunkiTransakcji/Zamowienia instead.
func newOrder(ordering *bill.Ordering) *Order {
	if ordering == nil || len(ordering.Purchases) == 0 {
		return nil
	}

	ref := ordering.Purchases[0]

	// Only generate Zamowienie when order-level financial data is present
	if ref.Payable == nil && ref.Tax == nil {
		return nil
	}

	// Build Order
	order := &Order{}
	if ref.Payable != nil {
		order.OrderAmount = ref.Payable.String()
	}

	// Build order lines from tax rates
	if ref.Tax != nil && len(ref.Tax.Categories) > 0 {
		cat := ref.Tax.Categories[0]
		for i, rate := range cat.Rates {
			ol := &OrderLine{
				LineNumber: i + 1,
				Name:       ref.Description,
				Quantity:   "1",
			}
			if !rate.Base.IsZero() {
				ol.NetPriceTotal = rate.Base.String()
				ol.NetUnitPrice = rate.Base.String()
			}
			if !rate.Amount.IsZero() {
				ol.TaxValue = rate.Amount.String()
			}
			if rate.Percent != nil {
				ol.VATRate = rate.Percent.Amount().MinimalString()
			}
			order.LineItems = append(order.LineItems, ol)
		}
	}

	return order
}

// parseOrderingLines maps Zamowienie (order) and WarunkiTransakcji data into
// Ordering fields. Zamowienia references are parsed into Purchases, Umowy
// references into Contracts. When a Zamowienie (Order) block is present its
// amount and line-item tax data are merged into the first purchase ref.
func (inv *Inv) parseOrderingLines(goblInv *bill.Invoice) error {
	hasOrder := inv.Order != nil
	hasOrders := inv.TransactionConditions != nil && len(inv.TransactionConditions.Orders) > 0
	hasContracts := inv.TransactionConditions != nil && len(inv.TransactionConditions.Contracts) > 0

	if !hasOrder && !hasOrders && !hasContracts {
		return nil
	}

	if goblInv.Ordering == nil {
		goblInv.Ordering = &bill.Ordering{}
	}

	// Parse Zamowienia (order references) from TransactionConditions
	if hasOrders {
		for _, or := range inv.TransactionConditions.Orders {
			ref := &org.DocumentRef{}
			if or.Date != "" {
				date, err := parseDate(or.Date)
				if err != nil {
					return fmt.Errorf("parsing order date: %w", err)
				}
				ref.IssueDate = &date
			}
			if or.Number != "" {
				ref.Code = cbc.Code(or.Number)
			}
			if ref.Code == "" {
				ref.Code = "unknown"
			}
			goblInv.Ordering.Purchases = append(goblInv.Ordering.Purchases, ref)
		}
	}

	// Parse Umowy (contract references) from TransactionConditions
	if hasContracts {
		for _, c := range inv.TransactionConditions.Contracts {
			ref := &org.DocumentRef{}
			if c.Date != "" {
				date, err := parseDate(c.Date)
				if err != nil {
					return fmt.Errorf("parsing contract date: %w", err)
				}
				ref.IssueDate = &date
			}
			if c.Number != "" {
				ref.Code = cbc.Code(c.Number)
			}
			if ref.Code == "" {
				ref.Code = "unknown"
			}
			goblInv.Ordering.Contracts = append(goblInv.Ordering.Contracts, ref)
		}
	}

	// Parse Zamowienie (Order) data - merge into the first purchase ref
	if hasOrder {
		var ref *org.DocumentRef
		if len(goblInv.Ordering.Purchases) > 0 {
			ref = goblInv.Ordering.Purchases[0]
		} else {
			ref = &org.DocumentRef{Code: "unknown"}
			goblInv.Ordering.Purchases = append(goblInv.Ordering.Purchases, ref)
		}

		// Set payable from order total
		if inv.Order.OrderAmount != "" {
			payable, err := parseAmount(inv.Order.OrderAmount)
			if err != nil {
				return fmt.Errorf("parsing order amount: %w", err)
			}
			ref.Payable = &payable
		}

		// Build tax total and description from order lines (one rate per line)
		if len(inv.Order.LineItems) > 0 {
			var rates []*tax.RateTotal
			var taxSum num.Amount
			var descriptions []string

			for _, ol := range inv.Order.LineItems {
				if ol.Name != "" {
					descriptions = append(descriptions, ol.Name)
				}

				rt := &tax.RateTotal{}

				if ol.NetPriceTotal != "" {
					netAmt, err := parseAmount(ol.NetPriceTotal)
					if err != nil {
						return fmt.Errorf("parsing order line net: %w", err)
					}
					rt.Base = netAmt
				}

				if ol.TaxValue != "" {
					taxAmt, err := parseAmount(ol.TaxValue)
					if err != nil {
						return fmt.Errorf("parsing order line tax: %w", err)
					}
					rt.Amount = taxAmt
					taxSum = taxSum.MatchPrecision(taxAmt)
					taxSum = taxSum.Add(taxAmt)
				}

				if ol.VATRate != "" {
					rateF, err := strconv.ParseFloat(ol.VATRate, 64)
					if err == nil && rateF > 0 {
						pct := num.MakePercentage(int64(rateF*10), 3)
						rt.Percent = &pct
					}
				}

				rates = append(rates, rt)
			}

			if len(descriptions) > 0 {
				ref.Description = descriptions[0]
				for _, d := range descriptions[1:] {
					ref.Description += ", " + d
				}
			}

			if len(rates) > 0 {
				ref.Tax = &tax.Total{
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
		}
	}

	return nil
}
