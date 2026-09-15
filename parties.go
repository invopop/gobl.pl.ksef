package ksef

import (
	favat "github.com/invopop/gobl.pl.ksef/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// Address defines the XML structure for KSeF addresses
type Address struct {
	CountryCode string `xml:"KodKraju"`
	AddressL1   string `xml:"AdresL1,omitempty"`
	AddressL2   string `xml:"AdresL2,omitempty"`
	GLN         string `xml:"GLN,omitempty"` // Global Location Number
}

// Seller defines the XML structure for KSeF seller
type Seller struct {
	VATPrefix             string          `xml:"PrefiksPodatnika,omitempty"`
	NIP                   string          `xml:"DaneIdentyfikacyjne>NIP"`
	Name                  string          `xml:"DaneIdentyfikacyjne>Nazwa"`
	EORI                  string          `xml:"NrEORI,omitempty"`
	Address               *Address        `xml:"Adres"`
	CorrespondenceAddress *Address        `xml:"AdresKoresp,omitempty"`
	Contact               *ContactDetails `xml:"DaneKontaktowe,omitempty"`
	TaxpayerStatus        int             `xml:"StatusInfoPodatnika,omitempty"` // 1=liquidation, 2=restructuring, 3=bankruptcy, 4=inheritance
}

// ContactDetails defines the XML structure for KSeF contact
type ContactDetails struct {
	Email string `xml:"Email,omitempty"`
	Phone string `xml:"Telefon,omitempty"`
}

// Buyer defines the XML structure for KSeF buyer
type Buyer struct {
	NIP string `xml:"DaneIdentyfikacyjne>NIP,omitempty"`
	// or
	UECode      string `xml:"DaneIdentyfikacyjne>KodUE,omitempty"`   // Country code when in European Union
	UEVatNumber string `xml:"DaneIdentyfikacyjne>NrVatUE,omitempty"` // EU VAT number
	// or
	CountryCode string `xml:"DaneIdentyfikacyjne>KodKraju,omitempty"` // Country code outside European Union
	IDNumber    string `xml:"DaneIdentyfikacyjne>NrID,omitempty"`     // Tax ID number outside European Union
	// or
	NoID int `xml:"DaneIdentyfikacyjne>BrakID,omitempty"`

	Name                  string          `xml:"DaneIdentyfikacyjne>Nazwa,omitempty"`
	BuyerID               string          `xml:"IDNabywcy,omitempty"`
	EORI                  string          `xml:"NrEORI,omitempty"`
	Address               *Address        `xml:"Adres,omitempty"`
	CorrespondenceAddress *Address        `xml:"AdresKoresp,omitempty"`
	Contact               *ContactDetails `xml:"DaneKontaktowe,omitempty"`
	CustomerNumber        string          `xml:"NrKlienta,omitempty"`

	JST string `xml:"JST"` // JST (Jednostka Samorządu Terytorialnego = local government unit) 1 = Yes, 2 = No
	GV  string `xml:"GV"`  // GV (Group VAT) 1 = Yes, 2 = No
}

// ThirdParty defines the XML structure for KSeF third party (Podmiot3)
type ThirdParty struct {
	BuyerID               string          `xml:"IDNabywcy,omitempty"`
	EORI                  string          `xml:"NrEORI,omitempty"`
	NIP                   string          `xml:"DaneIdentyfikacyjne>NIP,omitempty"`
	InternalID            string          `xml:"DaneIdentyfikacyjne>IDWew,omitempty"`
	UECode                string          `xml:"DaneIdentyfikacyjne>KodUE,omitempty"`
	UEVatNumber           string          `xml:"DaneIdentyfikacyjne>NrVatUE,omitempty"`
	CountryCode           string          `xml:"DaneIdentyfikacyjne>KodKraju,omitempty"`
	IDNumber              string          `xml:"DaneIdentyfikacyjne>NrID,omitempty"`
	NoID                  int             `xml:"DaneIdentyfikacyjne>BrakID,omitempty"`
	Name                  string          `xml:"DaneIdentyfikacyjne>Nazwa,omitempty"`
	Address               *Address        `xml:"Adres,omitempty"`
	CorrespondenceAddress *Address        `xml:"AdresKoresp,omitempty"`
	Contact               *ContactDetails `xml:"DaneKontaktowe,omitempty"`
	Role                  string          `xml:"Rola,omitempty"`     // TRolaPodmiotu3: 1-11
	OtherRole             int             `xml:"RolaInna,omitempty"` // 1 for other role
	OtherRoleDescription  string          `xml:"OpisRoli,omitempty"` // description when OtherRole=1
	Share                 string          `xml:"Udzial,omitempty"`   // percentage share
	CustomerNumber        string          `xml:"NrKlienta,omitempty"`
}

// AuthorizedEntity defines the XML structure for KSeF authorized entity (PodmiotUpowazniony)
type AuthorizedEntity struct {
	EORI                  string   `xml:"NrEORI,omitempty"`
	NIP                   string   `xml:"DaneIdentyfikacyjne>NIP"`
	Name                  string   `xml:"DaneIdentyfikacyjne>Nazwa"`
	Address               *Address `xml:"Adres"`
	CorrespondenceAddress *Address `xml:"AdresKoresp,omitempty"`
	Email                 string   `xml:"DaneKontaktowe>EmailPU,omitempty"`
	Phone                 string   `xml:"DaneKontaktowe>TelefonPU,omitempty"`
	Role                  int      `xml:"RolaPU"` // 1=enforcement authority, 2=court bailiff, 3=tax representative
}

// newAddress gets the address data from GOBL address
func newAddress(address *org.Address) *Address {
	addressLine1 := addressLine1(address)
	var addressLine2 string
	if len(addressLine1) > 512 {
		addressLine2 = addressLine1[512:]
		addressLine1 = addressLine1[:512]
	}
	adres := &Address{
		CountryCode: string(address.Country),
		AddressL1:   addressLine1,
		AddressL2:   addressLine2,
	}

	return adres
}

// NewFavatSeller converts a GOBL Party into a KSeF seller
func NewFavatSeller(supplier *org.Party) *Seller {
	seller := &Seller{
		Name: supplier.Name,
	}
	if supplier.TaxID != nil {
		seller.VATPrefix = supplier.TaxID.Country.String()
		seller.NIP = string(supplier.TaxID.Code)
	}
	if len(supplier.Addresses) > 0 {
		seller.Address = newAddress(supplier.Addresses[0])
	}
	if len(supplier.Telephones) > 0 {
		seller.Contact = &ContactDetails{
			Phone: supplier.Telephones[0].Number,
		}
	}
	if len(supplier.Emails) > 0 {
		if seller.Contact == nil {
			seller.Contact = &ContactDetails{}
		}
		seller.Contact.Email = supplier.Emails[0].Address
	}

	return seller
}

// NewFavatBuyer converts a GOBL Party into a KSeF buyer
func NewFavatBuyer(customer *org.Party) *Buyer {

	buyer := &Buyer{
		JST: "2",
		GV:  "2",
	}

	if customer == nil {
		buyer.NoID = 1
		return buyer
	}

	if customer.TaxID == nil || len(customer.TaxID.Code) == 0 {
		// No tax ID — consumer or company without identifier
		buyer.NoID = 1
	} else if customer.TaxID.Country == l10n.PL.Tax() {
		// Polish buyer
		buyer.NIP = string(customer.TaxID.Code)
	} else if l10n.Union(l10n.EU).HasMember(customer.TaxID.Country.Code()) {
		// EU buyer (non-Polish)
		buyer.UECode = string(customer.TaxID.Country)
		buyer.UEVatNumber = string(customer.TaxID.Code)
	} else {
		// Third-country buyer with known tax ID
		buyer.CountryCode = string(customer.TaxID.Country)
		buyer.IDNumber = string(customer.TaxID.Code)
	}

	if len(customer.Addresses) > 0 {
		buyer.Address = newAddress(customer.Addresses[0])
	}

	if len(customer.Telephones) > 0 {
		buyer.Contact = &ContactDetails{
			Phone: customer.Telephones[0].Number,
		}
	}
	if len(customer.Emails) > 0 {
		if buyer.Contact == nil {
			buyer.Contact = &ContactDetails{}
		}
		buyer.Contact.Email = customer.Emails[0].Address
	}

	if customer.Name != "" {
		buyer.Name = customer.Name
	}

	if customer.Ext.Get(favat.ExtKeyJST) != "" {
		buyer.JST = customer.Ext.Get(favat.ExtKeyJST).String()
	}
	if customer.Ext.Get(favat.ExtKeyGroupVAT) != "" {
		buyer.GV = customer.Ext.Get(favat.ExtKeyGroupVAT).String()
	}

	return buyer
}

func addressLine1(address *org.Address) string {
	line1 := address.Street
	if address.Number != "" {
		line1 += " " + address.Number
	}
	if address.Block != "" {
		line1 += " " + address.Block
	}
	if address.Floor != "" {
		line1 += " " + address.Floor
	}
	if address.Door != "" {
		line1 += " " + address.Door
	}

	if address.Code.String() != "" {
		line1 += ", " + address.Code.String()
	}
	if address.Locality != "" {
		line1 += ", " + address.Locality
	}

	return line1
}

func NewThirdParties(invoice *bill.Invoice) []*ThirdParty {
	thirdParties := make([]*ThirdParty, 0, 100)

	// TODO: Reading from identities work for third parties like Group VAT or JST. However, for other third parties like issuer or recipient should be mapped from another GOBL structure.
	if len(invoice.Supplier.Identities) > 0 {
		thirdParty := newThirdPartyFromIdentity(invoice.Supplier.Identities[0])
		if thirdParty != nil {
			thirdParties = append(thirdParties, thirdParty)
		}
	}

	if invoice.Customer != nil {
		if len(invoice.Customer.Identities) > 0 {
			thirdParty := newThirdPartyFromIdentity(invoice.Customer.Identities[0])
			if thirdParty != nil {
				thirdParties = append(thirdParties, thirdParty)
			}
		}
	}

	return thirdParties
}

func newThirdPartyFromIdentity(identity *org.Identity) *ThirdParty {
	role := identity.Ext.Get(favat.ExtKeyThirdPartyRole)
	if role == "" {
		return nil
	}

	thirdParty := &ThirdParty{
		Role: role.String(),
	}

	if identity.Code == "" {
		thirdParty.NoID = 1
		return thirdParty
	}

	if identity.Country == l10n.PL.ISO() {
		thirdParty.NIP = identity.Code.String()
		return thirdParty
	}

	if l10n.Union(l10n.EU).HasMember(identity.Country.Code()) {
		thirdParty.UECode = identity.Country.String()
		thirdParty.UEVatNumber = identity.Code.String()
		return thirdParty
	}

	thirdParty.IDNumber = identity.Code.String()
	if identity.Country != "" {
		thirdParty.CountryCode = identity.Country.String()
	}

	return thirdParty
}

// ToGOBL converts a KSEF Seller to a GOBL Party (supplier).
func (s *Seller) ToGOBL() *org.Party {
	party := &org.Party{
		Name: s.Name,
	}

	// Parse tax ID
	if s.NIP != "" {
		country := l10n.PL.Tax()
		if s.VATPrefix != "" && s.VATPrefix != "PL" {
			country = l10n.Code(s.VATPrefix).Tax()
		}
		party.TaxID = &tax.Identity{
			Country: country,
			Code:    cbc.Code(s.NIP),
		}
	}

	// Parse address
	if s.Address != nil {
		party.Addresses = []*org.Address{parseAddress(s.Address)}
	}

	// Parse contact details
	if s.Contact != nil {
		if s.Contact.Email != "" {
			party.Emails = []*org.Email{{Address: s.Contact.Email}}
		}
		if s.Contact.Phone != "" {
			party.Telephones = []*org.Telephone{{Number: s.Contact.Phone}}
		}
	}

	return party
}

// ToGOBL converts a KSEF Buyer to a GOBL Party (customer).
func (b *Buyer) ToGOBL() *org.Party {
	if b.NoID == 1 && b.Name == "" && b.Address == nil && b.Contact == nil {
		return nil
	}

	party := &org.Party{
		Name: b.Name,
	}

	// Parse tax ID — skip when NoID is set
	if b.NoID != 1 {
		if b.NIP != "" {
			party.TaxID = &tax.Identity{
				Country: l10n.PL.Tax(),
				Code:    cbc.Code(b.NIP),
			}
		} else if b.UEVatNumber != "" && b.UECode != "" {
			party.TaxID = &tax.Identity{
				Country: l10n.Code(b.UECode).Tax(),
				Code:    cbc.Code(b.UEVatNumber),
			}
		} else if b.IDNumber != "" {
			country := l10n.PL.Tax()
			if b.CountryCode != "" {
				country = l10n.Code(b.CountryCode).Tax()
			}
			party.TaxID = &tax.Identity{
				Country: country,
				Code:    cbc.Code(b.IDNumber),
			}
		}
	}

	// Parse address
	if b.Address != nil {
		party.Addresses = []*org.Address{parseAddress(b.Address)}
	}

	// Parse contact details
	if b.Contact != nil {
		if b.Contact.Email != "" {
			party.Emails = []*org.Email{{Address: b.Contact.Email}}
		}
		if b.Contact.Phone != "" {
			party.Telephones = []*org.Telephone{{Number: b.Contact.Phone}}
		}
	}

	// Parse extensions
	if b.JST == "1" || b.GV == "1" {
		if party.Ext.IsZero() {
			party.Ext = tax.MakeExtensions()
		}

		if b.JST == "1" {
			party.Ext = party.Ext.Set(favat.ExtKeyJST, "1")
		}

		if b.GV == "1" {
			party.Ext = party.Ext.Set(favat.ExtKeyGroupVAT, "1")
		}
	}

	return party
}

// toIdentity converts a KSEF ThirdParty to a GOBL Identity.
func (tp *ThirdParty) toIdentity() *org.Identity {
	if tp.NoID == 1 {
		return nil
	}

	identity := &org.Identity{
		Ext: tax.MakeExtensions(),
	}

	// Set role
	if tp.Role != "" {
		identity.Ext = identity.Ext.Set(favat.ExtKeyThirdPartyRole, cbc.Code(tp.Role))
	}

	// Parse tax ID
	if tp.NIP != "" {
		identity.Country = l10n.PL.ISO()
		identity.Code = cbc.Code(tp.NIP)
	} else if tp.UEVatNumber != "" && tp.UECode != "" {
		identity.Country = l10n.ISOCountryCode(tp.UECode)
		identity.Code = cbc.Code(tp.UEVatNumber)
	} else if tp.IDNumber != "" {
		if tp.CountryCode != "" {
			identity.Country = l10n.ISOCountryCode(tp.CountryCode)
		}
		identity.Code = cbc.Code(tp.IDNumber)
	} else if tp.InternalID != "" {
		identity.Code = cbc.Code(tp.InternalID)
	}

	return identity
}

// parseAddress converts a KSEF Address to a GOBL Address.
func parseAddress(addr *Address) *org.Address {
	addressLine := addr.AddressL1
	if addr.AddressL2 != "" {
		addressLine += " " + addr.AddressL2
	}

	return &org.Address{
		Country: l10n.ISOCountryCode(addr.CountryCode),
		Street:  addressLine,
	}
}
