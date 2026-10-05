import { useMessageStore } from "@pengi/shared";
import { UiTextProvider } from "@pengi/ui";
import { render, screen } from "@testing-library/react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

vi.mock("@/api", () => ({ apiWithTenant: {} }));

import type { ExamOrderItem } from "@/api/exam-order-service";
import type { PatientAttachment } from "@/api/patient-attachment-service";
import { ExamResultItem } from "@/components/features/exam-orders/exam-result-item";

// Ecuador: UTC midnight of the 4th is 19:00 of the 3rd.
const ORIGINAL_TZ = process.env.TZ;
beforeAll(() => {
	process.env.TZ = "America/Guayaquil";
	useMessageStore.setState({ lang: "es", messages: {} });
});
afterAll(() => {
	process.env.TZ = ORIGINAL_TZ;
});

const attachment = {
	ID: 5,
	file_name: "hemograma.pdf",
	taken_at: "2026-10-04T00:00:00Z",
} as PatientAttachment;

const item = {
	ID: 1,
	name: "Hemograma",
	category: "laboratory",
	subgroup: "",
	indications: "",
	attachments: [attachment],
	reviewed_at: "2026-10-04T15:00:00Z",
	reviewed_by_name: "Dra. Ruiz",
} as unknown as ExamOrderItem;

function renderItem() {
	render(
		<UiTextProvider value={{ textGet: (key) => key }}>
			<ul>
				<ExamResultItem
					item={item}
					canReview={false}
					selected={false}
					onSelect={() => {}}
					canUpload={false}
					onUpload={() => {}}
					canViewFiles={false}
					canDeleteFile={() => false}
					onView={() => {}}
					onDownload={() => {}}
					onDelete={() => {}}
				/>
			</ul>
		</UiTextProvider>,
	);
}

describe("ExamResultItem", () => {
	it("runs behind UTC (the bug: the 4th read as the 3rd)", () => {
		expect(new Date(attachment.taken_at).getDate()).toBe(3);
	});

	it("shows the result's exam day as a calendar date", () => {
		renderItem();
		expect(screen.getByText("4 oct 2026")).toBeTruthy();
		expect(screen.queryByText("3 oct 2026")).toBeNull();
	});
});
