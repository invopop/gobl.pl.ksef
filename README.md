# 🇵🇱 GOBL Poland KSeF

The Polish KSeF module for GOBL: the `pl-favat-v3` addon and bidirectional
conversion between GOBL and the FA_VAT XML format.

The module is laid out in two parts:

- [`addon/`](addon) — the `pl-favat-v3` GOBL addon (extensions, scenarios,
  normalization and validation rules). See its [README](addon/README.md).
- the root package — the FA_VAT XML converter and the KSeF API client, built
  on top of the addon.

## Installation

```bash
go get github.com/invopop/gobl.pl.ksef
```

The addon registers itself on import. Consumers that only need GOBL documents
declaring `pl-favat-v3` to normalize and validate can blank-import it:

```go
import _ "github.com/invopop/gobl.pl.ksef/addon"
```

Importing the root converter package pulls the addon in automatically.

## Main Conversion Entrypoints

**GOBL → KSeF:**
- `ksef.BuildFavat(env *gobl.Envelope) (*Invoice, error)` - Converts a GOBL envelope to a KSeF FA_VAT invoice model
- `(*Invoice).Bytes() ([]byte, error)` - Returns the XML representation as bytes

**KSeF → GOBL:**
- `ksef.ParseKSeF(xmlData []byte) (*gobl.Envelope, error)` - Converts KSeF FA_VAT XML to a GOBL envelope

Copyright [Invopop Ltd.](https://invopop.com) 2023. Released publicly under the [Apache License Version 2.0](LICENSE). For commercial licenses please contact the [dev team at invopop](mailto:dev@invopop.com). In order to accept contributions to this library we will require transferring copyrights to Invopop Ltd.

## Supported Features

The converter handles the following invoice types and features:

**Invoice types (GOBL → KSeF):**
- `VAT` - Standard invoices
- `ZAL` - Advance/prepayment invoices (tag: `partial`)
- `ROZ` - Settlement invoices (tag: `settlement`)
- `UPR` - Simplified invoices (tag: `simplified`)
- `KOR` - Correction invoices (credit notes)
- `KOR_ZAL` - Correction of advance invoices
- `KOR_ROZ` - Correction of settlement invoices

**Parties:**
- Seller (Podmiot1) with Polish NIP, address, contact details, and EU VAT prefix
- Buyer (Podmiot2) with Polish NIP, EU VAT number, or non-EU tax ID
- Third parties (Podmiot3) for JST (local government units) and Group VAT scenarios

**Tax rates:**
- Standard (23%), reduced (8%), super-reduced (5%), and other percentage rates
- Zero-rated (0 KR), intra-community (0 WDT), export (0 EX)
- Tax exempt (zw), outside scope (np I), reverse charge (np II), domestic reverse charge (oo)
- OSS (One Stop Shop) rates
- Margin scheme rates

**Annotations:**
- Cash accounting, self-billing, reverse charge, split payment mechanism
- Tax exemption with legal basis (Polish law, EU directive, or other)
- Margin scheme (travel agency, used goods, art works, collectibles/antiques)

**Other features:**
- Line item discounts
- Units of measure (P_8A), see [Units](#units-of-measure-p_8a)
- Invoice periods (P_6_Od / P_6_Do), emitted only when both dates are known
- Amounts recalculated to the currency's precision, see [Rounding](#rounding)
- Correction/credit note references with KSeF numbers
- Payment details: means of payment, bank accounts, due dates, advance payments
- Additional description lines (DodatkowyOpis)
- Ordering data with order references and order lines (Zamowienie / WarunkiTransakcji)
- Rounding adjustments to reconcile KSeF and GOBL totals
- Gross pricing (P_9B) support in KSeF → GOBL direction
- Credit notes with before/after correction lines (StanPrzed)
- Prepayment invoices without line items (bypass mode with totals from tax summaries)

## CLI

The `gobl.ksef` CLI provides two commands:

- **`convert`** - Convert a GOBL JSON envelope into a FA_VAT XML document:
  ```bash
  gobl.ksef convert input.json output.xml
  ```

- **`send`** - Convert and send a GOBL JSON envelope to the KSeF API:
  ```bash
  gobl.ksef send input.json [nip] [token] [keyPath]
  ```

## Testing

The test suite includes tests for both conversion directions and round-trip validation.

### Running Tests

**Run all tests:**
```bash
go test ./test -v
```

**Update golden files:**
```bash
go test ./test --update -v
```

**With XSD schema validation (requires libxml2):**
```bash
# Using the helper script (sets LD_LIBRARY_PATH automatically)
./test/test.sh -v
./test/test.sh --update -v

# Or manually
LD_LIBRARY_PATH=/home/linuxbrew/.linuxbrew/opt/libxml2/lib:$LD_LIBRARY_PATH go test -tags xsdvalidate ./test -v
```

### Test Data

**GOBL → KSeF conversion:**
- **Input**: GOBL JSON files in `test/data/gobl.ksef/*.json`
- **Output**: KSeF XML files in `test/data/gobl.ksef/out/*.xml`

**KSeF → GOBL conversion:**
- **Input**: KSeF XML files in `test/data/ksef.gobl/*.xml`
- **Output**: GOBL JSON files in `test/data/ksef.gobl/out/*.json`

**Schema validation:**
- **Schema**: FA3 XSD and dependencies in `test/data/schema/`

## Unsupported fields

See [unsupported-fields.md](unsupported-fields.md) for the list of unsupported fields.

## FA_VAT documentation

FA_VAT is the Polish electronic invoice format. The format uses XML.

- [XML schema](https://github.com/CIRFMF/ksef-docs/blob/main/faktury/schemy/FA/schemat_FA(3)_v1-0E.xsd) for V3 (description of fields is in Polish)
- [Types definition](https://raw.githubusercontent.com/CIRFMF/ksef-docs/refs/heads/main/faktury/schemy/FA/bazowe/ElementarneTypyDanych_v10-0E.xsd) (description of fields is in Polish) - we have to open it as raw, as [the original link](https://github.com/CIRFMF/ksef-docs/blob/main/faktury/schemy/FA/bazowe/StrukturyDanych_v10-0E.xsd) does not add newlines
- [Complex types definition](https://raw.githubusercontent.com/CIRFMF/ksef-docs/refs/heads/main/faktury/schemy/FA/bazowe/StrukturyDanych_v10-0E.xsd) (description of fields is in Polish) - we have to open it as raw, as [the original link](https://github.com/CIRFMF/ksef-docs/blob/main/faktury/schemy/FA/bazowe/StrukturyDanych_v10-0E.xsd) does not add newlines

## Parsing (KSeF → GOBL)

The parsing functionality converts KSeF FA_VAT XML documents back into GOBL format. The implementation includes:

- **Party conversion**: Converts seller (Podmiot1), buyer (Podmiot2), and third parties (Podmiot3) to GOBL parties
- **Invoice data**: Parses invoice metadata including type, codes, dates, currency, and annotations
- **Line items**: Converts FA_VAT line items (FaWiersz) to GOBL invoice lines, including discounts and tax combos
- **Credit notes**: Handles line inversion for correction invoices, including before/after correction lines (StanPrzed)
- **Gross pricing**: Supports invoices using gross unit prices (P_9B) by setting `PricesInclude = VAT`
- **Ordering data**: Maps Zamowienie (order) and WarunkiTransakcji (transaction conditions) to GOBL ordering purchases
- **Payment details**: Extracts payment means, bank accounts, due dates, and advance payments
- **Prepayment invoices**: Handles advance invoices without line items (ZAL/KOR_ZAL) using bypass mode with totals from tax summary fields
- **Settlement invoices**: Derives advance payments for ROZ/KOR_ROZ invoices (see below)
- **Rounding adjustments**: Handles rounding differences between KSeF and GOBL calculation methods
- **Round-trip validation**: All GOBL → KSeF conversions are validated through round-trip tests (GOBL → KSeF → GOBL)

## Rounding

FA(3) amounts use the `TKwotowy` type, which allows at most two decimal places.
GOBL's `precise` rounding rule keeps extra decimals on line totals, so
`BuildFavat` recalculates every invoice with the `currency` rule
(`bill.Invoice.RoundToCurrency`) before conversion. Any resulting change to the
amount payable is carried in the totals' `rounding` amount (BT-114 in EN 16931).
Invoices already within the currency's precision are left untouched.

In the KSeF → GOBL direction the parsed invoice is likewise given the `currency`
rounding rule, and `AdjustRounding` reconciles the calculated total against
`P_15`, since KSeF rounds each line before summing.

## Units of measure (P_8A)

KSeF accepts free-form unit strings, while GOBL takes a defined unit key with
any UN/ECE Recommendation 20/21 code held in the `untdid-unit` extension. A
measure read from KSeF is mapped as follows:

| P_8A value | GOBL representation |
| ---------- | ------------------- |
| a GOBL unit key (`h`, `kg`, …) | `item.unit` |
| a UN/ECE code (`HUR`, `KGM`, `SZT`, …) | `item.ext["untdid-unit"]`, plus `item.unit` when the code has a GOBL equivalent |
| anything else (`szt.`, `kilo`, …) | `item.meta["unit-label"]` |

Going the other way, P_8A is taken from `unit-label` if present, then from the
`untdid-unit` extension, and finally from the standard mapping of the GOBL unit
key. Every unit GOBL defines has an exact UNTDID equivalent, so only an item
without a unit at all leaves P_8A empty.

## Settlement Invoices (ROZ)

Settlement invoices (`ROZ`) finalize orders that had advance payments (`ZAL` invoices). Per Art. 106f sec. 3 of the Polish VAT Act, they must show the full order value in line items (`FaWiersz`) while `P_15` contains only the remaining amount after advance deductions. The advance invoice references appear in `FakturaZaliczkowa` elements.

This creates a structural mismatch: GOBL calculates Payable from line items (full amount), but P_15 is the remaining balance. For example, an order worth 68,363.40 PLN with a 13,672.68 PLN advance would have lines totalling 68,363.40 but P_15 = 54,690.72.

**KSeF → GOBL:** The converter detects settlement invoices with `FakturaZaliczkowa` references and no explicit `ZaplataCzesciowa` (partial payment) entries, and derives the advance amount as `Payable - P_15`. This produces a natural GOBL invoice where lines represent the full order, advances represent prepaid amounts, and Due equals the remaining balance. When explicit `ZaplataCzesciowa` entries are present (as in some ERP systems), those are used directly and no derivation occurs.

**GOBL → KSeF:** When a settlement invoice has advances, the converter prorates the tax summary fields (`P_13_X`/`P_14_X`) by the ratio `Due / Payable` so they reflect only the remaining amounts. Advance `ref` values are mapped to `FakturaZaliczkowa` elements. Lines and `ZaplataCzesciowa` entries are emitted unchanged.

## KSeF API

KSeF is the Polish system for submitting electronic invoices to the Polish authorities. The system uses API version 2.0, which introduced JWT-based authentication, mandatory invoice encryption, and a unified session model.

Useful links:

- [National e-Invoice System](https://ksef.mf.gov.pl/) - for details on system in general (English translation available - language picker is in the top right corner)
- [KSeF 2.0 Integrator's Guide](./docs/ksef-docs-en/README.md) - translated documentation with detailed integration instructions

KSeF provides three environments:

| Environment | Description | API Documentation |
| ----------- | ----------- | ----------------- |
| **Test** (Release Candidate) | For testing integration, contains RC versions | [api-test.ksef.mf.gov.pl](https://api-test.ksef.mf.gov.pl/docs/v2) |
| **Demo** (Pre-production) | Matches production configuration, for final validation | [api-demo.ksef.mf.gov.pl](https://api-demo.ksef.mf.gov.pl/docs/v2) |
| **Production** | Full legal validity, production data | [api.ksef.mf.gov.pl](https://api.ksef.mf.gov.pl/docs/v2) |

The OpenAPI specification is available at each environment's `/docs/v2/openapi.json` endpoint.

## Authentication

See [authentication.md](./authentication.md).

