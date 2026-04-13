import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
	plugins: [react(), tailwindcss()],
	define: {
		__APP_VERSION__: JSON.stringify(Date.now().toString()),
	},
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
	build: {
		rollupOptions: {
			output: {
				manualChunks(id) {
					if (
						id.includes("react-dom") ||
						id.includes("react-router") ||
						(id.includes("react") && !id.includes("react-hook-form"))
					) {
						return "vendor-react";
					}
					if (
						id.includes("@base-ui") ||
						id.includes("lucide-react") ||
						id.includes("recharts")
					) {
						return "vendor-ui";
					}
					if (
						id.includes("react-hook-form") ||
						id.includes("@hookform") ||
						id.includes("/zod/")
					) {
						return "vendor-forms";
					}
					if (id.includes("@dnd-kit")) {
						return "vendor-dnd";
					}
				},
			},
		},
	},
});
