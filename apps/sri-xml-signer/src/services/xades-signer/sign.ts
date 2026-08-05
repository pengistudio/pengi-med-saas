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
