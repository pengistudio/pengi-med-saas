import { UiTextProvider } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import { render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DataTable } from "@/components/custom/table/data-table";

vi.mock("@pengi/shared", async (importOriginal) => ({
	...(await importOriginal<typeof import("@pengi/shared")>()),
	useText: () => ({ textGet: (key: string) => key }),
}));

function stubViewport({ phone }: { phone: boolean }) {
	vi.stubGlobal("matchMedia", (media: string) => ({
		matches: phone,
		media,
		addEventListener: () => {},
		removeEventListener: () => {},
	}));
}

type Invoice = { ID: number; number: string; patient: string; total: string };

const columns: ColumnDef<Invoice>[] = [
	{ accessorKey: "patient", header: "Paciente", meta: { phone: "subtitle" } },
	{ accessorKey: "number", header: "Número", meta: { phone: "title" } },
	{ accessorKey: "total", header: "Total", meta: { phone: "end" } },
	{
		id: "notes",
		header: "Notas",
		cell: () => "sin notas",
		meta: { title: "billing.notes" },
	},
	{
		id: "actions",
		cell: () => <button type="button">acciones</button>,
	},
];

const data: Invoice[] = [
	{ ID: 1, number: "001-001-1", patient: "Ana Pérez", total: "$45.00" },
];

const renderTable = (cols: ColumnDef<Invoice>[]) =>
	render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<DataTable columns={cols} data={data} />
		</UiTextProvider>,
	);

describe("DataTable", () => {
	afterEach(() => vi.unstubAllGlobals());

	it("reflows into a list on a phone, placing columns by meta.phone", () => {
		stubViewport({ phone: true });
		renderTable(columns);

		expect(screen.queryByRole("table")).not.toBeInTheDocument();
		const [entry] = screen.getAllByRole("listitem");
		const [firstLine, secondLine, details] =
			entry.querySelectorAll(":scope > div > div");
		expect(firstLine).toHaveTextContent("001-001-1$45.00");
		expect(secondLine).toHaveTextContent("Ana Pérez");
		// Details carry their meta.title as a label.
		expect(details).toHaveTextContent("billing.notessin notas");
		expect(
			within(entry).getByRole("button", { name: "acciones" }),
		).toBeVisible();
	});

	it("leads with the first column when none is placed as title", () => {
		stubViewport({ phone: true });
		const plain: ColumnDef<Invoice>[] = [
			{ accessorKey: "patient", header: "Paciente" },
			{ accessorKey: "total", header: "Total" },
		];
		renderTable(plain);

		const firstLine = screen
			.getByRole("listitem")
			.querySelector(":scope > div > div");
		expect(firstLine).toHaveTextContent("Ana Pérez");
	});

	it("renders a table on a desktop", () => {
		stubViewport({ phone: false });
		renderTable(columns);

		expect(screen.getByRole("table")).toBeInTheDocument();
		expect(screen.queryByRole("list")).not.toBeInTheDocument();
	});
});
