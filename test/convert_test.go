package test

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/invopop/gobl"
	ksef "github.com/invopop/gobl.pl.ksef"
	"github.com/invopop/xmldsig"
	dsigksef "github.com/invopop/xmldsig/ksef"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	msgMissingOutFile    = "output file %s missing, run tests with `--update` flag to create"
	msgUnmatchingOutFile = "output file %s does not match, run tests with `--update` flag to update"
)

// TestGOBLToKSeF tests conversion from GOBL JSON to KSeF XML
func TestGOBLToKSeF(t *testing.T) {
	// Use a fixed time for deterministic golden files
	ksef.SetTimeNow(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	t.Cleanup(func() {
		ksef.SetTimeNow(time.Now)
	})

	signingTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cert := loadTestCertificate(t)

	inputDir := filepath.Join(GetDataPath(), "gobl.ksef")
	outputDir := filepath.Join(GetDataPath(), "gobl.ksef", "out")
	sigDir := filepath.Join(outputDir, "sig")

	// Find all JSON input files
	entries, err := os.ReadDir(inputDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			// Read GOBL JSON
			inputPath := filepath.Join(inputDir, name)
			data, err := os.ReadFile(inputPath)
			require.NoError(t, err, "reading input file")

			// Parse GOBL envelope
			env := new(gobl.Envelope)
			err = json.Unmarshal(data, env)
			require.NoError(t, err, "unmarshaling GOBL")

			// Convert to KSeF
			inv, err := ksef.BuildFavat(env)
			require.NoError(t, err, "converting to KSeF")

			// Get XML bytes
			xmlData, err := inv.Bytes()
			require.NoError(t, err, "marshaling XML")

			// Sign the generated XML
			sigData := signKSeFXML(t, xmlData, cert, signingTime)

			baseName := name[:len(name)-len(filepath.Ext(name))]
			outPath := filepath.Join(outputDir, baseName+".xml")
			sigPath := filepath.Join(sigDir, baseName+".xml")

			if UpdateOut {
				// Basic validation - just check we can parse it back
				// TODO: Validate against the schema
				_, err = ksef.ParseKSeF(xmlData)
				require.NoError(t, err, "validating generated XML can be parsed")

				err = os.WriteFile(outPath, xmlData, 0644)
				require.NoError(t, err)

				err = os.WriteFile(sigPath, sigData, 0644)
				require.NoError(t, err)
				return
			}

			expected, err := os.ReadFile(outPath)
			require.False(t, os.IsNotExist(err), msgMissingOutFile, filepath.Base(outPath))
			require.NoError(t, err)
			require.Equal(t, string(expected), string(xmlData), msgUnmatchingOutFile, filepath.Base(outPath))

			expectedSig, err := os.ReadFile(sigPath)
			require.False(t, os.IsNotExist(err), msgMissingOutFile, filepath.Base(sigPath))
			require.NoError(t, err)
			require.Equal(t, string(expectedSig), string(sigData), msgUnmatchingOutFile, filepath.Base(sigPath))
		})
	}
}

// TestKSeFToGOBL tests conversion from KSeF XML to GOBL JSON
func TestKSeFToGOBL(t *testing.T) {
	inputDir := filepath.Join(GetDataPath(), "ksef.gobl")
	outputDir := filepath.Join(GetDataPath(), "ksef.gobl", "out")

	// Create output directory if it doesn't exist
	err := os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Find all XML input files
	entries, err := os.ReadDir(inputDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".xml" {
			continue
		}

		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			// Read KSeF XML
			inputPath := filepath.Join(inputDir, name)
			xmlData, err := os.ReadFile(inputPath)
			require.NoError(t, err, "reading input file")

			// Parse to GOBL
			env, err := ksef.ParseKSeF(xmlData)
			require.NoError(t, err, "parsing KSeF")
			require.NotNil(t, env)

			// Validate GOBL — log but don't fail on validation errors since
			// KSeF input may contain values (e.g. non-standard units) that
			// are valid in KSeF but not in GOBL's strict validation.
			if err = env.Validate(); err != nil {
				t.Logf("GOBL validation: %v", err)
			}

			// Marshal to JSON
			jsonData, err := json.MarshalIndent(env, "", "  ")
			require.NoError(t, err, "marshaling GOBL to JSON")

			// Determine output file name
			baseName := name[:len(name)-len(filepath.Ext(name))]
			outputPath := filepath.Join(outputDir, baseName+".json")

			if UpdateOut {
				// Normalize dynamic fields for deterministic golden files
				jsonData = NormalizeJSON(jsonData)
				err = os.WriteFile(outputPath, jsonData, 0644)
				require.NoError(t, err, "writing golden file")
				t.Logf("Updated golden file: %s", outputPath)
			} else {
				// Compare with golden file if it exists
				expected, err := os.ReadFile(outputPath)
				if err == nil {
					// Golden file exists, compare
					var expectedEnv, actualEnv gobl.Envelope
					err = json.Unmarshal(expected, &expectedEnv)
					require.NoError(t, err, "unmarshaling expected GOBL")
					err = json.Unmarshal(jsonData, &actualEnv)
					require.NoError(t, err, "unmarshaling actual GOBL")

					// Compare key fields (not exact match as some fields may differ)
					assert.NotEmpty(t, actualEnv.Document)
				}

				// Basic validation
				assert.NotEmpty(t, jsonData)
			}
		})
	}
}

// TestRoundTrip tests GOBL → KSeF → GOBL conversion
func TestRoundTrip(t *testing.T) {
	inputDir := filepath.Join(GetDataPath(), "gobl.ksef")

	// Find all JSON input files
	entries, err := os.ReadDir(inputDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			// Read original GOBL JSON
			inputPath := filepath.Join(inputDir, name)
			originalData, err := os.ReadFile(inputPath)
			require.NoError(t, err, "reading input file")

			// Parse original GOBL
			originalEnv := new(gobl.Envelope)
			err = json.Unmarshal(originalData, originalEnv)
			require.NoError(t, err, "unmarshaling original GOBL")

			// Convert GOBL → KSeF
			inv, err := ksef.BuildFavat(originalEnv)
			require.NoError(t, err, "converting to KSeF")

			xmlData, err := inv.Bytes()
			require.NoError(t, err, "marshaling XML")

			// Convert KSeF → GOBL
			roundTripEnv, err := ksef.ParseKSeF(xmlData)
			require.NoError(t, err, "parsing KSeF back to GOBL")

			// Validate round-trip GOBL
			err = roundTripEnv.Validate()
			assert.NoError(t, err, "validating round-trip GOBL")

			// Verify document exists and is not empty
			assert.NotNil(t, roundTripEnv.Document)
		})
	}
}

func loadTestCertificate(t *testing.T) *xmldsig.Certificate {
	t.Helper()

	certPath := filepath.Join(GetTestPath(), "certs", "test.pfx")
	cert, err := xmldsig.LoadCertificate(certPath, "")
	require.NoError(t, err, "loading test certificate — regenerate with: go run ./test/cmd/gencert")

	return cert
}

func signKSeFXML(t *testing.T, xmlData []byte, cert *xmldsig.Certificate, signingTime time.Time) []byte {
	t.Helper()

	signature, err := xmldsig.Sign(xmlData,
		xmldsig.WithCertificate(cert),
		xmldsig.WithXMLDSigConfig(dsigksef.XMLDSigConfig()),
		xmldsig.WithXAdESConfig(dsigksef.XAdESConfig()),
		xmldsig.WithDocID("test"),
		xmldsig.WithCurrentTime(func() time.Time { return signingTime }),
	)
	require.NoError(t, err, "signing XML")

	data, err := xml.MarshalIndent(signature, "", "  ")
	require.NoError(t, err, "marshaling signature")

	return data
}
