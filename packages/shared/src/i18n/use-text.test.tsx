import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useMessageStore } from "./message-store";
import { useText } from "./use-text";

const MESSAGES = {
	es: {
		greeting: "Hola {name}",
		"tasks.total.one": "{count} tarea",
		"tasks.total.other": "{count} tareas",
		"template.error": "Revisa las etiquetas {{...}} y {unknown}",
	},
	en: {
		greeting: "Hello {name}",
		"tasks.total.one": "{count} task",
		"tasks.total.other": "{count} tasks",
	},
};

function loadLanguage(lang: "es" | "en") {
	act(() => {
		useMessageStore.setState({ lang, messages: MESSAGES[lang] });
	});
}

// 27 Sep 2026, 14:05 local time.
const DATE = new Date(2026, 8, 27, 14, 5);

beforeEach(() => loadLanguage("es"));

describe("useText · textGet", () => {
	it("fills {name} placeholders", () => {
		const { result } = renderHook(() => useText());
		expect(result.current.textGet("greeting", { name: "Ana" })).toBe(
			"Hola Ana",
		);
	});

	it("leaves placeholders without a value, and non-placeholders, as they are", () => {
		const { result } = renderHook(() => useText());
		expect(result.current.textGet("template.error", {})).toBe(
			"Revisa las etiquetas {{...}} y {unknown}",
		);
	});

	it("picks the plural form from count", () => {
		const { result } = renderHook(() => useText());
		expect(result.current.textGet("tasks.total", { count: 1 })).toBe("1 tarea");
		expect(result.current.textGet("tasks.total", { count: 0 })).toBe(
			"0 tareas",
		);
		expect(result.current.textGet("tasks.total", { count: 5 })).toBe(
			"5 tareas",
		);
	});

	it("renders a missing key as *key*", () => {
		const { result } = renderHook(() => useText());
		expect(result.current.textGet("nope")).toBe("*nope*");
		expect(result.current.textGet("nope", { count: 2 })).toBe("*nope*");
	});
});

describe("useText · formats follow the interface language", () => {
	it("formats dates, amounts and plurals in Spanish (es-EC)", () => {
		const { result } = renderHook(() => useText());
		expect(result.current.formatDate(DATE)).toBe("27 sept 2026");
		expect(result.current.formatDate(DATE, "long")).toBe(
			"27 de septiembre de 2026",
		);
		expect(result.current.formatDate(DATE, "day-month")).toBe("27 sept");
		expect(result.current.formatDate(DATE, "month-year")).toBe("sept 2026");
		expect(result.current.formatDateTime(DATE)).toMatch(/^27 sept 2026, 2:05/);
		expect(result.current.formatTime(DATE)).toMatch(/^2:05:00/);
		expect(result.current.formatMoney(1234.5)).toBe("$1.234,50");
		expect(result.current.formatFileSize(12.4 * 1024)).toMatch(/^12,4\s?kB$/);
		expect(result.current.formatFileSize(3.2 * 1024 * 1024)).toMatch(
			/^3,2\s?MB$/,
		);
		expect(result.current.formatFileSize(10 * 1024 ** 3)).toMatch(/^10\s?GB$/);
	});

	it("switches every format when the messages of another language arrive", () => {
		const { result } = renderHook(() => useText());
		loadLanguage("en");
		expect(result.current.textGet("tasks.total", { count: 1 })).toBe("1 task");
		expect(result.current.formatDate(DATE)).toBe("Sep 27, 2026");
		expect(result.current.formatDateTime(DATE)).toBe("Sep 27, 2026, 2:05 PM");
		expect(result.current.formatTime(DATE)).toBe("2:05:00 PM");
		expect(result.current.formatMoney(1234.5)).toBe("$1,234.50");
		expect(result.current.formatFileSize(12.4 * 1024)).toMatch(/^12\.4\s?kB$/);
		expect(result.current.formatFileSize(512)).toMatch(/^512\s?byte/);
	});

	it("formats relative time in the interface language", () => {
		vi.useFakeTimers({ now: new Date(DATE.getTime() + 3 * 60_000) });
		try {
			const { result } = renderHook(() => useText());
			expect(result.current.formatRelative(DATE)).toBe("hace 3 minutos");
			expect(result.current.formatRelative(DATE, { suffix: false })).toBe(
				"3 minutos",
			);
			loadLanguage("en");
			expect(result.current.formatRelative(DATE)).toBe("3 minutes ago");
		} finally {
			vi.useRealTimers();
		}
	});

	it("formats in a pinned time zone", () => {
		const { result } = renderHook(() => useText());
		// End of 16 Nov in Ecuador (UTC-5) is 17 Nov in UTC.
		expect(
			result.current.formatDate("2026-11-17T04:59:59Z", "medium", {
				timeZone: "America/Guayaquil",
			}),
		).toBe("16 nov 2026");
	});

	it("formats the short weekday of a bare calendar date in each language", () => {
		const { result } = renderHook(() => useText());
		// 2026-09-28 is a Monday, whatever the time zone of the machine.
		expect(result.current.formatDate("2026-09-28", "weekday")).toBe("lun");
		expect(result.current.formatDate("2026-10-04", "weekday")).toBe("dom");
		loadLanguage("en");
		expect(result.current.formatDate("2026-09-28", "weekday")).toBe("Mon");
		expect(result.current.formatDate("2026-10-04", "weekday")).toBe("Sun");
	});

	it("accepts ISO strings", () => {
		const { result } = renderHook(() => useText());
		expect(result.current.formatDate(DATE.toISOString())).toBe("27 sept 2026");
	});

	it("returns an empty string for missing or invalid values instead of throwing", () => {
		const { result } = renderHook(() => useText());
		const {
			formatDate,
			formatDateTime,
			formatTime,
			formatMoney,
			formatFileSize,
			formatRelative,
		} = result.current;
		for (const value of [null, undefined, "", "not a date"]) {
			expect(formatDate(value)).toBe("");
			expect(formatDateTime(value)).toBe("");
			expect(formatTime(value)).toBe("");
			expect(formatRelative(value)).toBe("");
		}
		expect(formatMoney(null)).toBe("");
		expect(formatMoney(Number.NaN)).toBe("");
		expect(formatFileSize(null)).toBe("");
		expect(formatFileSize(Number.NaN)).toBe("");
	});
});
