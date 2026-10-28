import { screen } from "@testing-library/react";

/** The code field's first slot: typing there fills the next slots. */
export async function findCodeField() {
	const [first] = await screen.findAllByRole("textbox", {
		name: "*portal.signin.code*",
	});
	return first;
}
