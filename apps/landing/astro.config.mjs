// @ts-check

import { fileURLToPath } from "node:url";
import react from "@astrojs/react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig, envField } from "astro/config";

// A production build needs every variable; the defaults apply only to
// `astro dev` (they match the dev stack in docker-compose.dev.yaml).
const isBuild = process.argv.includes("build");
const publicUrl = (/** @type {string} */ devDefault) =>
	envField.string({
		context: "client",
		access: "public",
		...(isBuild ? {} : { default: devDefault }),
	});

// https://astro.build/config
export default defineConfig({
	vite: {
		plugins: [tailwindcss()],
		// The repo root .env is the one env file for every app (see .env.example).
		envDir: fileURLToPath(new URL("../..", import.meta.url)),
	},

	// Every variable the landing reads: import them from "astro:env/client".
	env: {
		schema: {
			PUBLIC_API_URL: publicUrl("http://localhost:8000/api/v1"),
			// The web app, for the "Iniciar sesión" button.
			PUBLIC_APP_URL: publicUrl("http://localhost:3000"),
		},
	},

	integrations: [react()],

	// Astro 7 defaults to "jsx" whitespace handling; keep the HTML as before.
	compressHTML: true,
});
