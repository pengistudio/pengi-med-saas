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
