import { DOMParser } from "@xmldom/xmldom";
import { SignedXml } from "xml-crypto";

const EXCLUSIVE_C14N = "http://www.w3.org/2001/10/xml-exc-c14n#";

const parser = new DOMParser({
	errorHandler: () => {
		// osodreamer's XmlDomContext silences parser warnings the same way.
		// NOTE: @xmldom/xmldom self-heals many malformed inputs (e.g. a
		// mismatched closing tag) instead of failing to parse, so the `!root`
		// guard below only catches clearly-non-XML input (empty string,
		// non-XML text) — it does NOT guarantee the input was well-formed.
		// Callers are still responsible for passing well-formed XML.
	},
});

/**
 * Canonicalize a single well-formed XML fragment (one root element) using
 * Exclusive C14N (no comments), via xml-crypto's own canonicalizer — never
 * a hand-rolled implementation. Every digest in sign.ts goes through this.
 */
export function canonicalizeFragment(xml: string): string {
	const doc = parser.parseFromString(xml, "text/xml");
	const root = doc?.documentElement;
	if (!root) {
		throw new Error("canonicalizeFragment: input is not well-formed XML");
	}
	const signedXml = new SignedXml();
	return signedXml.getCanonXml([EXCLUSIVE_C14N], root);
}
