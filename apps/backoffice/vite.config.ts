import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

// The repo root .env is the one env file for every app (see .env.example).
const envDir = path.resolve(__dirname, "../..");
// A production build can't fall back to the dev defaults in src/lib/env.ts.
const requiredInBuild = ["VITE_API_URL", "VITE_APP_URL"];

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
	if (command === "build") {
		const env = loadEnv(mode, envDir, "VITE_");
		const missing = requiredInBuild.filter((name) => !env[name]);
		if (missing.length > 0) {
			throw new Error(
				`Missing ${missing.join(", ")} for the build (see .env.example)`,
			);
		}
	}

	return {
		envDir,
		plugins: [react(), tailwindcss()],
		resolve: {
			alias: {
				"@": path.resolve(__dirname, "./src"),
			},
		},
	};
});
