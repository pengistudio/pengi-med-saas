# XAdES Signer Canonicalization Fix — Design

## Context

`apps/sri-xml-signer` is the Node/Express microservice that XAdES-BES-signs
Ecuador SRI (tax authority) electronic invoices before they're submitted for
authorization. It delegates entirely to the third-party npm package
`osodreamer-sri-xml-signer@0.1.3`.

**Confirmed root cause (via independent verification, not guesswork):**
`osodreamer`'s internal `ExclusiveCanonicalisation` class produces
Exclusive-C14N output that does not match a spec-compliant canonicalizer.
Verified with `xmlsec1` (a mature, standards-based XML-DSig verifier,
external to both our code and the suspect library) against a real signed
invoice from this system: `Failure reason: REFERENCE` — the digest of the
`#comprobante` Reference (the invoice body itself) does not match. Two
targeted patches to the vendored package (declaring the correct Transform,
canonicalizing `SignedInfo` before signing) were applied and independently
re-verified with `xmlsec1` — both still fail the same way, isolating the bug
to the canonicalization *algorithm's output* itself, not a missing
declaration or an unsigned-vs-signed ordering bug.

**Corroborating evidence:** in this system's entire history, **zero**
invoices have ever reached SRI's `authorized` status — the furthest any has
gotten is `validated` (SRI's `/validate` step, which accepts the document
structurally — schema/WSDL-wise — via SOAP, without deep signature
verification). This is not a regression from anything recent; the signing
math has likely never worked.

**Decision (confirmed with the project owner):** replace only the signing
engine — the part that builds the XAdES-BES `<ds:Signature>` block and
computes the cryptographic digests/signature — with one built on
`xml-crypto@6.1.2` (already resolved in this project's lockfile as a
transitive dependency via `soap`, so no new supply-chain surface). The three
other endpoints (`/generate`, `/validate/:env`, `/authorization/:env`) are
SOAP calls to SRI with no canonicalization involved, have never shown this
failure mode, and keep delegating to `osodreamer-sri-xml-signer` unchanged.

**Revised during planning — the bug is systemic, not a single missing
declaration.** Reading `osodreamer`'s full signing flow (including the two
already-applied patches) surfaced that the `SignedProperties` and
`Certificate`/`KeyInfo` References are *never* canonicalized before being
digested either — only the raw builder string is hashed, with no
`<ds:Transforms>` declared for either. Only `#comprobante` and (after this
investigation's second patch) `SignedInfo` go through any canonicalization
at all. This means the earlier plan of "port the existing builders and keep
patching individual gaps" carries the same risk that already cost two failed
fix rounds. Confirmed with the project owner: instead of hand-rolling
canonicalization for each Reference again, every digest and the final
signature computation goes through `xml-crypto`'s own tested primitives —
its public `getCanonXml(transforms, node)` canonicalizer and its
`RsaSha1` signature algorithm — rather than `xml-crypto`'s full
`computeSignature()` auto-orchestration. `computeSignature()` doesn't have a
XAdES-aware extension point for injecting `SignedProperties`/`Object` into
the `<ds:Signature>` it builds, so reusing only its canonicalization/signing
primitives — while keeping full manual control over XAdES assembly, which
is already schema-valid per SRI's own `/validate` acceptance — avoids that
chicken-and-egg problem entirely.

## Goal

Replace `apps/sri-xml-signer`'s `/sign` implementation with a locally-owned
XAdES-BES signer that produces a signature `xmlsec1 verify` accepts, without
changing the HTTP contract that `apps/api` (Go) depends on.

## Non-goals

- Touching `/generate`, `/validate/:env`, `/authorization/:env` — these stay
  as thin delegations to `osodreamer-sri-xml-signer`'s `generateXmlInvoice`,
  `validateXml`, `authorizeXml`.
- Removing `osodreamer-sri-xml-signer` as a dependency (still used for the
  three endpoints above, and its P12/SOAP-adjacent types may still be
  useful as reference).
- Changing anything in `apps/api` (Go) — the request/response JSON shapes
  documented below are the contract and do not change.
- Supporting credit-note/debit-note signing differently than invoices — SRI
  documents (factura/notaCredito/notaDebito) all go through the same
  `/sign` endpoint today with a pre-built unsigned XML string; this design
  covers signing any well-formed SRI comprobante XML, not just invoices.

## Architecture

A new, independently-testable module owns the entire XAdES-BES construction
and signing flow:

```
apps/sri-xml-signer/src/services/xades-signer/
├── certificate.ts     — P12 → {privateKeyPem, certificateX509 (base64 DER),
│                         modulus, exponent, issuerName, serialNumber,
│                         certDigestSha1Base64} (node-forge; privateKeyPem is
│                         PEM-encoded since xml-crypto's RsaSha1 expects a
│                         crypto.KeyLike, not a forge key object)
├── canonicalize.ts     — canonicalizeFragment(xml: string): string — parses
│                         xml with @xmldom/xmldom and runs it through a
│                         `xml-crypto` SignedXml instance's public
│                         `getCanonXml(["http://www.w3.org/2001/10/xml-exc-c14n#"], node)`.
│                         Every digest in sign.ts (comprobante,
│                         SignedProperties, KeyInfo, SignedInfo) goes through
│                         this same function — no bespoke canonicalization
│                         code of our own.
├── builders.ts         — pure string-template functions: buildKeyInfo,
│                         buildSignedProperties, buildSignedInfo,
│                         buildXadesObject (the QualifyingProperties/Object
│                         block), assembleSignatureElement — ported from
│                         osodreamer's (schema-valid, SRI-accepted) templates,
│                         with every Reference's Transforms/CanonicalizationMethod
│                         declaring exc-c14n consistently
├── sign.ts             — signXades(xmlToSign, p12Buffer, password): Promise<string>
│                         — the orchestrator; the only export consumed by
│                         signer.service.ts. Uses canonicalize.ts for every
│                         digest and xml-crypto's RsaSha1 algorithm
│                         (`new signedXml.SignatureAlgorithms["http://www.w3.org/2000/09/xmldsig#rsa-sha1"]()`)
│                         for the final signature computation over the
│                         canonicalized SignedInfo.
└── xades-signer.test.ts / canonicalize.test.ts — see Testing
```

`signer.service.ts`'s `sign` function changes from delegating to
`osodreamer-sri-xml-signer`'s `signXml` to calling
`xadesSigner.signXades(...)`. `generate`, `validate`, `authorize` are
untouched.

## Data flow (signing one document)

1. Caller (`apps/api`, via HTTP `POST /sign`) sends `p12Buffer`, `password`,
   `xmlBuffer` (unsigned SRI comprobante XML, e.g. `<factura>...`) — exactly
   as today; `signer.handler.ts`'s request parsing/validation is unchanged.
2. `certificate.ts` decodes the P12 (via `node-forge`, already resolved in
   the lockfile) into the private key, X509 cert (base64 DER for
   `<ds:X509Certificate>`), RSA modulus/exponent (for `<ds:RSAKeyValue>`),
   issuer name and serial number (for XAdES `SigningCertificate`), and a
   SHA1 digest of the cert (for both `CertDigest` in SignedProperties and
   the `#Certificate...` Reference).
3. `canonicalize.ts#canonicalizeFragment` canonicalizes the raw unsigned
   XML's root element (Exclusive C14N, no comments, via `xml-crypto`'s
   `getCanonXml`) and SHA1-hashes it → `sha1_comprobante`.
4. `builders.ts#buildSignedProperties` builds the XAdES `SignedProperties`
   fragment (signing time, cert digest, issuer/serial, data-object format)
   with a fresh set of IDs. **Canonicalize it** (same `canonicalizeFragment`)
   before hashing → `sha1_SignedProperties` (osodreamer hashed this raw,
   uncanonicalized — one of the systemic gaps found during planning).
5. `builders.ts#buildKeyInfo` builds the `KeyInfo` fragment (X509Data +
   RSAKeyValue). **Canonicalize it** before hashing → `sha1_certificado`
   (same fix — osodreamer hashed this raw too).
6. `builders.ts#buildSignedInfo` builds `SignedInfo` with all three
   References (`SignedProperties`, `Certificate`, `#comprobante`), each with
   the *correct, consistent* Transforms/DigestMethod/DigestValue — the
   `#comprobante` Reference declares both `enveloped-signature` and
   `exc-c14n` Transforms, the `SignedProperties` and `Certificate`
   References each declare an explicit `exc-c14n` Transform (previously
   absent), and `SignedInfo`'s own `CanonicalizationMethod` declares
   `exc-c14n`.
7. `canonicalize.ts#canonicalizeFragment` canonicalizes the *built*
   `SignedInfo` fragment. `xml-crypto`'s `RsaSha1.getSignature(canonicalSignedInfo, privateKeyPem)`
   RSA-SHA1-signs the canonicalized bytes → `SignatureValue`. (This step —
   canonicalizing SignedInfo before signing — is mandatory per XML-DSig and
   was previously entirely skipped upstream; this investigation's second
   patch fixed it in the vendored package, and it carries over here as the
   same fix, now using a trusted canonicalizer.)
8. `builders.ts#assembleSignatureElement` combines `SignedInfo`,
   `SignatureValue`, `KeyInfo`, and the XAdES `Object`/`QualifyingProperties`
   (wrapping `SignedProperties`) into the final `<ds:Signature>` element.
9. Parse the original unsigned XML with `@xmldom/xmldom`, append
   `<ds:Signature>` as the last child of the document root, serialize back
   to a string. Return it as `signedXml`.
10. `signer.handler.ts` responds exactly as today:
    `{ data: { xml: signedXml }, message: "Operación exitosa", statusCode: 200 }`.

## Error handling

- P12 decode failure (wrong password, corrupt file) → thrown error with a
  clear message (`"Invalid P12 file or password"`), caught by the existing
  `errorMiddleware` → `500` with `err.message` surfaced, same as today's
  behavior for any `signXml` failure (no behavior change from the caller's
  perspective — `apps/api` already treats any non-200 from `/sign` as a
  pipeline failure and records it on the invoice).
- Malformed input XML (fails to parse) → thrown error, same path.
- No silent fallback signature paths — a signing failure must always throw,
  never return a `signedXml` that wasn't actually produced by a verified
  flow. (There is no plausible "partial success" state for this operation.)

## Testing

Today `apps/sri-xml-signer` has zero tests. This module gets:

- **`canonicalize.test.ts`** — unit tests against a couple of known,
  hand-verified Exclusive C14N input/output pairs (e.g. the classic W3C
  c14n test vectors, or a minimal fixture built from this project's own
  namespace-declaration situation: an element with unused `xmlns:ds`/
  `xmlns:xsi` on an ancestor, confirming they're correctly dropped from the
  exclusive-canonical form). Pure function, no I/O, fast.
- **`xades-signer.test.ts`** — integration test: sign a small fixture SRI
  comprobante XML with a throwaway/test P12 certificate (self-signed, generated
  for the test — no real tenant credentials involved), then **independently
  verify the result with `xmlsec1`** (shelled out via `child_process`, or
  skipped with a clear message if `xmlsec1` isn't on the PATH in a given CI
  environment) asserting exit code 0 / `Verification status: OK`. This is
  the same independent tool used to diagnose the original bug — using it in
  the test suite means a future regression is caught by `pnpm test`, not by
  a real invoice failing in front of a user weeks later.
- Existing manual verification path stays available too: the dev Docker
  Compose stack + republishing an `invoice_tasks` RabbitMQ message against
  a real invoice, as used during this investigation.

## Rollout

No feature flag needed — this is a bug fix for a pipeline that has never
successfully authorized a single document in this system's history, so
there is no working baseline behavior to preserve or migrate away from
gradually. Once `xmlsec1`-verified locally, it ships as a normal deploy of
`apps/sri-xml-signer`.

## Open questions for the implementation plan (not this design)

- Exact node-forge API calls for P12 → modulus/exponent/issuer-name
  extraction, and the per-CA `X509IssuerName` formatting quirks noted
  during investigation (the earlier exploration found `osodreamer` dispatches
  on 4 different issuer strategies — ANFAC, Banco Central, Security Data,
  Uanataca — because Ecuador's certifying authorities format
  `X509IssuerName` differently). The plan must confirm which of these are
  actually needed (this tenant's cert is Security Data; whether to port all
  4 strategies or only the ones actually exercised is a plan-level call,
  not a design-level one — recommend porting all 4 since the service signs
  for multiple tenants with potentially different CAs).
- Whether `xmlsec1` is available in the CI/test environment this project
  uses, and what the test should do if not (skip with a warning vs. hard
  fail) — a plan-level, not design-level, decision.
