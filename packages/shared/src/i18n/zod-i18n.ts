import * as z from "zod";

export const zodLocales = {
	es: {
		...z.locales.es(),
		invalid_type: ({
			expected,
			received,
		}: {
			expected: unknown;
			received: unknown;
		}) => {
			if (expected === "string" && received === "undefined") {
				return "El campo no puede estar vacío";
			}
			return `Tipo inválido: se esperaba ${expected}, recibido ${received}`;
		},
		required: () => "El campo no puede estar vacío",
	},
	en: {
		...z.locales.en(),
		invalid_type: ({
			expected,
			received,
		}: {
			expected: unknown;
			received: unknown;
		}) => {
			if (expected === "string" && received === "undefined") {
				return "This field cannot be empty";
			}
			return `Invalid type: expected ${expected}, received ${received}`;
		},
		required: () => "This field is required",
	},
};

export type SupportedLocale = keyof typeof zodLocales;

export const updateZodLocale = (locale: SupportedLocale) => {
	z.config(zodLocales[locale]);
};
