import { UiTextProvider } from "@pengi/ui";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/api", () => ({ apiWithTenant: {} }));

import type { ExamCatalogItem, ExamProfile } from "@/api/exam-order-service";
import { ExamPicker } from "@/components/features/exam-orders/exam-picker";
import type { DraftExamItem } from "@/lib/exam-orders";

const exam = (
	ID: number,
	name: string,
	category: ExamCatalogItem["category"],
	subgroup: string,
	active = true,
) =>
	({
		ID,
		name,
		category,
		subgroup,
		default_indications: "",
		active,
	}) as ExamCatalogItem;

const catalog = [
	exam(1, "Hemograma", "laboratory", "Hematología"),
	exam(2, "Ecografía abdominal", "imaging", "Ecografía"),
	exam(3, "Colesterol", "laboratory", "Química"),
	exam(4, "Retirado", "laboratory", "Química", false),
];
const profile = {
	ID: 1,
	name: "Chequeo",
	active: true,
	items: [catalog[0], catalog[2]],
} as ExamProfile;

function renderPicker(value: DraftExamItem[] = []) {
	const onChange = vi.fn();
	render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<ExamPicker
				catalog={catalog}
				profiles={[profile]}
				value={value}
				onChange={onChange}
			/>
		</UiTextProvider>,
	);
	return onChange;
}

describe("ExamPicker", () => {
	it("shows the active catalog grouped by subgroup", () => {
		renderPicker();
		expect(screen.getByText("Hematología")).toBeTruthy();
		expect(screen.getByText("Ecografía")).toBeTruthy();
		expect(screen.getByText("Colesterol")).toBeTruthy();
		expect(screen.queryByText("Retirado")).toBeNull();
	});

	it("filters with the search", () => {
		renderPicker();
		fireEvent.change(screen.getAllByRole("searchbox")[0], {
			target: { value: "ecografia" },
		});
		expect(screen.getByText("Ecografía abdominal")).toBeTruthy();
		expect(screen.queryByText("Hemograma")).toBeNull();
	});

	it("adds a profile's exams at once", () => {
		const onChange = renderPicker();
		fireEvent.click(screen.getByRole("button", { name: /Chequeo/ }));
		const drafts: DraftExamItem[] = onChange.mock.calls[0][0];
		expect(drafts.map((d) => d.catalog_item_id)).toEqual([1, 3]);
	});

	it("adds an exam written by hand", () => {
		const onChange = renderPicker();
		const input = screen.getByPlaceholderText(
			/clinical.exam_orders.picker.free_text.placeholder/,
		);
		fireEvent.change(input, { target: { value: "Prueba especial" } });
		fireEvent.keyDown(input, { key: "Enter" });
		const drafts: DraftExamItem[] = onChange.mock.calls[0][0];
		expect(drafts).toHaveLength(1);
		expect(drafts[0]).toMatchObject({
			name: "Prueba especial",
			category: "other",
		});
		expect(drafts[0].catalog_item_id).toBeUndefined();
	});
});
