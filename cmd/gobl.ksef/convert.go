package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/invopop/gobl"
	ksef "github.com/invopop/gobl.pl.ksef"
	"github.com/spf13/cobra"
)

type convertOpts struct {
	*rootOpts
}

func convert(o *rootOpts) *convertOpts {
	return &convertOpts{rootOpts: o}
}

func (c *convertOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert [infile] [outfile]",
		Short: "Convert a GOBL JSON into a FA_VAT XML document for Poland",
		RunE:  c.runE,
	}

	return cmd
}

func (c *convertOpts) runE(cmd *cobra.Command, args []string) error {
	// ctx := commandContext(cmd)

	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer input.Close() // nolint:errcheck

	out, err := c.openOutput(cmd, args)
	if err != nil {
		return err
	}
	defer out.Close() // nolint:errcheck

	inData, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	env := new(gobl.Envelope)
	if err := json.Unmarshal(inData, env); err != nil {
		return fmt.Errorf("parsing input as GOBL Envelope: %w", err)
	}

	// Calculate/normalize the envelope to apply addon extensions
	if err := env.Calculate(); err != nil {
		return fmt.Errorf("calculating envelope: %w", err)
	}

	doc, err := ksef.BuildFavat(env)
	if err != nil {
		return fmt.Errorf("building FA_VAT document: %w", err)
	}

	data, err := doc.Bytes()
	if err != nil {
		return fmt.Errorf("generating FA_VAT xml: %w", err)
	}

	if _, err = out.Write(data); err != nil {
		return fmt.Errorf("writing xml output: %w", err)
	}

	return nil
}
