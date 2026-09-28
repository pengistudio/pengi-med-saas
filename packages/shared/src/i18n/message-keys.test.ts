// @vitest-environment node
import { readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

/**
 * Every literal message key the frontends use must exist in both API message
 * files — otherwise the UI renders `*key*`.
 *
 * A key read with a `count` (`textGet("a.b", { count })`) may exist as its
 * plural forms instead: `a.b.one` and `a.b.other`.
 *
 * Only literal keys are checked: `textGet("a.b")`, `textGet('a.b')`,
 * `uuid="a.b"` and `uuid={"a.b"}`. Keys built at runtime (template literals,
 * variables, concatenation) can't be read statically and are skipped; the
 * test counts them (about a hundred, e.g. `textGet(\`status.${s}\`)`,
 * `uuid={item.labelKey}`) so a big jump is visible.
 */

const ROOT = fileURLToPath(new URL("../../../../", import.meta.url));
const SOURCE_DIRS = [
	"apps/web/src",
	"apps/backoffice/src",
	"packages/ui/src",
	"packages/shared/src",
];
const MESSAGE_FILES = [
	"apps/api/i18n/messages/messages_es.json",
	"apps/api/i18n/messages/messages_en.json",
];

const LITERAL_TEXT_GET = /\btextGet\(\s*(["'])([^"'`\n]+)\1\s*[,)]/g;
const ANY_TEXT_GET = /\btextGet\(/g;
const LITERAL_UUID = /\buuid=\{?\s*(["'])([^"'`\n]+)\1\s*\}?/g;
const ANY_UUID = /\buuid=/g;

function sourceFiles(dir: string): string[] {
	return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
		const path = join(dir, entry.name);
		if (entry.isDirectory()) {
			return ["node_modules", "dist", "__tests__", "test"].includes(entry.name)
				? []
				: sourceFiles(path);
		}
		return /\.tsx?$/.test(entry.name) &&
			!/\.(test|spec)\.tsx?$/.test(entry.name)
			? [path]
			: [];
	});
}

/** Accepts the `[{key, value}]` file format and a flat `{key: value}` one. */
function loadKeys(file: string): Set<string> {
	const data: unknown = JSON.parse(readFileSync(join(ROOT, file), "utf8"));
	if (Array.isArray(data)) {
		return new Set(data.map((m: { key: string }) => m.key));
	}
	return new Set(Object.keys(data as Record<string, unknown>));
}

function collectKeys() {
	const used = new Map<string, string>(); // key → first file using it
	let dynamic = 0;
	for (const dir of SOURCE_DIRS) {
		for (const file of sourceFiles(join(ROOT, dir))) {
			const source = readFileSync(file, "utf8");
			let literal = 0;
			for (const re of [LITERAL_TEXT_GET, LITERAL_UUID]) {
				for (const match of source.matchAll(re)) {
					literal++;
					if (!used.has(match[2])) used.set(match[2], relative(ROOT, file));
				}
			}
			const all =
				(source.match(ANY_TEXT_GET)?.length ?? 0) +
				(source.match(ANY_UUID)?.length ?? 0);
			dynamic += all - literal;
		}
	}
	return { used, dynamic };
}

describe("message keys used by the frontends", () => {
	const { used, dynamic } = collectKeys();

	it("finds the literal keys", () => {
		expect(used.size).toBeGreaterThan(500);
		// Sanity bound on the keys this test can't check.
		expect(dynamic).toBeLessThan(200);
	});

	it.each(MESSAGE_FILES)("all exist in %s", (file) => {
		const keys = loadKeys(file);
		const exists = (key: string) =>
			keys.has(key) || (keys.has(`${key}.one`) && keys.has(`${key}.other`));
		const missing = [...used]
			.filter(([key]) => !exists(key))
			.map(([key, where]) => `${key}  (${where})`);
		expect(missing).toEqual([]);
	});
});
