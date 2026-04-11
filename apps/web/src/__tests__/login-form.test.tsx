import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { LoginForm } from "@/sections/forms/login/login-form";

const mockUserLogin = vi.fn();

vi.mock("@/api/auth-service", () => ({
	userLogin: mockUserLogin,
}));

vi.mock("react-router", () => ({
	useNavigate: () => vi.fn(),
}));

vi.mock("@/store/token-store", () => ({
	useTokenStore: () => ({
		setToken: vi.fn(),
	}),
}));

vi.mock("@/hooks/use-text", () => ({
	useText: () => ({
		textGet: (key: string) => key,
	}),
}));

describe("LoginForm", () => {
	beforeEach(() => {
		mockUserLogin.mockClear();
	});

	it("renders username and password inputs", () => {
		render(<LoginForm />);

		expect(screen.getByPlaceholderText(/user_name|username/i)).toBeDefined();
		expect(screen.getByPlaceholderText(/password/i)).toBeDefined();
	});

	it("does not call userLogin when fields are empty and form is submitted", async () => {
		render(<LoginForm />);

		const submitButton = screen.getByRole("button", { name: /login|submit/i });
		fireEvent.click(submitButton);

		await waitFor(() => {
			expect(mockUserLogin).not.toHaveBeenCalled();
		});
	});

	it("calls userLogin with valid credentials", async () => {
		mockUserLogin.mockResolvedValue({
			success: true,
			data: { token: "test-token", exchange_token: "ex-token" },
		});

		render(<LoginForm />);

		const usernameInput = screen.getByPlaceholderText(
			/user_name|username/i,
		) as HTMLInputElement;
		const passwordInput = screen.getByPlaceholderText(
			/password/i,
		) as HTMLInputElement;

		fireEvent.change(usernameInput, { target: { value: "testuser" } });
		fireEvent.change(passwordInput, { target: { value: "password123" } });

		const submitButton = screen.getByRole("button", { name: /login|submit/i });
		fireEvent.click(submitButton);

		await waitFor(() => {
			expect(mockUserLogin).toHaveBeenCalledWith("testuser", "password123");
		});
	});
});
