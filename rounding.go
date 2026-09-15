package ksef

import (
	"fmt"

	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// RoundingError indicates that the total calculated from the GOBL invoice does
// not match the total reported by KSeF, and the difference is larger than what
// can be attributed to rounding. When this error is returned, the inv parameter
// passed to AdjustRounding has had the difference applied as a rounding
// adjustment anyway, allowing callers to use it despite the warning.
type RoundingError struct {
	Calculated num.Amount
	Expected   num.Amount
	Diff       num.Amount
	MaxAllowed num.Amount
}

func (e *RoundingError) Error() string {
	return fmt.Sprintf(
		"calculated GOBL total (%s) does not match KSeF total (%s): difference of %s exceeds rounding tolerance of %s",
		e.Calculated.String(), e.Expected.String(), e.Diff.String(), e.MaxAllowed.String(),
	)
}

// AdjustRounding checks and, if needed, adjusts the rounding in the GOBL invoice to match the
// KSEF total amount. KSEF calculates totals by rounding each line and then summing,
// which can lead to a mismatch with the total amount in GOBL.
func AdjustRounding(inv *bill.Invoice, ksefTotalDue string) error {
	// First calculate the GOBL totals
	if err := inv.Calculate(); err != nil {
		return err
	}

	// Parse the KSEF total amount
	expectedTotal, err := parseAmount(ksefTotalDue)
	if err != nil {
		return fmt.Errorf("parsing KSEF total amount: %w", err)
	}

	if inv.Totals == nil {
		// No totals after calculationh. This happens for header-only corrections
		// (e.g. KOR with no lines). If the expected total is also zero,
		// there's nothing to adjust.
		if expectedTotal.IsZero() {
			return nil
		}
		return fmt.Errorf("invoice totals are nil after calculation")
	}

	// Calculate the difference between the expected and the calculated totals.
	// For settlement (ROZ) and prepayment (ZAL) invoices, P_15 represents
	// the remaining amount (Due), not the full Payable. Use Due when it's
	// non-zero, or when both Due and expected are zero (fully-settled ROZ
	// where P_15=0). For standard invoices with partial payments, P_15 =
	// Payable and the partial payment is just tracking info.
	calculatedTotal := inv.Totals.Payable
	if (inv.HasTags(favat.TagSettlement) || inv.HasTags(tax.TagPartial)) && inv.Totals.Due != nil {
		if !inv.Totals.Due.IsZero() || expectedTotal.IsZero() {
			calculatedTotal = *inv.Totals.Due
		}
	}

	diff := expectedTotal.Subtract(calculatedTotal)
	if diff.IsZero() {
		// No difference. No adjustment needed
		return nil
	}

	// Check if the difference can be attributed to rounding
	maxErr := MaxRoundingError(inv)
	if diff.Abs().Compare(maxErr) == 1 {
		// Too much difference. Apply the adjustment anyway and return a warning
		inv.Totals.Rounding = &diff
		return &RoundingError{
			Calculated: calculatedTotal,
			Expected:   expectedTotal,
			Diff:       diff,
			MaxAllowed: maxErr,
		}
	}

	// Apply the rounding adjustment
	inv.Totals.Rounding = &diff

	return nil
}

// MaxRoundingError returns the maximum error that can be attributed to rounding in an invoice.
// It calculates 1 of the smallest subunit of the currency per line.
func MaxRoundingError(inv *bill.Invoice) num.Amount {
	// 1 of the smallest subunit of the currency per line
	subunits := inv.Currency.Def().Subunits
	return num.MakeAmount(int64(len(inv.Lines)), subunits)
}
