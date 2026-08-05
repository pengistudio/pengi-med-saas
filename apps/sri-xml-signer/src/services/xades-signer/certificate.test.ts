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
