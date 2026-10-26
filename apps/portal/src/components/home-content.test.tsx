import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Me, PortalSections } from "@/api/portal-service";
import { HomeContent } from "./home-content";

const empty = { success: true, code: 200, message: "", data: [] };

vi.mock("@/api/portal-service", () => ({
	getAppointments: vi.fn(async () => empty),
	getResults: vi.fn(async () => empty),
	getDocuments: vi.fn(async () => empty),
	getInvoices: vi.fn(async () => empty),
	downloadResultFile: vi.fn(),
	downloadDocument: vi.fn(),
	downloadInvoiceRide: vi.fn(),
	downloadInvoiceXml: vi.fn(),
}));

const meWith = (sections: PortalSections): Me => ({
	patient: {
		first_name: "Ana",
		last_name: "Paz",
		document: "1712345678",
		email: "",
		phone: "",
	},
	clinic: { name: "Clínica Sol", slug: "sol", has_logo: false, sections },
});

describe("HomeContent", () => {
	it("shows only the sections the clinic enabled", async () => {
		render(
			<HomeContent
				me={meWith({
					appointments: true,
					results: false,
					documents: true,
					invoices: false,
				})}
				onLogout={vi.fn()}
			/>,
		);

		// Each section is in the top bar (desktop) and the bottom bar (phone).
		const section = (key: string) =>
			screen.queryAllByRole("button", { name: `*portal.section.${key}*` });
		expect(section("appointments")).toHaveLength(2);
		expect(section("documents")).toHaveLength(2);
		expect(section("results")).toHaveLength(0);
		expect(section("invoices")).toHaveLength(0);
		expect(
			await screen.findByText("*portal.appointments.empty*"),
		).toBeVisible();
	});

	it("shows the clinic name and the sign out button", async () => {
		render(
			<HomeContent
				me={meWith({
					appointments: false,
					results: false,
					documents: false,
					invoices: true,
				})}
				onLogout={vi.fn()}
			/>,
		);

		expect(screen.getByText("Clínica Sol")).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: "*portal.logout*" }),
		).toBeInTheDocument();
		expect(await screen.findByText("*portal.invoices.empty*")).toBeVisible();
	});
});
