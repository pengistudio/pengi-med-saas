import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { describe, expect, it } from "vitest";
import type { Permission } from "@/api/permission-service";
import { PermissionPicker } from "./permission-picker";

const catalog: Permission[] = [
	{
		ID: "VIEW_PATIENTS",
		name: "Ver pacientes",
		category: "Clínico",
		description: "",
	},
	{
		ID: "EDIT_PATIENTS",
		name: "Editar pacientes",
		category: "Clínico",
		description: "",
	},
	{
		ID: "CREATE_INVOICE",
		name: "Crear factura",
		category: "Facturación",
		description: "",
	},
	{ ID: "EXPORT", name: "Exportar", category: "", description: "" },
];

const load = async () => ({
	success: true as const,
	code: 200,
	message: "",
	data: catalog,
});

function Harness({ initial = [] as string[] }) {
	const [ids, setIds] = React.useState(initial);
	return (
		<>
			<PermissionPicker value={ids} onChange={setIds} load={load} />
			<output>{[...ids].sort().join(",")}</output>
		</>
	);
}

const selected = () => screen.getByRole("status").textContent;

describe("PermissionPicker", () => {
	it("groups permissions by category, uncategorised ones under General", async () => {
		render(<Harness />);

		expect(await screen.findByText("Clínico")).toBeInTheDocument();
		expect(screen.getByText("Facturación")).toBeInTheDocument();
		expect(screen.getByText("General")).toBeInTheDocument();
		expect(
			screen.getByText("0 / 4 *backoffice.linking.selected*"),
		).toBeInTheDocument();
	});

	it("toggles a permission by clicking its name", async () => {
		const user = userEvent.setup();
		render(<Harness />);

		await user.click(await screen.findByText("Ver pacientes"));
		expect(selected()).toBe("VIEW_PATIENTS");

		await user.click(screen.getByText("Ver pacientes"));
		expect(selected()).toBe("");
	});

	it("selects and clears a whole category, keeping the rest", async () => {
		const user = userEvent.setup();
		render(<Harness initial={["EXPORT"]} />);

		const clinical = (await screen.findByText("Clínico")).closest(
			"section",
		) as HTMLElement;
		await user.click(
			within(clinical).getByRole("button", {
				name: "*backoffice.linking.select_all*",
			}),
		);
		expect(selected()).toBe("EDIT_PATIENTS,EXPORT,VIEW_PATIENTS");

		await user.click(
			within(clinical).getByRole("button", {
				name: "*backoffice.linking.deselect_all*",
			}),
		);
		expect(selected()).toBe("EXPORT");
	});

	it("selects and clears everything", async () => {
		const user = userEvent.setup();
		render(<Harness initial={["EXPORT"]} />);
		await screen.findByText("Clínico");

		const [all] = screen.getAllByRole("button", {
			name: "*backoffice.linking.select_all*",
		});
		await user.click(all);
		expect(selected()).toBe(
			"CREATE_INVOICE,EDIT_PATIENTS,EXPORT,VIEW_PATIENTS",
		);
		expect(
			screen.getByText("4 / 4 *backoffice.linking.selected*"),
		).toBeInTheDocument();

		const [none] = screen.getAllByRole("button", {
			name: "*backoffice.linking.deselect_all*",
		});
		await user.click(none);
		expect(selected()).toBe("");
	});
});
