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
