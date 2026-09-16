package ksef_test

import (
	"strings"
	"testing"

	ksef "github.com/invopop/gobl.pl.ksef"
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettlementOutbound(t *testing.T) {
	baseInvoice := func() *bill.Invoice {
		return &bill.Invoice{
			Currency: currency.PLN,
			Supplier: &org.Party{TaxID: &tax.Identity{Country: l10n.PL.Tax()}},
			Tax: &bill.Tax{
				Ext: tax.ExtensionsOf(cbc.CodeMap{favat.ExtKeyInvoiceType: "VAT"}),
			},
			Totals: &bill.Totals{Taxes: &tax.Total{}},
		}
	}

	t.Run("nil settlement when no charges or discounts", func(t *testing.T) {
		out := ksef.NewFavatInv(baseInvoice())
		assert.Nil(t, out.Settlement)
	})

	t.Run("emits charges as Obciazenia with totals", func(t *testing.T) {
		inv := baseInvoice()
		inv.Charges = []*bill.Charge{
			{Amount: num.MakeAmount(43681, 2), Reason: "Insurance"},
			{Amount: num.MakeAmount(1000, 2), Reason: "Handling"},
		}

		out := ksef.NewFavatInv(inv)
		require.NotNil(t, out.Settlement)
		require.Len(t, out.Settlement.Charges, 2)
		assert.Equal(t, "436.81", out.Settlement.Charges[0].Amount)
		assert.Equal(t, "Insurance", out.Settlement.Charges[0].Reason)
		assert.Equal(t, "10.00", out.Settlement.Charges[1].Amount)
		assert.Equal(t, "446.81", out.Settlement.TotalCharges)
		assert.Empty(t, out.Settlement.Deductions)
		assert.Empty(t, out.Settlement.TotalDeductions)
	})

	t.Run("emits discounts as Odliczenia with totals", func(t *testing.T) {
		inv := baseInvoice()
		inv.Discounts = []*bill.Discount{
			{Amount: num.MakeAmount(2500, 2), Reason: "Loyalty"},
		}

		out := ksef.NewFavatInv(inv)
		require.NotNil(t, out.Settlement)
		require.Len(t, out.Settlement.Deductions, 1)
		assert.Equal(t, "25.00", out.Settlement.Deductions[0].Amount)
		assert.Equal(t, "Loyalty", out.Settlement.Deductions[0].Reason)
		assert.Equal(t, "25.00", out.Settlement.TotalDeductions)
	})
}

// reportedSettlementXML reproduces the rounding-error case from the client
// report: four lines summing to 4874.50 PLN net, VAT 1121.13, and a single
// invoice-level surcharge of 436.81 under <Rozliczenie><Obciazenia>.
// P_15 = 6432.44 reflects the surcharge.
const reportedSettlementXML = `<?xml version="1.0" encoding="utf-8"?>
<Faktura xmlns="http://crd.gov.pl/wzor/2025/06/25/13775/">
  <Naglowek>
    <KodFormularza kodSystemowy="FA (3)" wersjaSchemy="1-0E">FA</KodFormularza>
    <WariantFormularza>3</WariantFormularza>
    <DataWytworzeniaFa>2026-05-01T14:54:52Z</DataWytworzeniaFa>
    <SystemInfo>Test</SystemInfo>
  </Naglowek>
  <Podmiot1>
    <DaneIdentyfikacyjne>
      <NIP>5213033865</NIP>
      <Nazwa>Supplier Sp. z o.o.</Nazwa>
    </DaneIdentyfikacyjne>
    <Adres>
      <KodKraju>PL</KodKraju>
      <AdresL1>ul. Test 1</AdresL1>
    </Adres>
  </Podmiot1>
  <Podmiot2>
    <DaneIdentyfikacyjne>
      <NIP>1231281838</NIP>
      <Nazwa>Customer Sp. z o.o.</Nazwa>
    </DaneIdentyfikacyjne>
    <Adres>
      <KodKraju>PL</KodKraju>
      <AdresL1>ul. Test 2</AdresL1>
    </Adres>
  </Podmiot2>
  <Fa>
    <KodWaluty>PLN</KodWaluty>
    <P_1>2026-05-01</P_1>
    <P_2>26133008</P_2>
    <P_13_1>4874.50</P_13_1>
    <P_14_1>1121.13</P_14_1>
    <P_15>6432.44</P_15>
    <Adnotacje>
      <P_16>2</P_16>
      <P_17>2</P_17>
      <P_18>2</P_18>
      <P_18A>2</P_18A>
      <Zwolnienie><P_19N>1</P_19N></Zwolnienie>
      <NoweSrodkiTransportu><P_22N>1</P_22N></NoweSrodkiTransportu>
      <P_23>2</P_23>
      <PMarzy><P_PMarzyN>1</P_PMarzyN></PMarzy>
    </Adnotacje>
    <RodzajFaktury>VAT</RodzajFaktury>
    <FaWiersz>
      <NrWierszaFa>1</NrWierszaFa>
      <P_7>Service A</P_7>
      <P_8A>PLN</P_8A>
      <P_8B>1</P_8B>
      <P_9A>782.02</P_9A>
      <P_11>782.02</P_11>
      <P_12>23</P_12>
    </FaWiersz>
    <FaWiersz>
      <NrWierszaFa>2</NrWierszaFa>
      <P_7>Service B</P_7>
      <P_8A>PLN</P_8A>
      <P_8B>1</P_8B>
      <P_9A>2592.48</P_9A>
      <P_11>2592.48</P_11>
      <P_12>23</P_12>
    </FaWiersz>
    <FaWiersz>
      <NrWierszaFa>3</NrWierszaFa>
      <P_7>Service C</P_7>
      <P_8A>PLN</P_8A>
      <P_8B>1</P_8B>
      <P_9A>677.79</P_9A>
      <P_11>677.79</P_11>
      <P_12>23</P_12>
    </FaWiersz>
    <FaWiersz>
      <NrWierszaFa>4</NrWierszaFa>
      <P_7>Service D</P_7>
      <P_8A>PLN</P_8A>
      <P_8B>1</P_8B>
      <P_9A>822.21</P_9A>
      <P_11>822.21</P_11>
      <P_12>23</P_12>
    </FaWiersz>
    <Rozliczenie>
      <Obciazenia>
        <Kwota>436.81</Kwota>
        <Powod>Insurance pass-through</Powod>
      </Obciazenia>
      <SumaObciazen>436.81</SumaObciazen>
    </Rozliczenie>
    <Platnosc>
      <TerminPlatnosci>
        <Termin>2026-05-15</Termin>
      </TerminPlatnosci>
      <FormaPlatnosci>6</FormaPlatnosci>
    </Platnosc>
  </Fa>
</Faktura>`

func TestSettlementInbound(t *testing.T) {
	t.Run("regression: reported invoice with Obciazenia parses without rounding error", func(t *testing.T) {
		env, err := ksef.ParseKSeF([]byte(reportedSettlementXML))
		require.NoError(t, err, "should not return RoundingError once Rozliczenie is parsed")
		require.NotNil(t, env)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.Len(t, inv.Charges, 1)
		assert.Equal(t, "436.81", inv.Charges[0].Amount.String())
		assert.Equal(t, "Insurance pass-through", inv.Charges[0].Reason)

		// Payable must reconcile to KSeF P_15 (6432.44) — no large rounding adjustment.
		require.NotNil(t, inv.Totals)
		assert.Equal(t, "6432.44", inv.Totals.Payable.String())
		if inv.Totals.Rounding != nil {
			assert.True(t, inv.Totals.Rounding.Abs().Compare(num.MakeAmount(4, 2)) <= 0,
				"rounding adjustment should be within per-line tolerance, got %s", inv.Totals.Rounding.String())
		}
	})

	t.Run("KOR with negative Obciazenia maps to positive GOBL Charges", func(t *testing.T) {
		// KSeF KOR XMLs carry negative monetary amounts (TKwotowy permits sign).
		// parseLines flips line quantities back to positive, and ksef.go inverts
		// totalDue before AdjustRounding. Settlement amounts must follow the
		// same rule, otherwise the negative charge would subtract from Payable
		// inside Calculate and the diff against the inverted P_15 would blow
		// past the rounding tolerance.
		const korXML = `<?xml version="1.0" encoding="UTF-8"?>
<Faktura xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns="http://crd.gov.pl/wzor/2025/06/25/13775/">
  <Naglowek>
    <KodFormularza kodSystemowy="FA (3)" wersjaSchemy="1-0E">FA</KodFormularza>
    <WariantFormularza>3</WariantFormularza>
    <DataWytworzeniaFa>2026-01-27T09:17:03Z</DataWytworzeniaFa>
    <SystemInfo>Invopop</SystemInfo>
  </Naglowek>
  <Podmiot1>
    <DaneIdentyfikacyjne>
      <NIP>9876543210</NIP>
      <Nazwa>Testowa Firma Sp. z o.o.</Nazwa>
    </DaneIdentyfikacyjne>
    <Adres><KodKraju>PL</KodKraju><AdresL1>ul. Test 1</AdresL1></Adres>
  </Podmiot1>
  <Podmiot2>
    <DaneIdentyfikacyjne>
      <NIP>1111111111</NIP>
      <Nazwa>Klient Testowy Sp. z o.o.</Nazwa>
    </DaneIdentyfikacyjne>
    <Adres><KodKraju>PL</KodKraju><AdresL1>ul. Test 2</AdresL1></Adres>
  </Podmiot2>
  <Fa>
    <KodWaluty>PLN</KodWaluty>
    <P_1>2026-01-20</P_1>
    <P_2>KOR-002</P_2>
    <P_13_1>-100.00</P_13_1>
    <P_14_1>-23.00</P_14_1>
    <P_15>-173.00</P_15>
    <Adnotacje>
      <P_16>2</P_16><P_17>2</P_17><P_18>2</P_18><P_18A>2</P_18A>
      <Zwolnienie><P_19N>1</P_19N></Zwolnienie>
      <NoweSrodkiTransportu><P_22N>1</P_22N></NoweSrodkiTransportu>
      <P_23>2</P_23>
      <PMarzy><P_PMarzyN>1</P_PMarzyN></PMarzy>
    </Adnotacje>
    <RodzajFaktury>KOR</RodzajFaktury>
    <PrzyczynaKorekty>Price correction</PrzyczynaKorekty>
    <TypKorekty>1</TypKorekty>
    <DaneFaKorygowanej>
      <DataWystFaKorygowanej>2026-01-20</DataWystFaKorygowanej>
      <NrFaKorygowanej>INVOICE-001</NrFaKorygowanej>
    </DaneFaKorygowanej>
    <FaWiersz>
      <NrWierszaFa>1</NrWierszaFa>
      <P_7>Service correction</P_7>
      <P_8A>HUR</P_8A>
      <P_8B>-10</P_8B>
      <P_9A>10.00</P_9A>
      <P_11>-100.00</P_11>
      <P_12>23</P_12>
    </FaWiersz>
    <Rozliczenie>
      <Obciazenia>
        <Kwota>-50.00</Kwota>
        <Powod>Insurance correction</Powod>
      </Obciazenia>
      <SumaObciazen>-50.00</SumaObciazen>
    </Rozliczenie>
  </Fa>
</Faktura>`

		env, err := ksef.ParseKSeF([]byte(korXML))
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)
		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)

		require.Len(t, inv.Charges, 1)
		assert.Equal(t, "50.00", inv.Charges[0].Amount.String(),
			"KOR negative Kwota must be inverted to positive GOBL convention")
		assert.Equal(t, "Insurance correction", inv.Charges[0].Reason)

		require.NotNil(t, inv.Totals)
		assert.Equal(t, "173.00", inv.Totals.Payable.String())
	})

	t.Run("Odliczenia maps to invoice Discounts", func(t *testing.T) {
		// Replace the Obciazenia block with an Odliczenia block.
		modified := strings.Replace(reportedSettlementXML,
			`<Rozliczenie>
      <Obciazenia>
        <Kwota>436.81</Kwota>
        <Powod>Insurance pass-through</Powod>
      </Obciazenia>
      <SumaObciazen>436.81</SumaObciazen>
    </Rozliczenie>`,
			`<Rozliczenie>
      <Odliczenia>
        <Kwota>50.00</Kwota>
        <Powod>Loyalty rebate</Powod>
      </Odliczenia>
      <SumaOdliczen>50.00</SumaOdliczen>
    </Rozliczenie>`, 1)
		// Adjust P_15 to match: lines+VAT − discount = 5995.64 − 50.00 = 5945.64
		modified = strings.Replace(modified, "<P_15>6432.44</P_15>", "<P_15>5945.64</P_15>", 1)

		env, err := ksef.ParseKSeF([]byte(modified))
		require.NoError(t, err)

		inv, ok := env.Extract().(*bill.Invoice)
		require.True(t, ok)

		require.Len(t, inv.Discounts, 1)
		assert.Equal(t, "50.00", inv.Discounts[0].Amount.String())
		assert.Equal(t, "Loyalty rebate", inv.Discounts[0].Reason)
		assert.Empty(t, inv.Charges)
	})
}
