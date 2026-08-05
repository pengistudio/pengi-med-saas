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
