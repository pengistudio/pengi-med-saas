import { z } from "zod";

// Every VITE_* variable the portal reads. Read them from `env`, never from
// import.meta.env. Values come from the repo root .env in dev (envDir in
// vite.config.ts) and from build args in the Docker image; the dev default
// below applies only to `vite dev` and tests, and a production build without
// the variable fails in vite.config.ts.
const url = (devDefault: string) =>
	z.preprocess(
		(value) =>
			value === undefined || value === ""
				? import.meta.env.DEV
					? devDefault
					: undefined
				: value,
		z.url(),
	);

const schema = z.object({
	// Includes /api/v1, like the other apps.
	VITE_API_URL: url("http://localhost:8000/api/v1"),
	// Cloudflare Turnstile site key: when set, signing in needs the captcha.
	VITE_TURNSTILE_SITE_KEY: z.preprocess(
		(value) => (value === "" ? undefined : value),
		z.string().optional(),
	),
});

export const env = schema.parse(import.meta.env);
