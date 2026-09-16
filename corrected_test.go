package ksef_test

import (
	"testing"

	ksef "github.com/invopop/gobl.pl.ksef"
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
)

func TestNewCorrectedInv(t *testing.T) {
	t.Run("creates corrected invoice with series and code", func(t *testing.T) {
		prc := &org.DocumentRef{
			Series: "INV",
			Code:   "001",
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, "INV-001", result.SequentialNumber)
	})

	t.Run("creates corrected invoice without series", func(t *testing.T) {
		prc := &org.DocumentRef{
			Code: "002",
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, "002", result.SequentialNumber)
	})

	t.Run("sets issue date when present", func(t *testing.T) {
		issueDate := cal.MakeDate(2025, 12, 20)
		prc := &org.DocumentRef{
			Code:      "001",
			IssueDate: &issueDate,
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, "2025-12-20", result.IssueDate)
	})

	t.Run("does not set issue date when nil", func(t *testing.T) {
		prc := &org.DocumentRef{
			Code: "001",
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, "", result.IssueDate)
	})

	t.Run("sets KSeF number when stamp present", func(t *testing.T) {
		prc := &org.DocumentRef{
			Code: "001",
			Stamps: []*head.Stamp{
				{
					Provider: favat.StampKSeFNumber,
					Value:    "1234567890-20251220-ABC123-FF",
				},
			},
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, 1, result.KsefNumberPresent)
		assert.Equal(t, "1234567890-20251220-ABC123-FF", result.KsefNumber)
		assert.Equal(t, 0, result.NoKsefNumberPresent)
	})

	t.Run("sets NoKsefNumberPresent when no KSeF stamp", func(t *testing.T) {
		prc := &org.DocumentRef{
			Code:   "001",
			Stamps: []*head.Stamp{},
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, 0, result.KsefNumberPresent)
		assert.Equal(t, "", result.KsefNumber)
		assert.Equal(t, 1, result.NoKsefNumberPresent)
	})

	t.Run("ignores other stamps", func(t *testing.T) {
		prc := &org.DocumentRef{
			Code: "001",
			Stamps: []*head.Stamp{
				{
					Provider: cbc.Key("other-stamp"),
					Value:    "some-value",
				},
			},
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, 0, result.KsefNumberPresent)
		assert.Equal(t, 1, result.NoKsefNumberPresent)
	})

	t.Run("finds KSeF stamp among multiple stamps", func(t *testing.T) {
		prc := &org.DocumentRef{
			Code: "001",
			Stamps: []*head.Stamp{
				{
					Provider: cbc.Key("other-stamp"),
					Value:    "some-value",
				},
				{
					Provider: favat.StampKSeFNumber,
					Value:    "ksef-number-123",
				},
				{
					Provider: cbc.Key("another-stamp"),
					Value:    "another-value",
				},
			},
		}

		result := ksef.NewCorrectedInv(prc)

		assert.Equal(t, 1, result.KsefNumberPresent)
		assert.Equal(t, "ksef-number-123", result.KsefNumber)
	})
}


