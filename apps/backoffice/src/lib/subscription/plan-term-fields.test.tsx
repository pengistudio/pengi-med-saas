import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { describe, expect, it } from "vitest";
import type { Plan } from "@/api/plan-service";
import { type PlanTerm, PlanTermFields } from "./plan-term-fields";

const pro = {
	ID: 1,
	name: "Pro",
	code: "pro",
	price: 30,
	pricings: [
		{ months: 12, price: 300 },
		{ months: 1, price: 30 },
		{ months: 3, price: 90 },
	],
} as Plan;

const today = new Date("2027-01-31T20:00:00-05:00");

function Harness({ initial }: { initial: PlanTerm }) {
	const [term, setTerm] = React.useState(initial);
	return (
		<>
			<PlanTermFields
				plans={[pro]}
				value={term}
				onChange={setTerm}
				now={today}
			/>
			<output>{JSON.stringify(term)}</output>
		</>
	);
}

describe("PlanTermFields", () => {
	it("offers the plan's periods in order and suggests the expiry of the one picked", async () => {
		const user = userEvent.setup();
		render(<Harness initial={{ planCode: "pro", expiresAt: "" }} />);

		const periods = screen.getAllByRole("button", {
			name: /subscription\.plans\.period/,
		});
		expect(periods.map((b) => b.textContent)).toEqual([
			expect.stringContaining("period.1"),
			expect.stringContaining("period.3"),
			expect.stringContaining("period.12"),
		]);

		await user.click(periods[0]);
		expect(screen.getByRole("status")).toHaveTextContent(
			'"expiresAt":"2027-02-28"',
		);

		await user.click(periods[2]);
		expect(screen.getByRole("status")).toHaveTextContent(
			'"expiresAt":"2028-01-31"',
		);
	});

	it("lets the admin type any expiry date", async () => {
		const user = userEvent.setup();
		render(<Harness initial={{ planCode: "pro", expiresAt: "2027-02-28" }} />);

		const input = screen.getByLabelText(
			"*backoffice.subscriptions.col.expires*",
		);
		await user.clear(input);
		await user.type(input, "2027-06-15");

		expect(screen.getByRole("status")).toHaveTextContent(
			'"expiresAt":"2027-06-15"',
		);
	});
});
