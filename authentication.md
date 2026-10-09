# Authentication

Based on:

- [Authentication in KSEF](https://github.com/CIRFMF/ksef-docs/blob/main/uwierzytelnianie.md) (in Polish)
- [XAdES digital signature](https://github.com/CIRFMF/ksef-docs/blob/main/auth/podpis-xades.md) (in Polish)
- [How to use the official .NET client to generate a test XAdES certificate](https://github.com/CIRFMF/ksef-docs/blob/main/auth/testowe-certyfikaty-i-podpisy-xades.md) (in Polish)

## Login to the KSeF API

This is translated from [Authentication in KSeF](https://github.com/CIRFMF/ksef-docs/blob/main/uwierzytelnianie.md) document:

To login, XAdES digital signature or a KSeF token is needed.

Base URL for the test environment: https://api-test.ksef.mf.gov.pl/v2

1. Submit `POST /auth/challenge` with no body, no headers, response has fields `challenge` (opaque string), `timestamp`, `timestampMs`.
2. Depending on the login method (XAdES / KSeF token) submit `POST /auth/xades-signature` (body is in XML and should contain challenge, context, signature - see below for details) or `POST /auth/ksef-token` (body contains challenge, context, KSeF token + timestamp encrypted with public key). In both cases we receive JSON response with `referenceNumber` and `authenticationToken`.
3. Keep polling `GET /auth/[referenceNumber]` with header `Bearer [authenticationToken]` - field `status` will indicate: 100 - in progress, 200 - successful, 4xx/5xx - error.
4. When the endpoint above returns status 200, send `POST /auth/token/redeem` with header `Bearer [authenticationToken]` and no body. Response contains `accessToken` and `refreshToken` + their expiration times. Redeeming can be done once - more attempts will result in 40x errors.
`accessToken` can be used for most actions in the API.
5. To get a new `accessToken`, send `POST /auth/token/refresh` with header `Bearer [refreshToken]` and no body. Response contains a new `accessToken` + expiration time.
6. List of current login sessions is at `GET /auth/sessions`.
7. To logout, send `DELETE /auth/sessions/current` or `DELETE /auth/sessions/[referenceNumber]`.

A single subject can have multiple login sessions. One login session is associated with a single context.

Note that the API documentation uses the name `referenceNumber` in other endpoints for asynchronous operations (submit and poll for status), not only for identifying login sessions.

### What is subject and context?

- Subject = who is logging in
- Context = business entity the operations are about

E.g. context = company X, subject = accountant of company X, employee of an accounting company having contract with company X, etc. This way, a single accountant or accounting company can work with multiple companies.

Using an API endpoint, a subject having appropriate permissions to a given context can provide another subject with permissions to the same context. It's possible to revoke permissions for a subject. It's also possible to mark the permissions as transferable - this is useful when company X gives permissions to accounting company Y, and company Y gives permissions to one of its employees.

### How to login using XAdES digital signature?

At first, it's necessary to have a certificate. For the testing environment, a self-signed certificate is allowed. For the production environment, it's necessary to have:
- a qualified certificate issued by a [trusted service provider recognized by EU](https://eidas.ec.europa.eu/efda/trust-services/browse/eidas/tls)
- a KSeF certificate - it's a type of certificate only for accessing KSeF and generated on demand using KSeF, intended for client applications and other automated operations.

See [./authentication-requirements.md](here) for details.

For a login request, create an XML document containing:
- challenge string
- context
- subject identifier type

The schema for the XML document is [here](https://ksef-test.mf.gov.pl/docs/v2/schemas/authv2.xsd). An example of the XML document, but with already attached signature (`ds:Signature` element) is [here](https://ksef-test.mf.gov.pl/docs/v2/index.html#tag/Uzyskiwanie-dostepu/paths/~1auth~1xades-signature/post).

Attach the XAdES signature to the XML document using the `ds:Signature` element.

Code in C# of the official KSeF client, for signing the login request, is available [here](https://github.com/CIRFMF/ksef-client-csharp/blob/main/KSeF.Client/Api/Services/SignatureService.cs#L62).

### What is the KSeF token?

**IMPORTANT: KSeF tokens are deprecated and will stop working since Jan 1 2027.**

KSEF token (separate type from authentication token, access token and refresh token - don’t confuse them!) is intended for API integration, and has specific permissions and description string (name entered by user). KSEF tokens can be revoked. Permissions on a KSEF token cannot be changed - to add new permissions, it's necessary to create a new token.

Users logged in with XAdES can create, list and delete KSEF tokens using the API.

## How to obtain KSeF's public key

To obtain the public key certificate, use `GET /security/public-key-certificates`.

[Documentation here](https://ksef-test.mf.gov.pl/docs/v2/index.html#tag/Certyfikaty-klucza-publicznego)

Public key is needed to:
1. Login with KSeF token using the `POST /auth/ksef-token` endpoint.
2. Encrypt a symmetric AES key when uploading invoices (in both online and batch formats) and exporting (batch) incoming invoices. File upload and export endpoints don't require `accessToken`, but they accept or return chunks of the data respectively encrypted with the provided symmetric AES key. Online upload endpoint is a regular HTTP endpoint using `accessToken` for authentication, but also requires providing a key, and the invoice must be encrypted with that key.

## How to authorize a non-Polish company to access KSeF

If you are a Polish company X, having a contract with company Y based in another European Union country, it's possible to give access permissions to company Y to see and upload invoices:

1. Company Y needs to obtain an appropriate certificate - a qualified EU certificate
2. The Polish company X needs to give company Y permissions. It's possible to do this through the API endpoint for this purpose, `permissions/eu-entities/administration/grants`, where company X provides company Y's certificate fingerprint, EU VAT number and company name. It's also possible to do this through the web interface - [a video showing how to do it is here](https://youtu.be/COXvohndNCA).
3. After that, company Y can login to KSeF API using the qualified EU certificate, and providing context identifier (NipVatUe) containing company X's NIP (Polish business entity identifier) and Y's EU VAT number.
