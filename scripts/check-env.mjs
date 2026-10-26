#!/usr/bin/env node
// Checks that the environment variables the apps read and the ones the env
// templates document are the same set, and that no code reads variables
// outside the config modules. Run: node scripts/check-env.mjs (CI runs it).
//
// Readers (the only places a variable may be declared):
//   API         apps/api/core/config/config.go   `env:"NAME"` tags
//   web         apps/web/src/lib/env.ts          zod schema keys
//   backoffice  apps/backoffice/src/lib/env.ts   zod schema keys
//   portal      apps/portal/src/lib/env.ts       zod schema keys
//   landing     apps/landing/astro.config.mjs    env.schema keys
//   compose     docker-compose.dev.yaml          ${NAME} interpolation
// Templates:
//   .env.example              every variable above (API, frontends, compose)
//   deploy/.env.prod.example  every API variable
// A template documents a variable with `NAME=` or a commented `# NAME=`.

import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";

const root = join(import.meta.dirname, "..");
const read = (path) => readFileSync(join(root, path), "utf8");
const errors = [];

// ─── Readers ────────────────────────────────────────────────────────────────

// API: `FieldName  type  `env:"NAME,..."``. Fields named Legacy* hold old
// names kept as fallbacks: read, but not documented.
const api = new Set();
const apiLegacy = new Set();
for (const [, field, name] of read("apps/api/core/config/config.go").matchAll(
	/^\s*(\w+)\s+[\w.[\]*]+\s+`[^`]*env:"([A-Z0-9_]+)/gm,
)) {
	(field.startsWith("Legacy") ? apiLegacy : api).add(name);
}

const keysOf = (path, pattern) =>
	new Set([...read(path).matchAll(pattern)].map((m) => m[1]));
const web = keysOf("apps/web/src/lib/env.ts", /^\s*(VITE_[A-Z0-9_]+):/gm);
const backoffice = keysOf(
	"apps/backoffice/src/lib/env.ts",
	/^\s*(VITE_[A-Z0-9_]+):/gm,
);
const portal = keysOf("apps/portal/src/lib/env.ts", /^\s*(VITE_[A-Z0-9_]+):/gm);
const landing = keysOf(
	"apps/landing/astro.config.mjs",
	/^\s*(PUBLIC_[A-Z0-9_]+):/gm,
);
// Compose interpolation, minus variables the api already reads.
const compose = new Set(
	[...keysOf("docker-compose.dev.yaml", /\$\{([A-Z0-9_]+)/g)].filter(
		(name) => !api.has(name),
	),
);
// COMPOSE_PROFILES is read by docker compose itself.
compose.add("COMPOSE_PROFILES");

for (const [label, set] of Object.entries({
	api,
	web,
	backoffice,
	portal,
	landing,
})) {
	if (set.size === 0)
		errors.push(`found no variables for ${label}: did its file move?`);
}

// ─── Templates ──────────────────────────────────────────────────────────────

const documented = (path) => keysOf(path, /^(?:#\s?)?([A-Z][A-Z0-9_]*)=/gm);

function compare(template, expected, allowedExtra = new Set()) {
	const have = documented(template);
	for (const [name, reader] of expected) {
		if (!have.has(name))
			errors.push(`${template}: missing ${name} (read by ${reader})`);
	}
	for (const name of have) {
		if (!expected.has(name) && !allowedExtra.has(name)) {
			errors.push(
				`${template}: ${name} is not read by any app (remove it or add it to its reader)`,
			);
		}
	}
}

const tag = (set, reader) => [...set].map((name) => [name, reader]);
const merged = (...pairs) => {
	const map = new Map();
	for (const [name, reader] of pairs.flat()) {
		map.set(name, map.has(name) ? `${map.get(name)}, ${reader}` : reader);
	}
	return map;
};

compare(
	".env.example",
	merged(
		tag(api, "apps/api core/config"),
		tag(web, "apps/web src/lib/env.ts"),
		tag(backoffice, "apps/backoffice src/lib/env.ts"),
		tag(portal, "apps/portal src/lib/env.ts"),
		tag(landing, "apps/landing astro.config.mjs"),
		tag(compose, "docker-compose.dev.yaml"),
	),
	apiLegacy,
);
compare(
	"deploy/.env.prod.example",
	merged(tag(api, "apps/api core/config")),
	apiLegacy,
);

// ─── Reads outside the config modules ───────────────────────────────────────

function* files(dir, extensions) {
	for (const entry of readdirSync(join(root, dir), { withFileTypes: true })) {
		if (
			["node_modules", "dist", ".astro", "tmp", "storage"].includes(entry.name)
		)
			continue;
		const path = join(dir, entry.name);
		if (entry.isDirectory()) yield* files(path, extensions);
		else if (extensions.some((ext) => entry.name.endsWith(ext))) yield path;
	}
}

// Go: only core/config may call os.Getenv. Exceptions: core/secretbox (keys
// by name), committed code migrations (immutable), tests and test helpers.
const goAllowed = [
	/^apps\/api\/core\/config\//,
	/^apps\/api\/core\/secretbox\//,
	/^apps\/api\/migrations\/code-migrations\//,
	/^apps\/api\/testutils\//,
	/_test\.go$/,
];
for (const path of files("apps/api", [".go"])) {
	if (goAllowed.some((re) => re.test(path))) continue;
	read(path)
		.split("\n")
		.forEach((line, i) => {
			if (/\bos\.(Getenv|LookupEnv)\(/.test(line)) {
				errors.push(
					`${path}:${i + 1}: reads os.Getenv; add a field to core/config and use config.Get()`,
				);
			}
		});
}

// Frontends: import.meta.env.<NAME> only in env.ts; Vite's built-in flags
// are fine anywhere.
const builtIn = new Set(["DEV", "PROD", "MODE", "SSR", "BASE_URL"]);
for (const [dir, allowed] of [
	["apps/web/src", "apps/web/src/lib/env.ts"],
	["apps/backoffice/src", "apps/backoffice/src/lib/env.ts"],
	["apps/portal/src", "apps/portal/src/lib/env.ts"],
	["apps/landing/src", null],
]) {
	for (const path of files(dir, [".ts", ".tsx", ".astro"])) {
		if (path === allowed || path.endsWith(".d.ts")) continue;
		read(path)
			.split("\n")
			.forEach((line, i) => {
				for (const [, name] of line.matchAll(
					/import\.meta\.env\.([A-Z0-9_]+)/g,
				)) {
					if (!builtIn.has(name)) {
						errors.push(
							`${path}:${i + 1}: reads import.meta.env.${name}; read it from ${allowed ?? '"astro:env/client"'}`,
						);
					}
				}
			});
	}
}

if (errors.length > 0) {
	console.error(`check-env: ${errors.length} problem(s)\n`);
	for (const error of errors) console.error(`  ${error}`);
	process.exit(1);
}
console.log(
	`check-env: ok (${api.size} api, ${web.size} web, ${backoffice.size} backoffice, ${portal.size} portal, ${landing.size} landing, ${compose.size} compose)`,
);
