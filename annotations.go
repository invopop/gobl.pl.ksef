package ksef

import (
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// Annotations defines the XML structure for KSeF annotations
type Annotations struct {
	CashAccounting                      string             `xml:"P_16"`
	SelfBilling                         string             `xml:"P_17"`
	ReverseCharge                       string             `xml:"P_18"`
	SplitPaymentMechanism               string             `xml:"P_18A"`
	TaxExemption                        *TaxExemption      `xml:"Zwolnienie,omitempty"`
	NewTransportMeans                   *NewTransportMeans `xml:"NoweSrodkiTransportu,omitempty"`
	SimplifiedProcedureBySecondTaxpayer string             `xml:"P_23"`
	MarginScheme                        *MarginScheme      `xml:"PMarzy,omitempty"`
}

// TaxExemption defines the XML structure for tax exemption details
type TaxExemption struct {
	Marker           string `xml:"P_19,omitempty"`
	PolishLawBasis   string `xml:"P_19A,omitempty"`
	EUDirectiveBasis string `xml:"P_19B,omitempty"`
	OtherLegalBasis  string `xml:"P_19C,omitempty"`
	NoExemption      string `xml:"P_19N,omitempty"`
}

// NewTransportMeans defines the XML structure for new means of transport
type NewTransportMeans struct {
	Marker                 int                      `xml:"P_22,omitempty"`
	Art42Obligation        string                   `xml:"P_42_5,omitempty"`
	NewTransportMeansItems []*NewTransportMeansItem `xml:"NowySrodekTransportu,omitempty"`
	NoNewTransportMeans    string                   `xml:"P_22N,omitempty"`
}

// NewTransportMeansItem defines details for a single new transport means item
type NewTransportMeansItem struct {
	FirstUseDate       string `xml:"P_22A"`
	LineNumber         int    `xml:"P_NrWierszaNST"`
	Brand              string `xml:"P_22BMK,omitempty"`
	Model              string `xml:"P_22BMD,omitempty"`
	Color              string `xml:"P_22BK,omitempty"`
	RegistrationNumber string `xml:"P_22BNR,omitempty"`
	ProductionYear     string `xml:"P_22BRP,omitempty"`
	// For land vehicles
	Mileage       string `xml:"P_22B,omitempty"`
	VIN           string `xml:"P_22B1,omitempty"`
	BodyNumber    string `xml:"P_22B2,omitempty"`
	ChassisNumber string `xml:"P_22B3,omitempty"`
	FrameNumber   string `xml:"P_22B4,omitempty"`
	VehicleType   string `xml:"P_22BT,omitempty"`
	// For watercraft
	OperatingHoursWater string `xml:"P_22C,omitempty"`
	HullNumber          string `xml:"P_22C1,omitempty"`
	// For aircraft
	OperatingHoursAir string `xml:"P_22D,omitempty"`
	FactoryNumber     string `xml:"P_22D1,omitempty"`
}

// MarginScheme defines the XML structure for margin scheme
type MarginScheme struct {
	Marker                        string `xml:"P_PMarzy,omitempty"`
	TravelAgencyMargin            string `xml:"P_PMarzy_2,omitempty"`
	UsedGoodsMargin               string `xml:"P_PMarzy_3_1,omitempty"`
	ArtWorksMargin                string `xml:"P_PMarzy_3_2,omitempty"`
	CollectiblesAndAntiquesMargin string `xml:"P_PMarzy_3_3,omitempty"`
	NoMarginScheme                string `xml:"P_PMarzyN,omitempty"`
}

// newAnnotations sets annotations data
func newAnnotations(invoice *bill.Invoice) *Annotations {
	// default values for the most common case,
	// For fields P_16 to P_18 and field P_23 2 means "no", 1 means "yes".
	// for others 1 means "yes", no value means "no"
	annotations := &Annotations{
		CashAccounting:        "2",
		SelfBilling:           "2",
		ReverseCharge:         "2",
		SplitPaymentMechanism: "2",
		TaxExemption: &TaxExemption{
			NoExemption: "1",
		},
		NewTransportMeans: &NewTransportMeans{
			NoNewTransportMeans: "1",
		},
		SimplifiedProcedureBySecondTaxpayer: "2",
		MarginScheme: &MarginScheme{
			NoMarginScheme: "1",
		},
	}

	if invoice.Tax == nil {
		return annotations
	}

	if invoice.Tax.Ext.Get(favat.ExtKeyCashAccounting) == "1" {
		annotations.CashAccounting = "1"
	}

	if invoice.Tax.Ext.Get(favat.ExtKeySelfBilling) == "1" {
		annotations.SelfBilling = "1"
	}

	if invoice.Tax.Ext.Get(favat.ExtKeyReverseCharge) == "1" {
		annotations.ReverseCharge = "1"
	}

	if invoice.Tax.Ext.Get(favat.ExtKeySplitPayment) == "1" {
		annotations.SplitPaymentMechanism = "1"
	}

	if invoice.Tax.Ext.Get(favat.ExtKeyExemption) != "" {
		// Find the note in notes with key legal
		annotations.TaxExemption = &TaxExemption{
			Marker: "1",
		}
		for _, note := range invoice.Notes {
			if note.Key == org.NoteKeyLegal && note.Src == favat.ExtKeyExemption {
				switch invoice.Tax.Ext.Get(favat.ExtKeyExemption) {
				case "A": // polish law basis
					annotations.TaxExemption.PolishLawBasis = note.Text
				case "B": // EU directive basis
					annotations.TaxExemption.EUDirectiveBasis = note.Text
				case "C": // other legal basis
					annotations.TaxExemption.OtherLegalBasis = note.Text
				}
				break
			}
		}
	}

	if invoice.Tax.Ext.Get(favat.ExtKeyMarginScheme) != "" {
		annotations.MarginScheme = &MarginScheme{
			Marker: "1",
		}
		switch invoice.Tax.Ext.Get(favat.ExtKeyMarginScheme) {
		case "2": // travel agency margin scheme
			annotations.MarginScheme.TravelAgencyMargin = "1"
		case "3.1": // used goods margin scheme
			annotations.MarginScheme.UsedGoodsMargin = "1"
		case "3.2": // art works margin scheme
			annotations.MarginScheme.ArtWorksMargin = "1"
		case "3.3": // collectibles and antiques margin scheme
			annotations.MarginScheme.CollectiblesAndAntiquesMargin = "1"
		}
	}

	return annotations
}

// parseAnnotations converts KSEF annotations to GOBL tax extensions and notes.
func (inv *Inv) parseAnnotations(goblInv *bill.Invoice) {
	if inv.Annotations == nil {
		return
	}

	goblInv.Tax = &bill.Tax{
		Ext: tax.MakeExtensions(),
	}

	// Set invoice type extension
	if inv.InvoiceType != "" {
		goblInv.Tax.Ext = goblInv.Tax.Ext.Set(favat.ExtKeyInvoiceType, cbc.Code(inv.InvoiceType))
	}

	// Cash accounting
	if inv.Annotations.CashAccounting == "1" {
		goblInv.Tax.Ext = goblInv.Tax.Ext.Set(favat.ExtKeyCashAccounting, "1")
	}

	// Self billing
	if inv.Annotations.SelfBilling == "1" {
		goblInv.Tax.Ext = goblInv.Tax.Ext.Set(favat.ExtKeySelfBilling, "1")
		goblInv.Tags.List = append(goblInv.Tags.List, tax.TagSelfBilled)
	}

	// Reverse charge
	if inv.Annotations.ReverseCharge == "1" {
		goblInv.Tax.Ext = goblInv.Tax.Ext.Set(favat.ExtKeyReverseCharge, "1")
		goblInv.Tags.List = append(goblInv.Tags.List, tax.TagReverseCharge)
	}

	// Split payment
	if inv.Annotations.SplitPaymentMechanism == "1" {
		goblInv.Tax.Ext = goblInv.Tax.Ext.Set(favat.ExtKeySplitPayment, "1")
	}

	// Tax exemption
	if inv.Annotations.TaxExemption != nil && inv.Annotations.TaxExemption.Marker == "1" {
		var exemptionCode string
		var exemptionText string

		if inv.Annotations.TaxExemption.PolishLawBasis != "" {
			exemptionCode = "A"
			exemptionText = inv.Annotations.TaxExemption.PolishLawBasis
		} else if inv.Annotations.TaxExemption.EUDirectiveBasis != "" {
			exemptionCode = "B"
			exemptionText = inv.Annotations.TaxExemption.EUDirectiveBasis
		} else if inv.Annotations.TaxExemption.OtherLegalBasis != "" {
			exemptionCode = "C"
			exemptionText = inv.Annotations.TaxExemption.OtherLegalBasis
		}

		if exemptionCode != "" {
			goblInv.Tax.Ext = goblInv.Tax.Ext.Set(favat.ExtKeyExemption, cbc.Code(exemptionCode))

			if goblInv.Notes == nil {
				goblInv.Notes = []*org.Note{}
			}
			goblInv.Notes = append(goblInv.Notes, &org.Note{
				Key:  org.NoteKeyLegal,
				Code: cbc.Code(exemptionCode),
				Src:  favat.ExtKeyExemption,
				Text: exemptionText,
			})
		}
	}

	// Margin scheme
	if inv.Annotations.MarginScheme != nil && inv.Annotations.MarginScheme.Marker == "1" {
		var marginCode string
		if inv.Annotations.MarginScheme.TravelAgencyMargin == "1" {
			marginCode = "2"
		} else if inv.Annotations.MarginScheme.UsedGoodsMargin == "1" {
			marginCode = "3.1"
		} else if inv.Annotations.MarginScheme.ArtWorksMargin == "1" {
			marginCode = "3.2"
		} else if inv.Annotations.MarginScheme.CollectiblesAndAntiquesMargin == "1" {
			marginCode = "3.3"
		}

		if marginCode != "" {
			goblInv.Tax.Ext = goblInv.Tax.Ext.Set(favat.ExtKeyMarginScheme, cbc.Code(marginCode))
		}
	}
}
