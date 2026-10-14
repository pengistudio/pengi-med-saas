import { z } from "zod";

// Every VITE_* variable the backoffice reads. Read them from `env`, never from
// import.meta.env. Values come from the repo root .env in dev (envDir in
// vite.config.ts) and from build args in the Docker image; the dev defaults
// below apply only to `vite dev` and tests, and a production build without the
// variable fails in vite.config.ts.
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
	VITE_API_URL: url("http://localhost:8000/api/v1"),
	// The web app, for the signup links the backoffice hands out.
	VITE_APP_URL: url("http://localhost:3000"),
});

export const env = schema.parse(import.meta.env);
