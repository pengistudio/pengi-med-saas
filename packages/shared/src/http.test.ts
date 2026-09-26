import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createHttpService } from "./http";

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

/** An axios client whose "API" answers every request with `status` + `body`. */
function api(status: number, body: unknown) {
	return createHttpService(
		axios.create({
			adapter: async (config: InternalAxiosRequestConfig) => {
				const response = {
					data: body,
					status,
					statusText: "",
					headers: {},
					config,
				};
				if (status >= 400) {
					throw new AxiosError(
						"fail",
						"ERR_BAD_RESPONSE",
						config,
						null,
						response,
					);
				}
				return response;
			},
		}),
	);
}

describe("HttpService (the API envelope)", () => {
	beforeEach(() => vi.clearAllMocks());

	it("unwraps a success envelope", async () => {
		const res = await api(200, {
			code: 200,
			message: "Plan creado",
			data: { ID: 1 },
		}).post("/plans", {}, { notifySuccess: true });

		expect(res).toEqual({
			success: true,
			code: 200,
			message: "Plan creado",
			data: { ID: 1 },
		});
		expect(toast.success).toHaveBeenCalledWith("Plan creado");
	});

	it("turns an error envelope into a failed response with the API's error", async () => {
		const res = await api(400, {
			code: 400,
			message: "Solicitud inválida",
			data: { error_code: "E-BO-001", error_message: "Datos inválidos" },
		}).put("/plans/1", {}, { notifyError: true });

		expect(res).toEqual({
			success: false,
			code: 400,
			message: "Solicitud inválida",
			data: { error_code: "E-BO-001", error_message: "Datos inválidos" },
		});
		expect(toast.error).toHaveBeenCalledWith("Solicitud inválida");
	});

	it("reports a failure without an envelope (e.g. network or proxy error)", async () => {
		const res = await api(502, "<html>Bad Gateway</html>").get("/plans");

		expect(res).toMatchObject({
			success: false,
			code: 502,
			data: { error_code: "UNKNOWN" },
		});
		expect(toast.error).not.toHaveBeenCalled();
	});

	it("uses a given toast text instead of the API's message", async () => {
		await api(200, { code: 200, message: "ok", data: null }).delete(
			"/plans/1",
			{
				notifySuccess: "Eliminado",
			},
		);
		expect(toast.success).toHaveBeenCalledWith("Eliminado");
	});
});
