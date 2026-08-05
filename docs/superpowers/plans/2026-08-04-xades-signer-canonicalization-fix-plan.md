# XAdES Signer Canonicalization Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `apps/sri-xml-signer`'s `/sign` implementation with a locally-owned XAdES-BES signer whose output `xmlsec1 verify` accepts, fixing a confirmed canonicalization bug in the vendored `osodreamer-sri-xml-signer` package that has caused every SRI authorization attempt in this system's history to fail with "FIRMA INVALIDA".

**Architecture:** New module `apps/sri-xml-signer/src/services/xades-signer/` owns P12 extraction, XAdES fragment building, and canonicalization/signing, replacing only the `sign` delegation in `signer.service.ts`. Every digest (comprobante, SignedProperties, KeyInfo, SignedInfo) is computed by first canonicalizing with `xml-crypto`'s public `getCanonXml()` primitive and then SHA1-hashing the result — no hand-rolled canonicalization anywhere. RSA-SHA1 signing uses `xml-crypto`'s `RsaSha1` algorithm class. `/generate`, `/validate/:env`, `/authorization/:env` are untouched.

**Tech Stack:** TypeScript, Express 5, `xml-crypto@6.1.2`, `@xmldom/xmldom`, `node-forge`, `nodemon`/`ts-node` (dev), pnpm.

## Global Constraints

- `xmlsec1 verify --id-attr:id factura --enabled-key-data x509 --insecure <file>` must report `Verification status: OK` on a document signed by this module — this is the ground truth used throughout, not SRI's opaque error message.
- The HTTP contract of `POST /sign` (request shape, response shape, status codes) does not change — `apps/api` (Go) is an unmodified consumer.
- `/generate`, `/validate/:env`, `/authorization/:env` keep delegating to `osodreamer-sri-xml-signer` unchanged — do not touch `signerService.generate`, `.validate`, `.authorize`.
- Every digest in the signing flow (comprobante, SignedProperties, KeyInfo, SignedInfo) must be computed over **canonicalized** content — never hash a raw, uncanonicalized builder string (this was the root cause).
- Exclusive C14N (`http://www.w3.org/2001/10/xml-exc-c14n#`) is used consistently everywhere a `CanonicalizationMethod`/`Transform` is declared — never mix in the inclusive C14N URI (`http://www.w3.org/TR/2001/REC-xml-c14n-20010315`).
- Ecuador does not observe DST; `SigningTime` is always UTC-5, matching the existing `ClockImplement` behavior being ported.

---

## Task 1: `canonicalize.ts` — the shared canonicalization primitive

**Files:**
- Create: `apps/sri-xml-signer/src/services/xades-signer/canonicalize.ts`
- Test: `apps/sri-xml-signer/src/services/xades-signer/canonicalize.test.ts`
- Modify: `apps/sri-xml-signer/package.json` (add `xml-crypto`, `@xmldom/xmldom`, `node-forge` to `dependencies`; add `@types/node-forge`, a test runner — see Step 6 — to `devDependencies`)

**Interfaces:**
- Produces: `canonicalizeFragment(xml: string): string` — parses `xml` (any single-root-element well-formed XML string) and returns its Exclusive-C14N canonical form (no comments). Consumed by Task 3's `sign.ts` for every digest.

- [ ] **Step 1: Write the failing test**

```ts
// apps/sri-xml-signer/src/services/xades-signer/canonicalize.test.ts
import { describe, expect, it } from "vitest";
import { canonicalizeFragment } from "./canonicalize";

describe("canonicalizeFragment", () => {
	it("drops namespace declarations unused by the element or its descendants", () => {
		// Mirrors this project's actual bug: <factura> declares xmlns:ds/xmlns:xsi
		// but nothing in the body uses those prefixes — exclusive C14N must drop them.
		const xml =
			'<factura id="comprobante" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><infoTributaria><ambiente>1</ambiente></infoTributaria></factura>';
		const result = canonicalizeFragment(xml);
		expect(result).not.toContain("xmlns:ds");
		expect(result).not.toContain("xmlns:xsi");
		expect(result).toBe(
			'<factura id="comprobante"><infoTributaria><ambiente>1</ambiente></infoTributaria></factura>',
		);
	});

	it("keeps a namespace declaration that is actually used by a descendant", () => {
		const xml =
			'<ds:SignedInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:etsi="http://uri.etsi.org/01903/v1.3.2#" Id="x"><ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"></ds:CanonicalizationMethod></ds:SignedInfo>';
		const result = canonicalizeFragment(xml);
		expect(result).toContain("xmlns:ds=");
		expect(result).not.toContain("xmlns:etsi=");
	});

	it("expands self-closing empty elements to open/close tag pairs", () => {
		const xml = '<ds:Transform xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>';
		const result = canonicalizeFragment(xml);
		expect(result).toBe(
			'<ds:Transform xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"></ds:Transform>',
		);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/sri-xml-signer && npx vitest run src/services/xades-signer/canonicalize.test.ts`
Expected: FAIL — `Cannot find module './canonicalize'` (file doesn't exist yet).

- [ ] **Step 3: Implement `canonicalize.ts`**

```ts
// apps/sri-xml-signer/src/services/xades-signer/canonicalize.ts
import { DOMParser } from "@xmldom/xmldom";
import { SignedXml } from "xml-crypto";

const EXCLUSIVE_C14N = "http://www.w3.org/2001/10/xml-exc-c14n#";

const parser = new DOMParser({
	errorHandler: () => {
		// osodreamer's XmlDomContext silences parser warnings the same way;
		// malformed input still yields no documentElement below and throws there.
	},
});

/**
 * Canonicalize a single well-formed XML fragment (one root element) using
 * Exclusive C14N (no comments), via xml-crypto's own canonicalizer — never
 * a hand-rolled implementation. Every digest in sign.ts goes through this.
 */
export function canonicalizeFragment(xml: string): string {
	const doc = parser.parseFromString(xml, "text/xml");
	const root = doc.documentElement;
	if (!root) {
		throw new Error("canonicalizeFragment: input is not well-formed XML");
	}
	const signedXml = new SignedXml();
	return signedXml.getCanonXml([EXCLUSIVE_C14N], root);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/sri-xml-signer && npx vitest run src/services/xades-signer/canonicalize.test.ts`
Expected: PASS, 3/3.

- [ ] **Step 5: Add dependencies**

In `apps/sri-xml-signer/package.json`, add to `dependencies`:

```json
"xml-crypto": "^6.1.2",
"@xmldom/xmldom": "^0.8.13",
"node-forge": "^1.4.0"
```

and to `devDependencies`:

```json
"@types/node-forge": "^1.3.11",
"vitest": "^2.1.9"
```

Add a `"test": "vitest run"` script (replacing the current no-op stub).

- [ ] **Step 6: Install and re-run**

Run: `cd apps/sri-xml-signer && CI=true pnpm install && npx vitest run src/services/xades-signer/canonicalize.test.ts`
Expected: PASS, 3/3, with `xml-crypto`/`@xmldom/xmldom`/`node-forge` now explicit (not phantom-transitive) dependencies.

- [ ] **Step 7: Commit**

```bash
git add apps/sri-xml-signer/package.json apps/sri-xml-signer/pnpm-lock.yaml \
  apps/sri-xml-signer/src/services/xades-signer/canonicalize.ts \
  apps/sri-xml-signer/src/services/xades-signer/canonicalize.test.ts
git commit -m "feat(sri-xml-signer): canonicalizeFragment primitive backed by xml-crypto"
```

---

## Task 2: `certificate.ts` — P12 extraction

**Files:**
- Create: `apps/sri-xml-signer/src/services/xades-signer/certificate.ts`
- Test: `apps/sri-xml-signer/src/services/xades-signer/certificate.test.ts`

**Interfaces:**
- Produces:
  ```ts
  export interface CertificateData {
  	privateKeyPem: string;
  	certificateX509: string; // base64 DER, for <ds:X509Certificate>
  	certDigestSha1Base64: string; // SHA1(DER bytes), base64 — for CertDigest + #Certificate reference input
  	issuerName: string; // RFC4514-ish DN string, for <ds:X509IssuerName>
  	serialNumber: string; // decimal string, for <ds:X509SerialNumber>
  	modulus: string; // base64, for <ds:Modulus>
  	exponent: string; // base64, for <ds:Exponent>
  }
  export function extractCertificateData(p12Buffer: Buffer, password: string): CertificateData;
  ```
  Consumed by Task 3's `sign.ts`.

- [ ] **Step 1: Write the failing test**

The test builds its own self-signed test certificate/P12 with `node-forge` — no real tenant credentials involved. The CA-strategy dispatch in `certificate.ts` (Step 3) reads the 3rd attribute of the issuer DN when no PKCS12 friendlyName bag is present (mirroring `osodreamer`'s exact fallback), so the test issuer's 3rd DN attribute is set to a value matching the "Security Data" strategy.

```ts
// apps/sri-xml-signer/src/services/xades-signer/certificate.test.ts
import forge from "node-forge";
import { describe, expect, it } from "vitest";
import { extractCertificateData } from "./certificate";

function buildTestP12(): { buffer: Buffer; password: string } {
	const keys = forge.pki.rsa.generateKeyPair(2048);
	const cert = forge.pki.createCertificate();
	cert.publicKey = keys.publicKey;
	cert.serialNumber = "01";
	cert.validity.notBefore = new Date();
	cert.validity.notAfter = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000);
	// 3rd attribute (index 2) is what certificate.ts falls back to for CA-strategy
	// dispatch when there's no PKCS12 friendlyName bag — matches SecurityDataStrategy.
	const attrs = [
		{ shortName: "C", value: "EC" },
		{ shortName: "O", value: "TEST" },
		{ shortName: "CN", value: "SECURITY DATA TEST CA" },
	];
	cert.setSubject(attrs);
	cert.setIssuer(attrs);
	cert.sign(keys.privateKey, forge.md.sha1.create());

	const password = "test-password";
	const p12Asn1 = forge.pkcs12.toPkcs12Asn1(keys.privateKey, [cert], password, {
		algorithm: "3des",
	});
	const p12Der = forge.asn1.toDer(p12Asn1).getBytes();
	return { buffer: Buffer.from(p12Der, "binary"), password };
}

describe("extractCertificateData", () => {
	it("extracts a usable private key, certificate and RSA key material", () => {
		const { buffer, password } = buildTestP12();
		const data = extractCertificateData(buffer, password);

		expect(data.privateKeyPem).toContain("BEGIN RSA PRIVATE KEY");
		expect(data.certificateX509.length).toBeGreaterThan(0);
		expect(data.certDigestSha1Base64.length).toBeGreaterThan(0);
		expect(data.issuerName).toContain("SECURITY DATA TEST CA");
		expect(data.serialNumber).toBe("1");
		expect(data.modulus.length).toBeGreaterThan(0);
		expect(data.exponent.length).toBeGreaterThan(0);
	});

	it("throws a clear error for the wrong password", () => {
		const { buffer } = buildTestP12();
		expect(() => extractCertificateData(buffer, "wrong-password")).toThrow();
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/sri-xml-signer && npx vitest run src/services/xades-signer/certificate.test.ts`
Expected: FAIL — `Cannot find module './certificate'`.

- [ ] **Step 3: Implement `certificate.ts`**

Ported from `osodreamer-sri-xml-signer`'s `CertificateProviderImplement` + its 4 CA strategies (`AnfacStrategy`, `BancoCentralStrategy`, `SecurityDataStrategy`, `UanatacaStrategy`), simplified: modulus/exponent conversion goes through one safe, even-length-padded hex→base64 helper instead of the original's two different (one of them regex-chunking, parity-fragile) implementations — see Global Constraints; this is a deliberate correctness improvement, not a behavior port.

```ts
// apps/sri-xml-signer/src/services/xades-signer/certificate.ts
import forge from "node-forge";

export interface CertificateData {
	privateKeyPem: string;
	certificateX509: string;
	certDigestSha1Base64: string;
	issuerName: string;
	serialNumber: string;
	modulus: string;
	exponent: string;
}

// hex → base64 of the raw bytes, with even-length padding (a hex string with
// an odd digit count is not a valid byte sequence — pad a leading zero).
function hexToBase64(hex: string): string {
	const padded = hex.length % 2 === 0 ? hex : `0${hex}`;
	return Buffer.from(padded, "hex").toString("base64");
}

function bigIntegerToBase64(value: forge.jsbn.BigInteger): string {
	return hexToBase64(value.toString(16));
}

interface CaStrategy {
	supports(friendlyName: string): boolean;
	getIssuerName(cert: forge.pki.Certificate): string;
}

function escapeRfc4514(value: string): string {
	let v = value
		.replace(/\\/g, "\\\\")
		.replace(/"/g, '\\"')
		.replace(/,/g, "\\,")
		.replace(/\+/g, "\\+")
		.replace(/;/g, "\\;")
		.replace(/</g, "\\<")
		.replace(/>/g, "\\>")
		.replace(/#/g, "\\#")
		.replace(/=/g, "\\=");
	if (v.startsWith(" ")) v = `\\${v}`;
	if (v.endsWith(" ")) v = `${v.slice(0, -1)}\\ `;
	return v;
}

const OID_TO_RFC4514: Record<string, string> = {
	"2.5.4.6": "C",
	"2.5.4.10": "O",
	"2.5.4.11": "OU",
	"2.5.4.3": "CN",
	"2.5.4.7": "L",
	"2.5.4.8": "ST",
	"2.5.4.9": "STREET",
	"2.5.4.5": "SERIALNUMBER",
	"1.2.840.113549.1.9.1": "E",
};

// A local shape for DN attributes instead of forge's own type name — the
// exact interface name for issuer.attributes[i] varies across
// @types/node-forge versions, but the fields used here (type/shortName/
// name/value) are stable across all of them.
interface DnAttribute {
	type?: string;
	shortName?: string;
	name?: string;
	value?: unknown;
}

function rfc4514AttrName(a: DnAttribute): string {
	return (
		OID_TO_RFC4514[a.type ?? ""] ??
		a.shortName?.toUpperCase() ??
		a.name?.toUpperCase() ??
		a.type ??
		""
	);
}

// ANFAC formats the issuer DN as RFC4514 (escaped, reversed attribute order).
const anfacStrategy: CaStrategy = {
	supports: (friendlyName) => /ANFAC/i.test(friendlyName),
	getIssuerName: (cert) =>
		cert.issuer.attributes
			.slice()
			.reverse()
			.map((a) => `${rfc4514AttrName(a)}=${escapeRfc4514(String(a.value))}`)
			.join(","),
};

// Banco Central and Security Data format the issuer DN as a simple reversed
// shortName=value join.
const shortNameJoinStrategy = (namePattern: RegExp): CaStrategy => ({
	supports: (friendlyName) => namePattern.test(friendlyName),
	getIssuerName: (cert) =>
		cert.issuer.attributes
			.slice()
			.reverse()
			.map((a) => `${a.shortName}=${a.value}`)
			.join(","),
});

const bancoCentralStrategy = shortNameJoinStrategy(/BANCO CENTRAL/i);
const securityDataStrategy = shortNameJoinStrategy(/SECURITY DATA/i);

// Uanataca: reversed join like the others, but attributes with no shortName
// (some OIDs Forge doesn't have a friendly name for) are hex-encoded as a
// DirectoryString (tag 0x0c = UTF8String) instead of dropped.
function hexEncodeUtf8DirectoryString(value: string): string {
	const utf8Bytes = forge.util.encodeUtf8(value);
	const hex = forge.util.bytesToHex(utf8Bytes);
	return `#0c${utf8Bytes.length.toString(16).padStart(2, "0")}${hex}`;
}

const uanatacaStrategy: CaStrategy = {
	supports: (friendlyName) => /UANATACA/i.test(friendlyName),
	getIssuerName: (cert) =>
		cert.issuer.attributes
			.slice()
			.reverse()
			.filter((a) => a.shortName || a.type)
			.map((a) =>
				a.shortName
					? `${a.shortName}=${a.value}`
					: `${a.type}=${hexEncodeUtf8DirectoryString(String(a.value))}`,
			)
			.join(","),
};

const CA_STRATEGIES: CaStrategy[] = [
	bancoCentralStrategy,
	securityDataStrategy,
	uanatacaStrategy,
	anfacStrategy,
];

function resolveIssuerName(cert: forge.pki.Certificate, friendlyName: string): string {
	const strategy = CA_STRATEGIES.find((s) => s.supports(friendlyName));
	if (!strategy) {
		// Fall back to the same simple reversed-join format used by the two
		// most common Ecuadorian CAs rather than throwing — every SRI cert
		// this service has seen resolves via bancoCentral/securityData; this
		// keeps a new/unrecognized CA working instead of hard-failing signing.
		return shortNameJoinStrategy(/.*/).getIssuerName(cert);
	}
	return strategy.getIssuerName(cert);
}

export function extractCertificateData(p12Buffer: Buffer, password: string): CertificateData {
	// node-forge's PKCS12 typings (@types/node-forge) are loosely/inconsistently
	// typed across versions — using `any` here for the raw ASN.1/PKCS12 bag
	// structures is deliberate, not a shortcut; forge.pki.Certificate and
	// forge.jsbn.BigInteger (used below) are the well-typed, stable parts of
	// the surface and keep their real types.
	const p12Asn1 = forge.asn1.fromDer(forge.util.createBuffer(p12Buffer.toString("binary")));
	// biome-ignore lint/suspicious/noExplicitAny: forge's PKCS12 bag types are unreliable across @types/node-forge versions
	let p12: any;
	try {
		p12 = forge.pkcs12.pkcs12FromAsn1(p12Asn1, password);
	} catch (err) {
		throw new Error("Invalid P12 file or password", { cause: err });
	}

	const keyBags = p12.getBags({ bagType: forge.pki.oids.pkcs8ShroudedKeyBag });
	const certBags = p12.getBags({ bagType: forge.pki.oids.certBag });
	const keyBag = keyBags[forge.pki.oids.pkcs8ShroudedKeyBag]?.[0];
	const certBagList = certBags[forge.pki.oids.certBag];
	if (!keyBag?.key || !certBagList?.length) {
		throw new Error("P12 file is missing a private key or certificate");
	}
	const privateKey = keyBag.key as forge.pki.rsa.PrivateKey;

	const friendlyName =
		certBagList[1]?.attributes?.friendlyName?.[0] ??
		certBagList[0]?.cert?.issuer.attributes[2]?.value ??
		"";

	// The "main" certificate is the one with the most extensions (matches
	// osodreamer's heuristic — P12 chains commonly include intermediate CA
	// certs alongside the leaf cert; the leaf has the most extensions).
	const mainCertBag = certBagList.reduce((prev, current) =>
		(current.cert?.extensions.length ?? 0) > (prev.cert?.extensions.length ?? 0) ? current : prev,
	);
	const certificate = mainCertBag.cert;
	if (!certificate) {
		throw new Error("P12 file's certificate bag has no certificate");
	}

	const certificateX509Der = forge.asn1.toDer(forge.pki.certificateToAsn1(certificate));
	const certificateX509 = forge.util.encode64(certificateX509Der.bytes());
	const certDigestSha1Base64 = forge.util.encode64(
		forge.md.sha1.create().update(certificateX509Der.bytes()).digest().bytes(),
	);
	const serialNumber = new forge.jsbn.BigInteger(
		Array.from(Buffer.from(certificate.serialNumber, "hex")),
	).toString();

	return {
		privateKeyPem: forge.pki.privateKeyToPem(privateKey),
		certificateX509,
		certDigestSha1Base64,
		issuerName: resolveIssuerName(certificate, friendlyName),
		serialNumber,
		modulus: bigIntegerToBase64(privateKey.n),
		exponent: bigIntegerToBase64(privateKey.e),
	};
}
```

**Resolving the design's "Open questions":** all 4 CA strategies found during investigation (ANFAC, Banco Central, Security Data, Uanataca) are ported above, per the design's recommendation. `resolveIssuerName`'s fallback (a 5th, catch-all path) only engages for a certificate from a CA outside these 4 known ones — signing still proceeds in that case (same shortName-join format as Banco Central/Security Data), rather than hard-failing.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/sri-xml-signer && npx vitest run src/services/xades-signer/certificate.test.ts`
Expected: PASS, 2/2.

- [ ] **Step 5: Commit**

```bash
git add apps/sri-xml-signer/src/services/xades-signer/certificate.ts \
  apps/sri-xml-signer/src/services/xades-signer/certificate.test.ts
git commit -m "feat(sri-xml-signer): port P12 certificate extraction"
```

---

## Task 3: `builders.ts` + `sign.ts` — XAdES assembly and the signing orchestrator

**Files:**
- Create: `apps/sri-xml-signer/src/services/xades-signer/builders.ts`
- Create: `apps/sri-xml-signer/src/services/xades-signer/sign.ts`
- Test: `apps/sri-xml-signer/src/services/xades-signer/sign.test.ts`

**Interfaces:**
- Consumes: `canonicalizeFragment` (Task 1), `extractCertificateData`/`CertificateData` (Task 2).
- Produces: `signXades(xmlToSign: string, p12Buffer: Buffer, password: string): string` — the only export `signer.service.ts` (Task 4) calls.

- [ ] **Step 1: Implement `builders.ts`**

Ported from `osodreamer`'s `KeyInfoBuilder`/`SignedInfoBuilder`/`SignedPropertiesBuilder`/`XadesDocumentAssembler` (already schema-valid — SRI's `/validate` has always accepted this structure), with every Reference's canonicalization declared as Exclusive C14N consistently (the systemic gap this plan fixes) and IDs generated the same way (random 6-7 digit numbers as strings — `osodreamer`'s `SignatureIdGeneratorImplement`).

```ts
// apps/sri-xml-signer/src/services/xades-signer/builders.ts
const DS = "http://www.w3.org/2000/09/xmldsig#";
const ETSI = "http://uri.etsi.org/01903/v1.3.2#";
const XMLNS_ATTRS = `xmlns:ds="${DS}" xmlns:etsi="${ETSI}"`;
const SHA1 = "http://www.w3.org/2000/09/xmldsig#sha1";
const RSA_SHA1 = "http://www.w3.org/2000/09/xmldsig#rsa-sha1";
const EXC_C14N = "http://www.w3.org/2001/10/xml-exc-c14n#";
const ENVELOPED_SIGNATURE = "http://www.w3.org/2000/09/xmldsig#enveloped-signature";
const SIGNED_PROPERTIES_TYPE = "http://uri.etsi.org/01903#SignedProperties";

export interface SignatureIds {
	certificateNumber: string;
	signatureNumber: string;
	signedPropertiesNumber: string;
	signedInfoNumber: string;
	signedPropertiesIdNumber: string;
	referenceIdNumber: string;
	signatureValueNumber: string;
	objectNumber: string;
}

function randomId(): string {
	// Matches osodreamer's SignatureIdGeneratorImplement range (100000-9999999).
	return String(100000 + Math.floor(Math.random() * 9900000));
}

export function generateSignatureIds(): SignatureIds {
	return {
		certificateNumber: randomId(),
		signatureNumber: randomId(),
		signedPropertiesNumber: randomId(),
		signedInfoNumber: randomId(),
		signedPropertiesIdNumber: randomId(),
		referenceIdNumber: randomId(),
		signatureValueNumber: randomId(),
		objectNumber: randomId(),
	};
}

// Ecuador has no DST; SRI expects local (UTC-5) time in SigningTime.
export function nowInEcuadorIso(): string {
	const nowUtcMs = Date.now();
	const ecuadorMs = nowUtcMs - 5 * 60 * 60 * 1000;
	const d = new Date(ecuadorMs);
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}-05:00`;
}

export function buildKeyInfo(params: {
	certificateNumber: string;
	certificateX509: string;
	modulus: string;
	exponent: string;
}): string {
	const { certificateNumber, certificateX509, modulus, exponent } = params;
	return (
		`<ds:KeyInfo ${XMLNS_ATTRS} Id="Certificate${certificateNumber}">` +
		"<ds:X509Data>" +
		`<ds:X509Certificate>${certificateX509}</ds:X509Certificate>` +
		"</ds:X509Data>" +
		"<ds:KeyValue><ds:RSAKeyValue>" +
		`<ds:Modulus>${modulus}</ds:Modulus>` +
		`<ds:Exponent>${exponent}</ds:Exponent>` +
		"</ds:RSAKeyValue></ds:KeyValue>" +
		"</ds:KeyInfo>"
	);
}

export function buildSignedProperties(params: {
	signatureNumber: string;
	signedPropertiesNumber: string;
	certDigestSha1Base64: string;
	issuerName: string;
	serialNumber: string;
	referenceIdNumber: string;
	signingTimeIso: string;
}): string {
	const {
		signatureNumber,
		signedPropertiesNumber,
		certDigestSha1Base64,
		issuerName,
		serialNumber,
		referenceIdNumber,
		signingTimeIso,
	} = params;
	return (
		`<etsi:SignedProperties ${XMLNS_ATTRS} Id="Signature${signatureNumber}-SignedProperties${signedPropertiesNumber}">` +
		"<etsi:SignedSignatureProperties>" +
		`<etsi:SigningTime>${signingTimeIso}</etsi:SigningTime>` +
		"<etsi:SigningCertificate><etsi:Cert><etsi:CertDigest>" +
		`<ds:DigestMethod Algorithm="${SHA1}"/>` +
		`<ds:DigestValue>${certDigestSha1Base64}</ds:DigestValue>` +
		"</etsi:CertDigest><etsi:IssuerSerial>" +
		`<ds:X509IssuerName>${issuerName}</ds:X509IssuerName>` +
		`<ds:X509SerialNumber>${serialNumber}</ds:X509SerialNumber>` +
		"</etsi:IssuerSerial></etsi:Cert></etsi:SigningCertificate>" +
		"</etsi:SignedSignatureProperties>" +
		"<etsi:SignedDataObjectProperties>" +
		`<etsi:DataObjectFormat ObjectReference="#Reference-ID-${referenceIdNumber}">` +
		"<etsi:Description>contenido comprobante</etsi:Description>" +
		"<etsi:MimeType>text/xml</etsi:MimeType>" +
		"</etsi:DataObjectFormat>" +
		"</etsi:SignedDataObjectProperties>" +
		"</etsi:SignedProperties>"
	);
}

export function buildSignedInfo(params: {
	ids: SignatureIds;
	sha1SignedProperties: string;
	sha1Certificate: string;
	sha1Comprobante: string;
}): string {
	const { ids, sha1SignedProperties, sha1Certificate, sha1Comprobante } = params;
	return (
		`<ds:SignedInfo ${XMLNS_ATTRS} Id="Signature-SignedInfo${ids.signedInfoNumber}">` +
		`<ds:CanonicalizationMethod Algorithm="${EXC_C14N}"/>` +
		`<ds:SignatureMethod Algorithm="${RSA_SHA1}"/>` +
		`<ds:Reference Id="SignedPropertiesID${ids.signedPropertiesIdNumber}" Type="${SIGNED_PROPERTIES_TYPE}" URI="#Signature${ids.signatureNumber}-SignedProperties${ids.signedPropertiesNumber}">` +
		`<ds:Transforms><ds:Transform Algorithm="${EXC_C14N}"/></ds:Transforms>` +
		`<ds:DigestMethod Algorithm="${SHA1}"/>` +
		`<ds:DigestValue>${sha1SignedProperties}</ds:DigestValue>` +
		"</ds:Reference>" +
		`<ds:Reference URI="#Certificate${ids.certificateNumber}">` +
		`<ds:Transforms><ds:Transform Algorithm="${EXC_C14N}"/></ds:Transforms>` +
		`<ds:DigestMethod Algorithm="${SHA1}"/>` +
		`<ds:DigestValue>${sha1Certificate}</ds:DigestValue>` +
		"</ds:Reference>" +
		`<ds:Reference Id="Reference-ID-${ids.referenceIdNumber}" URI="#comprobante">` +
		`<ds:Transforms><ds:Transform Algorithm="${ENVELOPED_SIGNATURE}"/><ds:Transform Algorithm="${EXC_C14N}"/></ds:Transforms>` +
		`<ds:DigestMethod Algorithm="${SHA1}"/>` +
		`<ds:DigestValue>${sha1Comprobante}</ds:DigestValue>` +
		"</ds:Reference>" +
		"</ds:SignedInfo>"
	);
}

export function assembleSignatureElement(params: {
	ids: SignatureIds;
	signedInfo: string;
	signatureValueBase64: string;
	keyInfo: string;
	signedProperties: string;
}): string {
	const { ids, signedInfo, signatureValueBase64, keyInfo, signedProperties } = params;
	return (
		`<ds:Signature xmlns:ds="${DS}" xmlns:etsi="${ETSI}" Id="Signature${ids.signatureNumber}">` +
		signedInfo +
		`<ds:SignatureValue Id="SignatureValue${ids.signatureValueNumber}">${signatureValueBase64}</ds:SignatureValue>` +
		keyInfo +
		`<ds:Object Id="Signature${ids.signatureNumber}-Object${ids.objectNumber}">` +
		`<etsi:QualifyingProperties Target="#Signature${ids.signatureNumber}">${signedProperties}</etsi:QualifyingProperties>` +
		"</ds:Object>" +
		"</ds:Signature>"
	);
}
```

- [ ] **Step 2: Implement `sign.ts`**

```ts
// apps/sri-xml-signer/src/services/xades-signer/sign.ts
import { createHash } from "node:crypto";
import { DOMParser, XMLSerializer } from "@xmldom/xmldom";
import { SignedXml } from "xml-crypto";
import {
	assembleSignatureElement,
	buildKeyInfo,
	buildSignedInfo,
	buildSignedProperties,
	generateSignatureIds,
	nowInEcuadorIso,
} from "./builders";
import { canonicalizeFragment } from "./canonicalize";
import { extractCertificateData } from "./certificate";

function sha1Base64(canonicalXml: string): string {
	return createHash("sha1").update(canonicalXml, "utf8").digest("base64");
}

function rsaSha1Sign(canonicalSignedInfo: string, privateKeyPem: string): string {
	const signedXml = new SignedXml();
	const RsaSha1 = signedXml.SignatureAlgorithms["http://www.w3.org/2000/09/xmldsig#rsa-sha1"];
	const algorithm = new RsaSha1();
	return algorithm.getSignature(canonicalSignedInfo, privateKeyPem) as string;
}

/**
 * Sign an SRI comprobante XML (factura/notaCredito/notaDebito) with XAdES-BES,
 * using xml-crypto's canonicalization and RSA-SHA1 primitives for every
 * digest and the final signature — see design doc for why: osodreamer's
 * hand-rolled canonicalizer produced digests xmlsec1 rejects.
 */
export function signXades(xmlToSign: string, p12Buffer: Buffer, password: string): string {
	const cert = extractCertificateData(p12Buffer, password);
	const ids = generateSignatureIds();

	const sha1Comprobante = sha1Base64(canonicalizeFragment(xmlToSign));

	const signedProperties = buildSignedProperties({
		signatureNumber: ids.signatureNumber,
		signedPropertiesNumber: ids.signedPropertiesNumber,
		certDigestSha1Base64: cert.certDigestSha1Base64,
		issuerName: cert.issuerName,
		serialNumber: cert.serialNumber,
		referenceIdNumber: ids.referenceIdNumber,
		signingTimeIso: nowInEcuadorIso(),
	});
	const sha1SignedProperties = sha1Base64(canonicalizeFragment(signedProperties));

	const keyInfo = buildKeyInfo({
		certificateNumber: ids.certificateNumber,
		certificateX509: cert.certificateX509,
		modulus: cert.modulus,
		exponent: cert.exponent,
	});
	const sha1Certificate = sha1Base64(canonicalizeFragment(keyInfo));

	const signedInfo = buildSignedInfo({
		ids,
		sha1SignedProperties,
		sha1Certificate,
		sha1Comprobante,
	});
	const canonicalSignedInfo = canonicalizeFragment(signedInfo);
	const signatureValueBase64 = rsaSha1Sign(canonicalSignedInfo, cert.privateKeyPem);

	const signatureElementXml = assembleSignatureElement({
		ids,
		signedInfo,
		signatureValueBase64,
		keyInfo,
		signedProperties,
	});

	const originalDoc = new DOMParser().parseFromString(xmlToSign, "text/xml");
	const signatureNode = new DOMParser().parseFromString(signatureElementXml, "text/xml")
		.documentElement;
	if (!originalDoc.documentElement || !signatureNode) {
		throw new Error("signXades: failed to parse XML during signature assembly");
	}
	const imported = originalDoc.importNode(signatureNode, true);
	originalDoc.documentElement.appendChild(imported);
	return new XMLSerializer().serializeToString(originalDoc);
}
```

- [ ] **Step 3: Write the integration test (independent verification with `xmlsec1`)**

```ts
// apps/sri-xml-signer/src/services/xades-signer/sign.test.ts
import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import forge from "node-forge";
import { beforeAll, describe, expect, it } from "vitest";
import { signXades } from "./sign";

function xmlsec1Available(): boolean {
	try {
		execFileSync("xmlsec1", ["--version"], { stdio: "ignore" });
		return true;
	} catch {
		return false;
	}
}

function buildTestP12(): { buffer: Buffer; password: string } {
	const keys = forge.pki.rsa.generateKeyPair(2048);
	const cert = forge.pki.createCertificate();
	cert.publicKey = keys.publicKey;
	cert.serialNumber = "01";
	cert.validity.notBefore = new Date();
	cert.validity.notAfter = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000);
	const attrs = [
		{ shortName: "C", value: "EC" },
		{ shortName: "O", value: "TEST" },
		{ shortName: "CN", value: "SECURITY DATA TEST CA" },
	];
	cert.setSubject(attrs);
	cert.setIssuer(attrs);
	cert.sign(keys.privateKey, forge.md.sha1.create());
	const password = "test-password";
	const p12Asn1 = forge.pkcs12.toPkcs12Asn1(keys.privateKey, [cert], password, {
		algorithm: "3des",
	});
	const p12Der = forge.asn1.toDer(p12Asn1).getBytes();
	return { buffer: Buffer.from(p12Der, "binary"), password };
}

const FIXTURE_XML =
	'<factura id="comprobante" version="1.1.0" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">' +
	"<infoTributaria><ambiente>1</ambiente><razonSocial>Test</razonSocial><ruc>0999999999001</ruc></infoTributaria>" +
	"<infoFactura><totalSinImpuestos>10.00</totalSinImpuestos></infoFactura>" +
	"</factura>";

describe("signXades", () => {
	let xmlsecOk = false;
	beforeAll(() => {
		xmlsecOk = xmlsec1Available();
		if (!xmlsecOk) {
			console.warn(
				"xmlsec1 not found on PATH — skipping independent signature verification. " +
					"Install it (e.g. `apk add xmlsec` / `apt install xmlsec1`) to run this check.",
			);
		}
	});

	it("produces XML that xmlsec1 independently verifies as a valid signature", () => {
		if (!xmlsecOk) return;
		const { buffer, password } = buildTestP12();
		const signed = signXades(FIXTURE_XML, buffer, password);

		const dir = mkdtempSync(join(tmpdir(), "xades-test-"));
		const file = join(dir, "signed.xml");
		writeFileSync(file, signed, "utf8");

		expect(() =>
			execFileSync("xmlsec1", [
				"verify",
				"--id-attr:id",
				"factura",
				"--enabled-key-data",
				"x509",
				"--insecure",
				file,
			]),
		).not.toThrow();
	});
});
```

- [ ] **Step 4: Run the integration test**

Run: `cd apps/sri-xml-signer && npx vitest run src/services/xades-signer/sign.test.ts`
Expected: PASS. If `xmlsec1` isn't installed in this environment, it prints the warning and the assertion body is skipped (test still passes) — install `xmlsec1` locally (`apk add xmlsec` inside the `sri-xml-signer` dev container, or `brew install libxmlsec1` on macOS) to actually exercise the check before trusting this task's result.

- [ ] **Step 5: Run the full test suite**

Run: `cd apps/sri-xml-signer && npx vitest run`
Expected: PASS, all tests from Tasks 1-3 green.

- [ ] **Step 6: Commit**

```bash
git add apps/sri-xml-signer/src/services/xades-signer/builders.ts \
  apps/sri-xml-signer/src/services/xades-signer/sign.ts \
  apps/sri-xml-signer/src/services/xades-signer/sign.test.ts
git commit -m "feat(sri-xml-signer): XAdES-BES orchestrator, canonicalized at every digest"
```

---

## Task 4: Wire into `signer.service.ts` and verify end-to-end

**Files:**
- Modify: `apps/sri-xml-signer/src/services/signer.service.ts:1-16` (current full contents — see below)

**Interfaces:**
- Consumes: `signXades` (Task 3).

- [ ] **Step 1: Update `signer.service.ts`**

Current contents (from investigation):

```ts
import { authorizeXml, type ComprobanteModel, generateXmlInvoice, type SRIEnv, signXml, validateXml } from "osodreamer-sri-xml-signer";

export const signerService = {
	generate: (payload: ComprobanteModel) => generateXmlInvoice(payload),
	sign: ({ p12Buffer, xmlBuffer, password }: SignParams) => signXml({ p12Buffer, xmlBuffer, password }),
	validate: ({ xmlBuffer, env }: ValidateParams) => validateXml({ xml: xmlBuffer, env }),
	authorize: ({ accessKey, env }: AuthorizeParams) => authorizeXml({ claveAcceso: accessKey, env }),
};
```

Replace only the `sign` line and its import — `generate`/`validate`/`authorize` and their type imports stay exactly as-is:

```ts
import { authorizeXml, type ComprobanteModel, generateXmlInvoice, type SRIEnv, validateXml } from "osodreamer-sri-xml-signer";
import { signXades } from "./xades-signer/sign";

export const signerService = {
	generate: (payload: ComprobanteModel) => generateXmlInvoice(payload),
	sign: ({ p12Buffer, xmlBuffer, password }: SignParams) =>
		Promise.resolve({ xml: signXades(xmlBuffer.toString("utf8"), Buffer.from(p12Buffer), password) }),
	validate: ({ xmlBuffer, env }: ValidateParams) => validateXml({ xml: xmlBuffer, env }),
	authorize: ({ accessKey, env }: AuthorizeParams) => authorizeXml({ claveAcceso: accessKey, env }),
};
```

`signXades` is synchronous; wrapping in `Promise.resolve(...)` keeps `sign`'s return type identical to today's (`osodreamer`'s `signXml` returns `Promise<string>`; the handler at `signer.handler.ts` already does `const signedXml = await signerService.sign(...)` and reads `.xml` off an object — confirm the exact current shape by reading `signer.handler.ts`'s `/sign` handler at implementation time, and match it exactly; if it currently does `const { xml: signedXml } = await signerService.sign(...)`, the shape above is already correct).

`p12Buffer`/`xmlBuffer` arrive as `Uint8Array` (per `toBuffer()` in `utils/buffer.ts`) — `Buffer.from(p12Buffer)` and `.toString("utf8")` normalize them for `signXades`'s `Buffer`/`string` parameters.

- [ ] **Step 2: Verify the full build**

Run: `cd apps/sri-xml-signer && pnpm run build`
Expected: `tsc` succeeds with no type errors.

- [ ] **Step 3: Manual end-to-end verification against the dev stack**

This reproduces the exact repro path used during the original investigation:

```bash
# From repo root, with the dev stack running (docker compose -f docker-compose.dev.yaml up -d)
docker compose -f docker-compose.dev.yaml restart sri-xml-signer
sleep 6
curl -s -u guest:guest -X POST http://localhost:15672/api/exchanges/%2f/amq.default/publish \
  -H "Content-Type: application/json" \
  -d '{"properties":{},"routing_key":"invoice_tasks","payload":"{\"invoice_id\":17}","payload_encoding":"string"}'
sleep 4
docker logs pengi-api --tail 10
docker logs sri-xml-signer --tail 10
```

Expected: no `ERROR ... Failed to run SRI pipeline` line; the invoice's status in the `invoices` table moves past `failed`/`FIRMA INVALIDA` (check with `docker exec pengi-db-dev psql -U postgres -d pengi_gentoo -c "select id, status, error_message from invoices where id = 17;"`).

Also independently verify the newly-generated signed XML on disk (`storage/tenants/47/invoices/<newest>.xml`) with `xmlsec1 verify --id-attr:id factura --enabled-key-data x509 --insecure <file>` → `Verification status: OK`.

- [ ] **Step 4: Commit**

```bash
git add apps/sri-xml-signer/src/services/signer.service.ts
git commit -m "fix(sri-xml-signer): sign via the new xades-signer module instead of osodreamer"
```

---

## Verification (whole plan)

1. `cd apps/sri-xml-signer && npx vitest run` — all unit + integration tests green.
2. `cd apps/sri-xml-signer && pnpm run build` — clean TypeScript build.
3. Manual dev-stack invoice reprocessing (Task 4 Step 3) — no `FIRMA INVALIDA`, and `xmlsec1 verify` on the resulting signed XML reports `Verification status: OK`.
4. Spot-check `/generate`, `/validate/:env`, `/authorization/:env` are untouched: `git diff` should show zero changes to `signer.handler.ts`, `routes/index.ts`, and to the `generate`/`validate`/`authorize` lines of `signer.service.ts`.
