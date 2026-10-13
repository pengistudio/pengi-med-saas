/// <reference types="vite/client" />

// Read these through src/lib/env.ts, which validates them.
interface ImportMetaEnv {
	readonly VITE_API_URL?: string;
	readonly VITE_APP_URL?: string;
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}
