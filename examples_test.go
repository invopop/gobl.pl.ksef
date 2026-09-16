package ksef_test

import (
	"testing"

	// Register the Polish KSeF FA_VAT addon so example documents declaring the
	// pl-favat-v3 addon normalize and validate.
	_ "github.com/invopop/gobl.pl.ksef/addon"

	"github.com/invopop/gobl.pl.ksef/test"
	"github.com/invopop/gobl/pkg/examples"
)

// TestExamples converts every document under examples/ to a calculated,
// validated JSON envelope and compares it against its golden output, using the
// shared GOBL example helpers.
//
// The `--update` flag is shared with the conversion golden files (see
// test.UpdateOut), so one run regenerates every golden in the repository.
func TestExamples(t *testing.T) {
	examples.Run(t, "examples", test.UpdateOut)
}
