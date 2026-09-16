package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/invopop/gobl"
	ksef "github.com/invopop/gobl.pl.ksef"
	ksef_api "github.com/invopop/gobl.pl.ksef/api"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/xmldsig"
	"github.com/spf13/cobra"
)

type sendOpts struct {
	*rootOpts
}

func send(o *rootOpts) *sendOpts {
	return &sendOpts{rootOpts: o}
}

func (c *sendOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [infile] [nip] [token] [keyPath]",
		Short: "Send a GOBL JSON to the KSeF API",
		RunE:  c.runE,
	}

	return cmd
}

func (c *sendOpts) runE(cmd *cobra.Command, args []string) error {
	// ctx := commandContext(cmd)
	nip := inputNip(args)
	token := inputToken(args)
	keyPath := inputKeyPath(args)
	_ = token

	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer func() {
		err = input.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	data, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	cert, err := xmldsig.LoadCertificate(keyPath, "")
	if err != nil {
		return fmt.Errorf("loading certificate: %w", err)
	}

	client := ksef_api.NewClient(
		&ksef_api.ContextIdentifier{Nip: nip},
		cert,
	)

	env, err := SendInvoice(client, data)
	if err != nil {
		return fmt.Errorf("sending invoices: %w", err)
	}

	data, err = json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}

	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return fmt.Errorf("invalid type %T", env.Document)
	}

	err = saveFile(filename(inv), data)
	if err != nil {
		return err
	}

	return nil
}

// SendInvoice sends invoices to KSeF
func SendInvoice(c *ksef_api.Client, data []byte) (*gobl.Envelope, error) {
	env := new(gobl.Envelope)
	if err := json.Unmarshal(data, env); err != nil {
		return nil, fmt.Errorf("parsing input as GOBL Envelope: %w", err)
	}

	doc, err := ksef.BuildFavat(env)
	if err != nil {
		return nil, fmt.Errorf("building FA_VAT document: %w", err)
	}

	dataXml, err := doc.Bytes()
	if err != nil {
		return nil, fmt.Errorf("generating FA_VAT xml: %w", err)
	}

	ctx := context.Background()
	err = c.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	uploadSession, err := c.CreateSession(ctx)
	if err != nil {
		return nil, err
	}

	err = uploadSession.UploadInvoice(ctx, dataXml)
	if err != nil {
		return nil, err
	}

	err = uploadSession.FinishUpload(ctx)
	if err != nil {
		return nil, err
	}

	_, err = uploadSession.PollStatus(ctx)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func saveFile(name string, data []byte) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Println("Error when closing:", err)
		}
	}()
	_, err = file.Write(data)
	return err
}

func filename(inv *bill.Invoice) string {
	if inv.Series != "" {
		return fmt.Sprintf("%s-%s.xml", inv.Series, inv.Code)
	}
	return fmt.Sprintf("%s.xml", inv.Code)
}
