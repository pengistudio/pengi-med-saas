import { formatDistanceToNow } from "date-fns";
import { enUS, es } from "date-fns/locale";
import React from "react";
import { parseBareDate } from "./date-only";
import { useMessageStore } from "./message-store";
import type { SupportedLocale } from "./zod-i18n";

/** Values for a message's `{name}` placeholders; `count` also picks the plural form. */
export type TextValues = Record<string, string | number>;

/** What the formatters accept: a Date, an ISO string or a timestamp. */
export type DateInput = Date | string | number | null | undefined;

/**
 * `medium` "27 sept 2026", `long` "27 de septiembre de 2026", `full` with the
 * weekday, `day-month` "27 sept", `month-year` "sept 2026", `weekday` "dom"
 * (short weekday only), `weekday-narrow` "D", `weekday-day-month`
 * "domingo, 27 sept".
 */
export type DateStyle =
	| "medium"
	| "long"
	| "full"
	| "day-month"
	| "month-year"
	| "weekday"
	| "weekday-narrow"
	| "weekday-day-month";

// The interface language decides the format locale; the currency is always
// USD (Ecuador), only its formatting follows the language.
const FORMAT_LOCALES: Record<SupportedLocale, string> = {
	es: "es-EC",
	en: "en-US",
};

const DATE_FNS_LOCALES = { es, en: enUS } as const;

const DATE_STYLES: Record<DateStyle, Intl.DateTimeFormatOptions> = {
	medium: { dateStyle: "medium" },
	long: { dateStyle: "long" },
	full: { dateStyle: "full" },
	"day-month": { day: "numeric", month: "short" },
	"month-year": { month: "short", year: "numeric" },
	weekday: { weekday: "short" },
	"weekday-narrow": { weekday: "narrow" },
	"weekday-day-month": { weekday: "long", day: "numeric", month: "short" },
};

const PLACEHOLDER = /\{(\w+)\}/g;

/** Replaces `{name}` with `values.name`; placeholders without a value stay as they are. */
function interpolate(template: string, values: TextValues): string {
	return template.replace(PLACEHOLDER, (match, name: string) =>
		name in values ? String(values[name]) : match,
	);
}

function toDate(value: DateInput): Date | undefined {
	if (value === null || value === undefined || value === "") return undefined;
	// A bare "YYYY-MM-DD" is a calendar day, not UTC midnight.
	const date =
		value instanceof Date
			? value
			: (typeof value === "string" && parseBareDate(value)) || new Date(value);
	return Number.isNaN(date.getTime()) ? undefined : date;
}

function createText(lang: SupportedLocale, messages: Record<string, string>) {
	const locale = FORMAT_LOCALES[lang];
	const plurals = new Intl.PluralRules(locale);
	const dateFormats = Object.fromEntries(
		Object.entries(DATE_STYLES).map(([style, options]) => [
			style,
			new Intl.DateTimeFormat(locale, options),
		]),
	) as Record<DateStyle, Intl.DateTimeFormat>;
	const dateTimeFormat = new Intl.DateTimeFormat(locale, {
		dateStyle: "medium",
		timeStyle: "short",
	});
	const timeFormat = new Intl.DateTimeFormat(locale, { timeStyle: "medium" });
	const moneyFormat = new Intl.NumberFormat(locale, {
		style: "currency",
		currency: "USD",
	});

	/**
	 * The message for `key`, with `{name}` placeholders filled from `values`.
	 * With a numeric `values.count` it reads `key.one` / `key.other` (by the
	 * language's plural rules) when those exist. A missing key renders as `*key*`.
	 */
	const textGet = (key: string, values?: TextValues): string => {
		if (typeof values?.count === "number") {
			const form = plurals.select(values.count);
			const plural = messages[`${key}.${form}`] ?? messages[`${key}.other`];
			if (plural) return interpolate(plural, values);
		}
		const message = messages[key];
		if (!message) return `*${key}*`;
		return values ? interpolate(message, values) : message;
	};

	// Formatters return "" for a missing or invalid value instead of throwing;
	// the caller decides what to show in its place.
	/**
	 * In the browser's time zone, unless `timeZone` pins one: a date that
	 * belongs to a place's calendar (e.g. subscription expiry, Ecuador).
	 */
	const formatDate = (
		value: DateInput,
		style: DateStyle = "medium",
		{ timeZone }: { timeZone?: string } = {},
	) => {
		const date = toDate(value);
		if (!date) return "";
		return timeZone
			? new Intl.DateTimeFormat(locale, {
					...DATE_STYLES[style],
					timeZone,
				}).format(date)
			: dateFormats[style].format(date);
	};

	const formatDateTime = (value: DateInput) => {
		const date = toDate(value);
		return date ? dateTimeFormat.format(date) : "";
	};

	/** "2:05:09 p. m." / "2:05:09 PM". */
	const formatTime = (value: DateInput) => {
		const date = toDate(value);
		return date ? timeFormat.format(date) : "";
	};

	const formatMoney = (amount: number | null | undefined) =>
		typeof amount === "number" && Number.isFinite(amount)
			? moneyFormat.format(amount)
			: "";

	/**
	 * "hace 3 minutos" / "3 minutes ago"; with `suffix: false`, "3 minutos"
	 * for messages that already carry the "hace"/"ago" wording.
	 */
	const formatRelative = (value: DateInput, { suffix = true } = {}) => {
		const date = toDate(value);
		return date
			? formatDistanceToNow(date, {
					addSuffix: suffix,
					locale: DATE_FNS_LOCALES[lang],
				})
			: "";
	};

	return {
		textGet,
		formatDate,
		formatDateTime,
		formatTime,
		formatMoney,
		formatRelative,
	};
}

/**
 * Text and formats in the interface language: the language of the messages
 * currently loaded (it changes when they arrive, not when it is picked), so
 * text, dates and amounts never disagree.
 */
const useText = () => {
	const { messages, lang } = useMessageStore();
	return React.useMemo(
		() => createText(lang ?? "es", messages),
		[lang, messages],
	);
};

/** What `useText` returns, for code that receives it (column builders, helpers). */
export type AppText = ReturnType<typeof createText>;

export { useText };
