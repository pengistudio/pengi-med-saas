import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import forge from "node-forge";
import { describe, expect, it } from "vitest";
import { signXades } from "./sign";

function xmlsec1Available(): boolean {
	try {
		execFileSync("xmlsec1", ["--version"], { stdio: "ignore" });
		return true;
	} catch {
		return false;
	}
}

// Computed once at module scope (not in beforeAll) so it.skipIf can read it
// synchronously at test-definition time.
const xmlsecOk = xmlsec1Available();
if (!xmlsecOk) {
	console.warn(
		"xmlsec1 not found on PATH — SKIPPING independent signature verification tests. " +
			"Install it (e.g. `apk add xmlsec` / `apt install xmlsec1`) to run this check.",
	);
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

// Single source of truth for the three SRI comprobante root tags — used to
// derive both the fixture XML's root element and the xmlsec1 --id-attr:id
// argument, so a new root tag can't drift out of sync between the two.
const SRI_ROOT_TAGS = ["factura", "notaCredito", "notaDebito"] as const;

function buildFixtureXml(rootTag: string): string {
	return (
		`<${rootTag} id="comprobante" version="1.1.0" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
		"<infoTributaria><ambiente>1</ambiente><razonSocial>Test</razonSocial><ruc>0999999999001</ruc></infoTributaria>" +
		"<infoFactura><totalSinImpuestos>10.00</totalSinImpuestos></infoFactura>" +
		`</${rootTag}>`
	);
}

// Writes `signed` to a fresh temp file and runs `xmlsec1 verify` against it,
// scoping the --id-attr:id to the given root tag (SRI's three comprobante
// root elements all use `id="comprobante"`, but xmlsec1 needs the tag name
// to resolve the ID attribute schema-lessly).
function verifyWithXmlsec1(signed: string, rootTag: string): void {
	const dir = mkdtempSync(join(tmpdir(), "xades-test-"));
	const file = join(dir, "signed.xml");
	writeFileSync(file, signed, "utf8");

	expect(() =>
		execFileSync("xmlsec1", [
			"verify",
			"--id-attr:id",
			rootTag,
			"--enabled-key-data",
			"x509",
			"--insecure",
			file,
		]),
	).not.toThrow();
}

describe.each(SRI_ROOT_TAGS)("signXades (%s)", (rootTag) => {
	it.skipIf(!xmlsecOk)(
		"produces XML that xmlsec1 independently verifies as a valid signature",
		() => {
			const { buffer, password } = buildTestP12();
			const signed = signXades(buildFixtureXml(rootTag), buffer, password);
			verifyWithXmlsec1(signed, rootTag);
		},
	);
});

describe("signXades preconditions", () => {
	it('throws a clear error when the root element is missing id="comprobante"', () => {
		const { buffer, password } = buildTestP12();
		expect(() => signXades("<factura><a>1</a></factura>", buffer, password)).toThrow(
			/id="comprobante"/,
		);
	});
});
