import type { InternalAxiosRequestConfig } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useSessionStore } from "@/store/session-store";
import { portalApi } from "./http-clients";
import {
	createNewPatientRequest,
	downloadDocument,
	downloadInvoiceRide,
	downloadInvoiceXml,
	downloadResultFile,
	getBookingOptions,
	getClinic,
	getDocuments,
	guestSlotsLoader,
	joinWaitlist,
	leaveWaitlist,
	requestCode,
} from "./portal-service";

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const originalAdapter = portalApi.defaults.adapter;
let calls: InternalAxiosRequestConfig[] = [];

beforeEach(() => {
	calls = [];
	useSessionStore.getState().setAccessToken("access-1");
	portalApi.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
		calls.push(config);
		const data =
			config.responseType === "blob"
				? new Blob(["%PDF"])
				: { code: 200, message: "", data: [] };
		return { data, status: 200, statusText: "", headers: {}, config };
	};
});

afterEach(() => {
	portalApi.defaults.adapter = originalAdapter;
	useSessionStore.getState().clear();
});

const last = () => calls[calls.length - 1];

describe("portal-service", () => {
	it("encodes the clinic slug in public routes", async () => {
		await getClinic("sol/../x");
		expect(last().url).toBe("/clinics/sol%2F..%2Fx");
		await requestCode("sol", { document: "1710034065", website: "" });
		expect(last().method).toBe("post");
		expect(last().url).toBe("/clinics/sol/login");
		expect(JSON.parse(last().data)).toEqual({
			document: "1710034065",
			website: "",
		});
	});

	it("reads lists with the session token", async () => {
		const res = await getDocuments();
		expect(res.success).toBe(true);
		expect(last().url).toBe("/documents");
		expect(last().headers.Authorization).toBe("Bearer access-1");
	});

	it("downloads files as blobs from the patient's own routes", async () => {
		const cases: [() => Promise<unknown>, string][] = [
			[() => downloadResultFile(5), "/results/files/5"],
			[
				() => downloadDocument("prescription", 6),
				"/documents/prescription/6/file",
			],
			[() => downloadInvoiceRide(7), "/invoices/7/ride"],
			[() => downloadInvoiceXml(7), "/invoices/7/xml"],
		];
		for (const [call, url] of cases) {
			const res = (await call()) as { success: boolean; data: unknown };
			expect(last().url).toBe(url);
			expect(last().responseType).toBe("blob");
			expect(res.success).toBe(true);
			expect(res.data).toBeInstanceOf(Blob);
		}
	});

	it("uses the guest token, not the session, for new patients", async () => {
		await getBookingOptions("guest-1");
		expect(last().headers.Authorization).toBe("Bearer guest-1");
		await guestSlotsLoader("guest-1")(
			3,
			{ doctor_id: 9 },
			"2026-12-01",
			"2026-12-07",
		);
		expect(last().url).toBe("/booking/slots");
		expect(last().params).toEqual({
			type_id: 3,
			doctor_id: 9,
			from: "2026-12-01",
			to: "2026-12-07",
		});
		expect(last().headers.Authorization).toBe("Bearer guest-1");
		await createNewPatientRequest("guest-1", {
			type_id: 3,
			date: "2026-12-01",
			start_time: "09:00",
			patient: {
				first_name: "Ana",
				last_name: "Ruiz",
				document_type: "cedula",
				document: "1710034065",
				birth_date: "1990-05-01",
			},
		});
		expect(last().url).toBe("/new-patient/requests");
		expect(last().headers.Authorization).toBe("Bearer guest-1");
		// Without a guest token, the catalog goes with the session.
		await getBookingOptions();
		expect(last().headers.Authorization).toBe("Bearer access-1");
	});

	it("joins the waitlist with or without a doctor and leaves by id", async () => {
		await joinWaitlist(3);
		expect(JSON.parse(last().data)).toEqual({ type_id: 3 });
		await joinWaitlist(3, 9);
		expect(JSON.parse(last().data)).toEqual({ type_id: 3, doctor_id: 9 });
		await leaveWaitlist(4);
		expect(last().method).toBe("delete");
		expect(last().url).toBe("/waitlist/4");
	});
});
