import * as z from "zod";
import { useMessageStore } from "./message-store";

const zodLocales = { es: z.locales.es, en: z.locales.en };

export type SupportedLocale = keyof typeof zodLocales;

/** A field left empty: missing, null, or "" where at least one character is required. */
const isEmptyField = (issue: z.core.$ZodRawIssue) =>
	(issue.code === "invalid_type" &&
		(issue.input === undefined || issue.input === null)) ||
	(issue.code === "too_small" &&
		issue.origin === "string" &&
		issue.minimum === 1 &&
		issue.input === "");

// Runs at validation time, so it reads the catalog of the current language.
// Any other issue (or a catalog not loaded yet) falls back to zod's locale.
const customError = (issue: z.core.$ZodRawIssue) =>
	isEmptyField(issue)
		? useMessageStore.getState().messages["form.validation.required"]
		: undefined;

export const updateZodLocale = (locale: SupportedLocale) => {
	z.config({ ...zodLocales[locale](), customError });
};
