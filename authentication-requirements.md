# Authentication requirements

Based on:
- [Authentication in KSEF](https://github.com/CIRFMF/ksef-docs/blob/main/uwierzytelnianie.md) (in Polish)

Environments:

- Test environment is an environment where a self-signed certificate can be used, and login information and invoice data should be fake.
- Demo environment is an environment where real login information should be used, but invoice data should be fake.

## Test environment

Certificates in the test environment can be self-signed.

There is an online process to register a company:

- [Test Company login](https://ksef-test.mf.gov.pl/web/login)
- [Generate a fake NIP (tax ID)](http://generatory.it/)

Once inside the test environment, you can create an Authorization token to use to make requests to the API.

### How to generate a test certificate

Based on [this](https://github.com/CIRFMF/ksef-docs/blob/main/auth/testowe-certyfikaty-i-podpisy-xades.md) document.

There is a CLI application in .NET that allows to generate a self-signed certificate that can be used to log into the test environment. To run the application:

1. Install .NET 10.0 SDK
2. Download the repository: `git clone https://github.com/CIRFMF/ksef-client-csharp.git`
3. Go to the application directory: `cd ksef-client-csharp/KSeF.Client.Tests.CertTestApp`
4. Run the application: `dotnet run --framework net10.0 --output file --nip 8976111986 --no-startup-warnings`
5. The application will generate a self-signed certificate and save it to the current directory. It will generate two files: `cert-{timestamp}.pfx` and `cert-{timestamp}.cer`.

## Production and demo environments

Authentication with production and demo KSeF environments can be done:

1. Using a qualified digital certificate. Qualified means that it's issued by trusted service providers on the European Union [Trusted List](https://eidas.ec.europa.eu/efda/trust-services/browse/eidas/tls).
2. Using a KSeF certificate. KSeF certificates are generated on demand, and intended for client applications, so that client applications won't have access to the qualified certificate.
3. Using ePUAP (a system where individuals with PESEL - Polish individual personal number - can access various government services).
4. Using a KSeF token, obtained similarly to KSeF certificates (deprecated, will work until end of 2026).

### What is a KSeF certificate and how to obtain it

It's a type of certificate that is:
- generated on demand
- intended for KSeF client applications, so that client applications can save it for later use without having to use owner's qualified certificate
- revocable by the owner in case of e.g. security breach, or when the company stops using the client application - a revoked certificate cannot be used again and a new one must be generated.

It can be obtained using using the following methods:
1. Company owner / approved employee logs into Aplikacja Podatnika ("Taxpayer's Application") - it's both a website and a mobile application - and generates a KSeF certificate there, downloads it to disk, and uploads it (certificate file, private key, password) to the client application. This application is available since February 2026. This is the simplest method for non-technical users.
2. Company owner / approved employee logs into MCU (moduł certyfikatów i uprawnień - certificate and permissions module), and does the same thing as above. MCU is available until the end of January 2026, and it's very similar to Aplikacja Podatnika.
3. Owner of qualified certificate uses it to log into KSeF API and calls and endpoint to generate a KSeF certificate. This is possible to do using this library (gobl.ksef) if the certificate is in a `.p12` / `.pfx` file.
4. Using ePUAP - described below, as it's a more complex process.

There are two types of KSeF certificate:
- Online - for logging into API
- Offline - for signing invoices in case of KSeF system unavailability - see [offline mode](offline-mode.md) for details

Both certificate types must be obtained separately.

A video tutorial (in Polish) about using Aplikacja Podatnika is available [here](https://www.youtube.com/watch?v=KkyNw_tBN2s).

### Logging into API using ePUAP

1. User logs into ePUAP
2. Client application generates an XML file with the authentication request
3. User uses ePUAP's feature to sign an XML file - uploads the XML file to ePUAP application
4. ePUAP returns the signed XML file
5. User uploads the signed XML file to the client application
6. Client application proceeds with login with the signed XML file
7. Client application, being authenticated as the user, generates a KSeF certificate and saves it for later use

Steps from 2 to 6 must be done in 5 minutes, otherwise the authentication challenge expires. See [here](./authentication.md) for details about the authentication process.

### Creating a KSeF certificate using this library

`gobl.ksef` library exposes functions that allow requesting a KSeF certificate.

How to use them:
```go
package main

import (
    "context"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "log"
    "time"

    ksef "github.com/invopop/gobl.ksef/api"
)

func main() {
    ctx := context.Background()

    // Load the certificate used to authenticate against KSeF (not the new one yet).
    certificateData, err := ksef.LoadCertificate("./path/to/auth-certificate.pfx")
    if err != nil {
        log.Fatal(err)
    }

    client := ksef.NewClient(&ksef.ContextIdentifier{Nip: "1234567890"}, certificateData)
    if err := client.Authenticate(ctx); err != nil {
        log.Fatal(err)
    }

    // Create ECDSA key pair that will represent the new KSeF certificate
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        log.Fatal(err)
    }

    // CreateKsefCertificate runs the whole flow: fetch data, generate CSR, submit, and poll.
    certEntry, err := client.CreateKsefCertificate(
        ctx,
        "My Auth Cert",
        ksef.CertificateTypeAuthentication,
        privateKey,
        nil, // optional validFrom
    )
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Certificate request completed, serial number %s", certEntry.CertificateSerialNumber)

    // Once you obtain the certificate serial number from subsequent steps,
    // you can revoke it at any time:
    if err := client.RevokeCertificate(ctx, certEntry.CertificateSerialNumber); err != nil {
        log.Fatal(err)
    }
}
```
