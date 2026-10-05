import { useText } from "@pengi/shared";
import { Button, Input } from "@pengi/ui";
import { Monitor } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";

/** A display token: 32 base64url characters (see tenant_models.NewDisplayToken). */
const DISPLAY_TOKEN_RE = /^[A-Za-z0-9_-]{32}$/;

/** Takes the pasted waiting-room link (or the bare token) and returns the token. */
export function extractDisplayToken(input: string): string | null {
	const value = input.trim();
	if (DISPLAY_TOKEN_RE.test(value)) return value;
	try {
		const token = new URL(value).searchParams.get("token") ?? "";
		return DISPLAY_TOKEN_RE.test(token) ? token : null;
	} catch {
		return null;
	}
}

const PairDisplayPage = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [value, setValue] = React.useState("");
	const [error, setError] = React.useState(false);

	const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		setValue(e.target.value);
		setError(false);
	};

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		const token = extractDisplayToken(value);
		if (!token) {
			setError(true);
			return;
		}
		navigate(`/display/waiting-room?token=${encodeURIComponent(token)}`);
	};

	return (
		<div className="min-h-screen bg-background flex items-center justify-center p-6">
			<div className="w-full max-w-md space-y-8 text-center">
				<div className="flex flex-col items-center gap-3">
					<div className="rounded-full bg-primary/10 p-3">
						<Monitor className="h-7 w-7 text-primary" />
					</div>
					<h1 className="text-xl font-bold">{textGet("display.pair.title")}</h1>
					<p className="text-muted-foreground text-sm">
						{textGet("display.pair.hint")}
					</p>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					<Input
						value={value}
						onChange={handleChange}
						placeholder={textGet("display.pair.placeholder")}
						autoComplete="off"
						spellCheck={false}
						className={`text-center font-mono h-12 ${error ? "border-destructive" : ""}`}
						autoFocus
					/>
					{error && (
						<p className="text-destructive text-sm">
							{textGet("display.pair.invalid")}
						</p>
					)}
					<Button
						type="submit"
						className="w-full"
						size="lg"
						disabled={!value.trim()}
					>
						{textGet("display.pair.submit")}
					</Button>
				</form>
			</div>
		</div>
	);
};

export default PairDisplayPage;
