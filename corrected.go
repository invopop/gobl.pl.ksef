package ksef

import (
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/org"
)

// CorrectedInv defines the XML structure for KSeF correction invoice
type CorrectedInv struct {
	IssueDate           string   `xml:"DataWystFaKorygowanej,omitempty"`
	SequentialNumber    string   `xml:"NrFaKorygowanej,omitempty"`
	CorrectionPeriod    string   `xml:"OkresFaKorygowanej,omitempty"`
	CorrectedInvoiceNo  string   `xml:"NrFaKorygowany,omitempty"`
	KsefNumberPresent   int      `xml:"NrKSeF,omitempty"`
	NoKsefNumberPresent int      `xml:"NrKSeFN,omitempty"`
	KsefNumber          string   `xml:"NrKSeFFaKorygowanej,omitempty"`
	CorrectedSeller     *Seller  `xml:"Podmiot1K,omitempty"`
	CorrectedBuyer      []*Buyer `xml:"Podmiot2K,omitempty"`
}

// NewCorrectedInv gets credit note data from GOBL invoice
func NewCorrectedInv(prc *org.DocumentRef) *CorrectedInv {
	inv := &CorrectedInv{
		SequentialNumber: invoiceNumber(prc.Series, prc.Code),
	}

	if prc.IssueDate != nil {
		inv.IssueDate = prc.IssueDate.String()
	}

	if id := findStamp(prc.Stamps, favat.StampKSeFNumber); id != -1 {
		inv.KsefNumberPresent = 1
		inv.KsefNumber = prc.Stamps[id].Value
	} else {
		inv.NoKsefNumberPresent = 1
	}

	return inv
}

func findStamp(a []*head.Stamp, x cbc.Key) int {
	for i, n := range a {
		if x == n.Provider {
			return i
		}
	}
	return -1
}
