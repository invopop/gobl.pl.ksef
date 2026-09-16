package main

import (
	"context"
	"fmt"
	"log"
	"os"

	ksef_api "github.com/invopop/gobl.pl.ksef/api"
	"github.com/invopop/xmldsig"
)

func main() {
	// ===== CONFIGURATION - EDIT THIS SECTION =====
	// XML file to send (in the same folder as this script)
	xmlFile := "invoice-prices-include-vat.xml"

	// KSeF authentication details
	nip := "8126178616"
	certPath := "../../../../api/test/cert-20260102-131809.pfx"
	// =============================================

	fmt.Printf("Testing FAVAT XML sending to KSeF\n")
	fmt.Printf("================================\n")
	fmt.Printf("NIP: %s\n", nip)
	fmt.Printf("Certificate: %s\n", certPath)
	fmt.Printf("XML file: %s\n\n", xmlFile)

	// Load XML file
	fmt.Println("Step 1: Loading FAVAT XML...")
	xmlBytes, err := os.ReadFile(xmlFile)
	if err != nil {
		log.Fatalf("Failed to read XML file: %v", err)
	}
	fmt.Printf("✓ Loaded XML file (%d bytes)\n\n", len(xmlBytes))

	// Create API client
	fmt.Println("Step 2: Authenticating with KSeF...")
	cert, err := xmldsig.LoadCertificate(certPath, "")
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	client := ksef_api.NewClient(
		&ksef_api.ContextIdentifier{Nip: nip},
		cert,
	)

	ctx := context.Background()
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("KSeF authentication failed: %v", err)
	}
	fmt.Printf("✓ Successfully authenticated\n\n")

	// Create upload session
	fmt.Println("Step 3: Creating upload session...")
	session, err := client.CreateSession(ctx)
	if err != nil {
		log.Fatalf("Session creation failed: %v", err)
	}
	fmt.Printf("✓ Session created\n\n")

	// Upload invoice
	fmt.Println("Step 4: Uploading invoice...")
	if err := session.UploadInvoice(ctx, xmlBytes); err != nil {
		log.Fatalf("Invoice upload failed: %v", err)
	}
	fmt.Printf("✓ Invoice uploaded\n\n")

	// Finish upload
	fmt.Println("Step 5: Finishing upload session...")
	if err := session.FinishUpload(ctx); err != nil {
		log.Fatalf("Closing session failed: %v", err)
	}
	fmt.Printf("✓ Upload session finished\n\n")

	// Poll for processing completion
	fmt.Println("Step 6: Polling for processing status...")
	statusResp, err := session.PollStatus(ctx)
	if err != nil {
		log.Printf("⚠ Polling failed: %v", err)
	} else {
		fmt.Printf("✓ Processing complete\n")
		fmt.Printf("  Invoice count: %d\n", statusResp.InvoiceCount)
		fmt.Printf("  Successful: %d\n", statusResp.SuccessfulInvoiceCount)
		fmt.Printf("  Failed: %d\n\n", statusResp.FailedInvoiceCount)
	}

	// Get uploaded invoices with KSeF numbers
	fmt.Println("Step 7: Retrieving KSeF numbers...")
	uploadedInvoices, err := session.ListUploadedInvoices(ctx)
	if err != nil {
		log.Printf("⚠ Failed to retrieve uploaded invoices: %v\n", err)
	} else {
		fmt.Printf("✓ Retrieved %d invoice(s)\n", len(uploadedInvoices))
		for _, inv := range uploadedInvoices {
			fmt.Printf("\n  Invoice #%d: %s\n", inv.OrdinalNumber, inv.InvoiceNumber)
			fmt.Printf("  ➜ KSeF Number: %s\n", inv.KsefNumber)
			fmt.Printf("  ➜ Reference: %s\n", inv.ReferenceNumber)
			fmt.Printf("  ➜ Acquisition Date: %s\n", inv.AcquisitionDate.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	// Check for failed uploads
	fmt.Println("Step 8: Checking for failed uploads...")
	failed, err := session.GetFailedUploadData(ctx)
	if err != nil {
		log.Printf("⚠ Failed uploads lookup failed: %v\n", err)
	} else if len(failed) > 0 {
		fmt.Printf("⚠ Found %d failed invoice(s):\n", len(failed))
		for _, inv := range failed {
			fmt.Printf("  - Reference: %s (Ordinal: %d)\n", inv.ReferenceNumber, inv.OrdinalNumber)
			fmt.Printf("    Status: %+v\n", inv.Status)
		}
	} else {
		fmt.Printf("✓ No failed uploads\n")
	}

	fmt.Println("\n✓ All steps completed successfully!")
}
