import { Form } from "@pengi/ui";
import {
	act,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { Toaster } from "sonner";
import { beforeEach, describe, expect, it } from "vitest";
import * as z from "zod";
import { AppTextBridge } from "./app-text-bridge";
import { LanguageProvider, useLanguage } from "./language-context";
import { useMessageStore } from "./message-store";

const schema = z.object({
	name: z.string().min(1),
	prices: z.array(z.object({ amount: z.number().min(1) })),
});

function SwitchToEnglish() {
	const { changeLanguage } = useLanguage();
	return (
		<button type="button" onClick={() => changeLanguage("en")}>
			english
		</button>
	);
}

function renderForm() {
	return render(
		<LanguageProvider>
			<AppTextBridge>
				<Toaster />
				<SwitchToEnglish />
				<Form
					schema={schema}
					defaultValues={{ name: "", prices: [{ amount: 0 }, { amount: 0 }] }}
					onSubmit={() => {}}
				>
					{(form) => (
						<>
							<p data-testid="name-error">
								{form.formState.errors.name?.message}
							</p>
							<button type="submit">save</button>
						</>
					)}
				</Form>
			</AppTextBridge>
		</LanguageProvider>,
	);
}

describe("Form inside the app's i18n providers", () => {
	beforeEach(() => {
		useMessageStore.setState({
			lang: "es",
			messages: {
				"form.validation.invalid_fields": "Hay campos inválidos",
				"form.validation.multiple_fields_error": "{count} campos con errores",
			},
		});
	});

	it("counts nested field errors in the invalid-submit toast", async () => {
		renderForm();

		await act(async () => fireEvent.click(screen.getByText("save")));

		expect(await screen.findByText("Hay campos inválidos")).toBeInTheDocument();
		expect(screen.getByText("3 campos con errores")).toBeInTheDocument();
	});

	it("re-validates in the new language after a failed submit", async () => {
		renderForm();

		await act(async () => fireEvent.click(screen.getByText("save")));
		expect(screen.getByTestId("name-error")).toHaveTextContent(
			/Demasiado pequeño/,
		);

		await act(async () => fireEvent.click(screen.getByText("english")));

		await waitFor(() =>
			expect(screen.getByTestId("name-error")).toHaveTextContent(/Too small/),
		);
	});
});
