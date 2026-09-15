package test

import (
	"os"
	"strings"
	"testing"

	ksef "github.com/invopop/gobl.ksef"
	"github.com/stretchr/testify/require"
)

// TestConvertAndValidateAll converts all JSON files in test/data to XML
// and validates them against the FA3 schema.
//
// Run without XSD validation:
//
//	go test ./test -run TestConvertAndValidateAll -v
//
// Run with XSD validation (requires libxml2):
//
//	go test -tags xsdvalidate ./test -run TestConvertAndValidateAll -v
func TestConvertAndValidateAll(t *testing.T) {
	dataPath := GetGOBLPath()

	entries, err := os.ReadDir(dataPath)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			// Load and convert
			env, err := LoadTestEnvelope(name)
			require.NoError(t, err, "failed to load envelope")

			doc, err := ksef.BuildFavat(env)
			require.NoError(t, err, "failed to build FA_VAT document")

			data, err := doc.Bytes()
			require.NoError(t, err, "failed to generate XML bytes")

			t.Logf("Generated %d bytes of XML", len(data))

			// Validate against schema
			ValidateAgainstFA3Schema(t, data)
		})
	}
}
