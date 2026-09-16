package ksef_test

import (
	"encoding/xml"
	"strings"
	"testing"

	ksef "github.com/invopop/gobl.pl.ksef"
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFavatSeller(t *testing.T) {
	t.Run("creates seller with basic data", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Equal(t, "PL", seller.VATPrefix)
		assert.Equal(t, "1234567890", seller.NIP)
		assert.Equal(t, "Test Company Sp. z o.o.", seller.Name)
		assert.NotNil(t, seller.Address)
		assert.Equal(t, "PL", seller.Address.CountryCode)
		assert.Contains(t, seller.Address.AddressL1, "ul. Testowa")
		assert.Contains(t, seller.Address.AddressL1, "123")
		assert.Nil(t, seller.Contact)
	})

	t.Run("creates seller with phone number", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
			Telephones: []*org.Telephone{
				{Number: "+48 123 456 789"},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.NotNil(t, seller.Contact)
		assert.Equal(t, "+48 123 456 789", seller.Contact.Phone)
		assert.Empty(t, seller.Contact.Email)
	})

	t.Run("creates seller with email", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
			Emails: []*org.Email{
				{Address: "contact@testcompany.pl"},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.NotNil(t, seller.Contact)
		assert.Equal(t, "contact@testcompany.pl", seller.Contact.Email)
		assert.Empty(t, seller.Contact.Phone)
	})

	t.Run("creates seller with both phone and email", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
			Telephones: []*org.Telephone{
				{Number: "+48 123 456 789"},
			},
			Emails: []*org.Email{
				{Address: "contact@testcompany.pl"},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.NotNil(t, seller.Contact)
		assert.Equal(t, "+48 123 456 789", seller.Contact.Phone)
		assert.Equal(t, "contact@testcompany.pl", seller.Contact.Email)
	})

	t.Run("splits long address into two lines", func(t *testing.T) {
		// Create a street name that will exceed 512 characters when combined
		longStreet := strings.Repeat("a", 500)
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   longStreet,
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Len(t, seller.Address.AddressL1, 512)
		assert.NotEmpty(t, seller.Address.AddressL2)
	})

	t.Run("does not set AddressL2 when address is under 512 chars", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Empty(t, seller.Address.AddressL2)
	})

	t.Run("uses first phone when multiple are present", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
			Telephones: []*org.Telephone{
				{Number: "+48 111 111 111"},
				{Number: "+48 222 222 222"},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Equal(t, "+48 111 111 111", seller.Contact.Phone)
	})

	t.Run("uses first email when multiple are present", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
			Emails: []*org.Email{
				{Address: "first@testcompany.pl"},
				{Address: "second@testcompany.pl"},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Equal(t, "first@testcompany.pl", seller.Contact.Email)
	})

	t.Run("nil TaxID does not panic", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			Addresses: []*org.Address{
				{
					Street:   "ul. Testowa",
					Number:   "123",
					Code:     "00-001",
					Locality: "Warszawa",
					Country:  l10n.PL.ISO(),
				},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Empty(t, seller.VATPrefix)
		assert.Empty(t, seller.NIP)
		assert.Equal(t, "Test Company Sp. z o.o.", seller.Name)
		assert.NotNil(t, seller.Address)
	})

	t.Run("no addresses does not panic", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Equal(t, "PL", seller.VATPrefix)
		assert.Equal(t, "1234567890", seller.NIP)
		assert.Nil(t, seller.Address)
	})

	t.Run("nil TaxID and no addresses does not panic", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.Empty(t, seller.VATPrefix)
		assert.Empty(t, seller.NIP)
		assert.Nil(t, seller.Address)
		assert.Equal(t, "Test Company Sp. z o.o.", seller.Name)
	})

	t.Run("address with only country does not emit empty AdresL1 in XML", func(t *testing.T) {
		supplier := &org.Party{
			Name: "Test Company Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Addresses: []*org.Address{
				{
					Country: l10n.PL.ISO(),
				},
			},
		}

		seller := ksef.NewFavatSeller(supplier)

		assert.NotNil(t, seller.Address)
		assert.Equal(t, "PL", seller.Address.CountryCode)

		data, err := xml.Marshal(seller.Address)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "<AdresL1></AdresL1>")
	})
}

func TestNewFavatBuyer(t *testing.T) {
	t.Run("nil customer returns buyer with NoID", func(t *testing.T) {
		buyer := ksef.NewFavatBuyer(nil)

		assert.Equal(t, 1, buyer.NoID)
		assert.Equal(t, "2", buyer.JST)
		assert.Equal(t, "2", buyer.GV)
		assert.Empty(t, buyer.NIP)
		assert.Empty(t, buyer.UECode)
		assert.Empty(t, buyer.CountryCode)
	})

	t.Run("customer with nil TaxID returns buyer with NoID", func(t *testing.T) {
		customer := &org.Party{
			Name: "Jan Kowalski",
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, 1, buyer.NoID)
		assert.Equal(t, "Jan Kowalski", buyer.Name)
	})

	t.Run("Polish business entity sets NIP", func(t *testing.T) {
		customer := &org.Party{
			Name: "Polish Buyer Sp. z o.o.",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "9876543210",
			},
			Addresses: []*org.Address{
				{
					Street:   "ul. Kupiecka",
					Number:   "45",
					Code:     "00-002",
					Locality: "Kraków",
					Country:  l10n.PL.ISO(),
				},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, "9876543210", buyer.NIP)
		assert.Empty(t, buyer.UECode)
		assert.Empty(t, buyer.CountryCode)
		assert.Equal(t, 0, buyer.NoID)
		assert.Equal(t, "Polish Buyer Sp. z o.o.", buyer.Name)
		assert.NotNil(t, buyer.Address)
		assert.Equal(t, "PL", buyer.Address.CountryCode)
	})

	t.Run("EU business entity (non-Polish) sets UECode and UEVatNumber", func(t *testing.T) {
		customer := &org.Party{
			Name: "German Company GmbH",
			TaxID: &tax.Identity{
				Country: l10n.DE.Tax(),
				Code:    "123456789",
			},
			Addresses: []*org.Address{
				{
					Street:   "Hauptstraße",
					Number:   "10",
					Code:     "10115",
					Locality: "Berlin",
					Country:  l10n.DE.ISO(),
				},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, "DE", buyer.UECode)
		assert.Equal(t, "123456789", buyer.UEVatNumber)
		assert.Empty(t, buyer.NIP)
		assert.Empty(t, buyer.CountryCode)
		assert.Equal(t, 0, buyer.NoID)
	})

	t.Run("non-EU business entity sets CountryCode and IDNumber", func(t *testing.T) {
		customer := &org.Party{
			Name: "US Corporation Inc.",
			TaxID: &tax.Identity{
				Country: l10n.US.Tax(),
				Code:    "12-3456789",
			},
			Addresses: []*org.Address{
				{
					Street:   "Main Street",
					Number:   "100",
					Code:     "10001",
					Locality: "New York",
					Country:  l10n.US.ISO(),
				},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, "US", buyer.CountryCode)
		assert.Equal(t, "12-3456789", buyer.IDNumber)
		assert.Empty(t, buyer.NIP)
		assert.Empty(t, buyer.UECode)
		assert.Equal(t, 0, buyer.NoID)
	})

	t.Run("non-EU business entity without tax code sets NoID", func(t *testing.T) {
		customer := &org.Party{
			Name: "US Corporation Inc.",
			TaxID: &tax.Identity{
				Country: l10n.US.Tax(),
				Code:    "",
			},
			Addresses: []*org.Address{
				{
					Street:   "Main Street",
					Number:   "100",
					Code:     "10001",
					Locality: "New York",
					Country:  l10n.US.ISO(),
				},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, 1, buyer.NoID)
		assert.Empty(t, buyer.CountryCode)
		assert.Empty(t, buyer.IDNumber)
	})

	t.Run("creates buyer with phone number", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Buyer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Telephones: []*org.Telephone{
				{Number: "+48 987 654 321"},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.NotNil(t, buyer.Contact)
		assert.Equal(t, "+48 987 654 321", buyer.Contact.Phone)
		assert.Empty(t, buyer.Contact.Email)
	})

	t.Run("creates buyer with email", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Buyer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Emails: []*org.Email{
				{Address: "buyer@example.pl"},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.NotNil(t, buyer.Contact)
		assert.Equal(t, "buyer@example.pl", buyer.Contact.Email)
		assert.Empty(t, buyer.Contact.Phone)
	})

	t.Run("creates buyer with both phone and email", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Buyer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Telephones: []*org.Telephone{
				{Number: "+48 987 654 321"},
			},
			Emails: []*org.Email{
				{Address: "buyer@example.pl"},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.NotNil(t, buyer.Contact)
		assert.Equal(t, "+48 987 654 321", buyer.Contact.Phone)
		assert.Equal(t, "buyer@example.pl", buyer.Contact.Email)
	})

	t.Run("uses first phone when multiple are present", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Buyer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Telephones: []*org.Telephone{
				{Number: "+48 111 111 111"},
				{Number: "+48 222 222 222"},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, "+48 111 111 111", buyer.Contact.Phone)
	})

	t.Run("uses first email when multiple are present", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Buyer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
			Emails: []*org.Email{
				{Address: "first@example.pl"},
				{Address: "second@example.pl"},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, "first@example.pl", buyer.Contact.Email)
	})

	t.Run("buyer without address has nil Address field", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Buyer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Nil(t, buyer.Address)
	})

	t.Run("JST and GV default to 2 (No)", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Buyer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1234567890",
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, "2", buyer.JST)
		assert.Equal(t, "2", buyer.GV)
	})

	t.Run("address with only country does not emit empty AdresL1 in XML", func(t *testing.T) {
		customer := &org.Party{
			Name: "German Company GmbH",
			TaxID: &tax.Identity{
				Country: l10n.DE.Tax(),
				Code:    "123456789",
			},
			Addresses: []*org.Address{
				{
					Country: l10n.DE.ISO(),
				},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.NotNil(t, buyer.Address)
		assert.Equal(t, "DE", buyer.Address.CountryCode)

		data, err := xml.Marshal(buyer.Address)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "<AdresL1></AdresL1>")
	})

	t.Run("Spanish EU business entity", func(t *testing.T) {
		customer := &org.Party{
			Name: "Spanish Company S.L.",
			TaxID: &tax.Identity{
				Country: l10n.ES.Tax(),
				Code:    "B12345678",
			},
			Addresses: []*org.Address{
				{
					Street:   "Calle Mayor",
					Number:   "1",
					Code:     "28001",
					Locality: "Madrid",
					Country:  l10n.ES.ISO(),
				},
			},
		}

		buyer := ksef.NewFavatBuyer(customer)

		assert.Equal(t, "ES", buyer.UECode)
		assert.Equal(t, "B12345678", buyer.UEVatNumber)
		assert.Empty(t, buyer.NIP)
		assert.Empty(t, buyer.CountryCode)
	})
}

func TestNewThirdParties(t *testing.T) {
	baseInvoice := func() *bill.Invoice {
		return &bill.Invoice{
			Supplier: &org.Party{
				Name: "Test Supplier Sp. z o.o.",
				TaxID: &tax.Identity{
					Country: l10n.PL.Tax(),
					Code:    "1234567890",
				},
			},
		}
	}

	t.Run("returns empty slice when no identities", func(t *testing.T) {
		inv := baseInvoice()

		thirdParties := ksef.NewThirdParties(inv)

		assert.Empty(t, thirdParties)
	})

	t.Run("returns empty slice when supplier identity has no ext", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "1234567890",
				Country: l10n.PL.ISO(),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Empty(t, thirdParties)
	})

	t.Run("returns empty slice when supplier identity has no role in ext", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "1234567890",
				Country: l10n.PL.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					"some-other-key": "value",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Empty(t, thirdParties)
	})

	t.Run("creates third party from Polish supplier identity", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "9876543210",
				Country: l10n.PL.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "1", // Factor
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "1", thirdParties[0].Role)
		assert.Equal(t, "9876543210", thirdParties[0].NIP)
		assert.Empty(t, thirdParties[0].UECode)
		assert.Empty(t, thirdParties[0].CountryCode)
		assert.Equal(t, 0, thirdParties[0].NoID)
	})

	t.Run("creates third party from EU supplier identity", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "DE123456789",
				Country: l10n.DE.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "2",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "2", thirdParties[0].Role)
		assert.Equal(t, "DE", thirdParties[0].UECode)
		assert.Equal(t, "DE123456789", thirdParties[0].UEVatNumber)
		assert.Empty(t, thirdParties[0].NIP)
		assert.Empty(t, thirdParties[0].CountryCode)
	})

	t.Run("creates third party from non-EU supplier identity", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "12-3456789",
				Country: l10n.US.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "3",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "3", thirdParties[0].Role)
		assert.Equal(t, "US", thirdParties[0].CountryCode)
		assert.Equal(t, "12-3456789", thirdParties[0].IDNumber)
		assert.Empty(t, thirdParties[0].NIP)
		assert.Empty(t, thirdParties[0].UECode)
	})

	t.Run("creates third party with NoID when identity has no code", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "",
				Country: l10n.PL.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "4",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "4", thirdParties[0].Role)
		assert.Equal(t, 1, thirdParties[0].NoID)
		assert.Empty(t, thirdParties[0].NIP)
		assert.Empty(t, thirdParties[0].UECode)
		assert.Empty(t, thirdParties[0].CountryCode)
	})

	t.Run("creates third party with only IDNumber when identity has code but no country", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code: "ABC123456",
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "4",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "4", thirdParties[0].Role)
		assert.Equal(t, "ABC123456", thirdParties[0].IDNumber)
		assert.Empty(t, thirdParties[0].CountryCode)
		assert.Empty(t, thirdParties[0].NIP)
		assert.Empty(t, thirdParties[0].UECode)
		assert.Equal(t, 0, thirdParties[0].NoID)
	})

	t.Run("creates third party from customer identity", func(t *testing.T) {
		inv := baseInvoice()
		inv.Customer = &org.Party{
			Name: "Test Customer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "1111111111",
			},
			Identities: []*org.Identity{
				{
					Code:    "2222222222",
					Country: l10n.PL.ISO(),
					Ext: tax.ExtensionsOf(cbc.CodeMap{
						favat.ExtKeyThirdPartyRole: "8", // JST subordinate unit
					}),
				},
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "8", thirdParties[0].Role)
		assert.Equal(t, "2222222222", thirdParties[0].NIP)
	})

	t.Run("creates third parties from both supplier and customer identities", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "1111111111",
				Country: l10n.PL.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "1", // Factor
				}),
			},
		}
		inv.Customer = &org.Party{
			Name: "Test Customer",
			TaxID: &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    "2222222222",
			},
			Identities: []*org.Identity{
				{
					Code:    "3333333333",
					Country: l10n.PL.ISO(),
					Ext: tax.ExtensionsOf(cbc.CodeMap{
						favat.ExtKeyThirdPartyRole: "10", // VAT group member
					}),
				},
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 2)
		assert.Equal(t, "1", thirdParties[0].Role)
		assert.Equal(t, "1111111111", thirdParties[0].NIP)
		assert.Equal(t, "10", thirdParties[1].Role)
		assert.Equal(t, "3333333333", thirdParties[1].NIP)
	})

	t.Run("ignores customer identity when customer is nil", func(t *testing.T) {
		inv := baseInvoice()
		inv.Customer = nil
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "1111111111",
				Country: l10n.PL.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "1",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
	})

	t.Run("uses only first supplier identity", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "1111111111",
				Country: l10n.PL.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "1",
				}),
			},
			{
				Code:    "2222222222",
				Country: l10n.PL.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "2",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "1111111111", thirdParties[0].NIP)
		assert.Equal(t, "1", thirdParties[0].Role)
	})

	t.Run("uses only first customer identity", func(t *testing.T) {
		inv := baseInvoice()
		inv.Customer = &org.Party{
			Name: "Test Customer",
			Identities: []*org.Identity{
				{
					Code:    "1111111111",
					Country: l10n.PL.ISO(),
					Ext: tax.ExtensionsOf(cbc.CodeMap{
						favat.ExtKeyThirdPartyRole: "8",
					}),
				},
				{
					Code:    "2222222222",
					Country: l10n.PL.ISO(),
					Ext: tax.ExtensionsOf(cbc.CodeMap{
						favat.ExtKeyThirdPartyRole: "10",
					}),
				},
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "1111111111", thirdParties[0].NIP)
		assert.Equal(t, "8", thirdParties[0].Role)
	})

	t.Run("Spanish EU identity sets UECode and UEVatNumber", func(t *testing.T) {
		inv := baseInvoice()
		inv.Supplier.Identities = []*org.Identity{
			{
				Code:    "B12345678",
				Country: l10n.ES.ISO(),
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					favat.ExtKeyThirdPartyRole: "5",
				}),
			},
		}

		thirdParties := ksef.NewThirdParties(inv)

		assert.Len(t, thirdParties, 1)
		assert.Equal(t, "ES", thirdParties[0].UECode)
		assert.Equal(t, "B12345678", thirdParties[0].UEVatNumber)
	})
}

func TestSellerToGOBL(t *testing.T) {
	t.Run("converts Polish seller to GOBL party", func(t *testing.T) {
		seller := &ksef.Seller{
			VATPrefix: "PL",
			NIP:       "1234567890",
			Name:      "Test Company Sp. z o.o.",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Testowa 123, 00-001 Warszawa",
			},
		}

		party := seller.ToGOBL()

		assert.Equal(t, "Test Company Sp. z o.o.", party.Name)
		assert.NotNil(t, party.TaxID)
		assert.Equal(t, l10n.PL.Tax(), party.TaxID.Country)
		assert.Equal(t, "1234567890", string(party.TaxID.Code))
		assert.Len(t, party.Addresses, 1)
		assert.Equal(t, l10n.PL.ISO(), party.Addresses[0].Country)
	})

	t.Run("converts EU seller with different VAT prefix", func(t *testing.T) {
		seller := &ksef.Seller{
			VATPrefix: "DE",
			NIP:       "123456789",
			Name:      "German Company GmbH",
			Address: &ksef.Address{
				CountryCode: "DE",
				AddressL1:   "Hauptstraße 10, 10115 Berlin",
			},
		}

		party := seller.ToGOBL()

		assert.NotNil(t, party.TaxID)
		assert.Equal(t, l10n.DE.Tax(), party.TaxID.Country)
		assert.Equal(t, "123456789", string(party.TaxID.Code))
	})

	t.Run("converts seller with contact info", func(t *testing.T) {
		seller := &ksef.Seller{
			VATPrefix: "PL",
			NIP:       "1234567890",
			Name:      "Test Company",
			Contact: &ksef.ContactDetails{
				Phone: "+48 123 456 789",
				Email: "contact@test.pl",
			},
		}

		party := seller.ToGOBL()

		assert.Len(t, party.Telephones, 1)
		assert.Equal(t, "+48 123 456 789", party.Telephones[0].Number)
		assert.Len(t, party.Emails, 1)
		assert.Equal(t, "contact@test.pl", party.Emails[0].Address)
	})

	t.Run("converts seller without address", func(t *testing.T) {
		seller := &ksef.Seller{
			VATPrefix: "PL",
			NIP:       "1234567890",
			Name:      "Test Company",
		}

		party := seller.ToGOBL()

		assert.Empty(t, party.Addresses)
	})
}

func TestBuyerToGOBL(t *testing.T) {
	t.Run("converts Polish buyer to GOBL party", func(t *testing.T) {
		buyer := &ksef.Buyer{
			NIP:  "9876543210",
			Name: "Polish Buyer Sp. z o.o.",
			Address: &ksef.Address{
				CountryCode: "PL",
				AddressL1:   "ul. Kupiecka 45, 00-002 Kraków",
			},
		}

		party := buyer.ToGOBL()

		assert.Equal(t, "Polish Buyer Sp. z o.o.", party.Name)
		assert.NotNil(t, party.TaxID)
		assert.Equal(t, l10n.PL.Tax(), party.TaxID.Country)
		assert.Equal(t, "9876543210", string(party.TaxID.Code))
	})

	t.Run("converts EU buyer with UE code", func(t *testing.T) {
		buyer := &ksef.Buyer{
			UECode:      "DE",
			UEVatNumber: "123456789",
			Name:        "German Buyer",
			Address: &ksef.Address{
				CountryCode: "DE",
				AddressL1:   "Hauptstraße 10",
			},
		}

		party := buyer.ToGOBL()

		assert.NotNil(t, party.TaxID)
		assert.Equal(t, l10n.DE.Tax(), party.TaxID.Country)
		assert.Equal(t, "123456789", string(party.TaxID.Code))
	})

	t.Run("converts non-EU buyer with country code", func(t *testing.T) {
		buyer := &ksef.Buyer{
			CountryCode: "US",
			IDNumber:    "12-3456789",
			Name:        "US Buyer",
			Address: &ksef.Address{
				CountryCode: "US",
				AddressL1:   "Main Street 100",
			},
		}

		party := buyer.ToGOBL()

		assert.NotNil(t, party.TaxID)
		assert.Equal(t, l10n.US.Tax(), party.TaxID.Country)
		assert.Equal(t, "12-3456789", string(party.TaxID.Code))
	})

	t.Run("returns nil for buyer with NoID=1 and no identifying info", func(t *testing.T) {
		buyer := &ksef.Buyer{
			NoID: 1,
		}

		party := buyer.ToGOBL()

		assert.Nil(t, party)
	})

	t.Run("converts buyer with contact info", func(t *testing.T) {
		buyer := &ksef.Buyer{
			NIP:  "1234567890",
			Name: "Test Buyer",
			Contact: &ksef.ContactDetails{
				Phone: "+48 987 654 321",
				Email: "buyer@example.pl",
			},
		}

		party := buyer.ToGOBL()

		assert.Len(t, party.Telephones, 1)
		assert.Equal(t, "+48 987 654 321", party.Telephones[0].Number)
		assert.Len(t, party.Emails, 1)
		assert.Equal(t, "buyer@example.pl", party.Emails[0].Address)
	})

	t.Run("converts buyer without address", func(t *testing.T) {
		buyer := &ksef.Buyer{
			NIP:  "1234567890",
			Name: "Test Buyer",
		}

		party := buyer.ToGOBL()

		assert.Empty(t, party.Addresses)
	})
}
