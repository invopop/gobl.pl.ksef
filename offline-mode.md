# Offline mode

Based on
- [article about offline mode](https://ksef.podatki.gov.pl/informacje-ogolne-ksef-20/tryb-offline24/)
- [article about offline mode due to KSeF failure](https://ksef.podatki.gov.pl/informacje-ogolne-ksef-20/tryb-offline-niedostepnosc-ksef)
- [article about KSeF failure mode](https://ksef.podatki.gov.pl/informacje-ogolne-ksef-20/tryb-awaryjny)
- [article about QR codes](https://ksef.podatki.gov.pl/informacje-ogolne-ksef-20/kody-weryfikujace-qr/)
- [QR codes documentation](https://github.com/CIRFMF/ksef-docs/blob/main/kody-qr.md)

This is a mode where invoices are not uploaded immediately to KSeF, but are stored locally and uploaded later. It can be used in the following cases:

- Network failure or other issues preventing the invoice from being uploaded to KSeF, on the side of the company / client application - in this case, the invoice must be uploaded at most on the next working day since the invoice was issued
- Failure of KSeF system itself, announced on the official website and in KSeF interface - in this case, the time extends to 7 days since the invoice was issued
- Failure of KSeF system announced publicly, where the KSeF interface and website is not available - in this case, it's not needed to upload the invoice to KSeF at all

To use this mode, it's necessary to have a KSeF **offline** certificate - see [here](authentication-requirements.md) for information about obtaining KSeF certificates.

Offline invoice has two QR codes:
- QR code with "Offline" text below
- QR code with "Certyfikat" (certificate) text below

## How to generate QR code with "Offline" text below

The code is in `api/qr.go` file in this repository, in `GenerateQrCodeURL` function. This function should be used both in online and offline mode, with a difference that:

- in online mode, the text below the QR code should be the KSeF number of the invoice
- in offline mode, the text below the QR code should be "Offline"

The URL contains:
1. Base URL, depending on environment (production, demo, test)
2. NIP (Polish tax ID)
3. Invoicing date
4. Invoice hash

## How to generate QR code with "Certyfikat" text below

Use `api/qr.go` file in this repository, in `GenerateCertificateQrCodeURL` function.

The function assembles the URL containing:

1. Base URL, depending on environment (production, demo, test)
2. Context NIP
3. Seller NIP
4. Certificate serial number
5. Invoice hash

Then, the URL is hashed and signed using the provided **offline** KSeF certificate (**not** the one used for authentication). Other certificates (qualified, KSeF online) won't work.

When using a RSA certificate, use the following parameters for signing:
- RSASSA-PSS algorithm
- Hash algorithm: SHA-256
- MGF1 with SHA-256
- Salt length: 32 bytes

When using a ECDSA certificate, use the following parameters for signing:
- Hash algorithm: SHA-256
- Curve: secp256r1
- IEEE P1363 format
