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
		// Cast to `any` here (rather than letting TS's control-flow analysis
		// narrow `p12` to forge's real PKCS12 return type) — that narrowed
		// type has an unreliable/inconsistent shape across @types/node-forge
		// versions and causes spurious implicit-any errors further down.
		p12 = forge.pkcs12.pkcs12FromAsn1(p12Asn1, password) as any;
	} catch (err) {
		// `es2016` (this project's tsconfig target) predates the ES2022
		// `Error(message, { cause })` constructor overload, so the cause is
		// attached as a plain property instead of via the constructor arg.
		const error = new Error("Invalid P12 file or password") as Error & { cause?: unknown };
		error.cause = err;
		throw error;
	}

	// biome-ignore lint/suspicious/noExplicitAny: forge's PKCS12 bag types are unreliable across @types/node-forge versions
	const keyBags: any = p12.getBags({ bagType: forge.pki.oids.pkcs8ShroudedKeyBag });
	// biome-ignore lint/suspicious/noExplicitAny: forge's PKCS12 bag types are unreliable across @types/node-forge versions
	const certBags: any = p12.getBags({ bagType: forge.pki.oids.certBag });
	const keyBag = keyBags[forge.pki.oids.pkcs8ShroudedKeyBag]?.[0];
	// biome-ignore lint/suspicious/noExplicitAny: forge's PKCS12 bag types are unreliable across @types/node-forge versions
	const certBagList: any[] = certBags[forge.pki.oids.certBag];
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
