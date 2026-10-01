// @vitest-environment node
import { readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

/**
 * Text interpolation and date/amount formatting go through `useText`, which
 * follows the interface language (see CONTEXT.md, "Idioma de la interfaz").
 * Formatting inline pins a locale (or the browser's) and ignores the language
 * the user picked, so it is not allowed outside the module.
 *
 * `toFixed(2)` is not checked: it is also used in calculations, a regex can't
 * tell them apart.
 */

const ROOT = fileURLToPath(new URL("../../../../", import.meta.url));

const SOURCE_DIRS = [
	"apps/web/src",
	"apps/backoffice/src",
	"packages/shared/src",
];

const ALLOWED = new Set([
	// The module itself.
	"packages/shared/src/i18n/use-text.ts",
	// Subscription expiry dates live on Ecuador's calendar (time zone
	// America/Guayaquil), and en-CA is only used to produce YYYY-MM-DD.
	"apps/backoffice/src/lib/subscription/term.ts",
]);

const FORBIDDEN: { pattern: RegExp; use: string }[] = [
	{
		pattern: /\.toLocale(Date|Time)?String\(/g,
		use: "formatDate / formatDateTime / formatTime",
	},
	{
		pattern: /\bIntl\.(DateTimeFormat|NumberFormat|RelativeTimeFormat)\b/g,
		use: "formatDate / formatMoney / formatRelative",
	},
	{ pattern: /\.replace\(\s*["'`]\{/g, use: "textGet(key, { name: value })" },
	{ pattern: /\bdateParser\b/g, use: "formatDate" },
];

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

describe("formatting goes through useText", () => {
	it("no inline date, amount or interpolation formatting", () => {
		const violations: string[] = [];
		for (const dir of SOURCE_DIRS) {
			for (const file of sourceFiles(join(ROOT, dir))) {
				const path = relative(ROOT, file);
				if (ALLOWED.has(path)) continue;
				const source = readFileSync(file, "utf8");
				for (const { pattern, use } of FORBIDDEN) {
					for (const match of source.matchAll(pattern)) {
						const line = source.slice(0, match.index).split("\n").length;
						violations.push(`${path}:${line}  ${match[0]}  → ${use}`);
					}
				}
			}
		}
		expect(violations).toEqual([]);
	});
});
