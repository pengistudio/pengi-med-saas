import { Monitor } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const PairDisplayPage = () => {
	const navigate = useNavigate();
	const [code, setCode] = React.useState("");
	const [error, setError] = React.useState(false);

	const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const value = e.target.value.replace(/\D/g, "").slice(0, 8);
		setCode(value);
		setError(false);
	};

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		if (code.length !== 8) {
			setError(true);
			return;
		}
		navigate(`/display/waiting-room?token=${code}`);
	};

	return (
		<div className="min-h-screen bg-background flex items-center justify-center p-6">
			<div className="w-full max-w-sm space-y-8 text-center">
				<div className="flex flex-col items-center gap-3">
					<div className="rounded-full bg-primary/10 p-4">
						<Monitor className="h-10 w-10 text-primary" />
					</div>
					<h1 className="text-2xl font-bold">Conectar pantalla</h1>
					<p className="text-muted-foreground text-sm">
						Ingresa el código de 8 dígitos que aparece en la aplicación
					</p>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					<Input
						value={code}
						onChange={handleChange}
						placeholder="00000000"
						inputMode="numeric"
						maxLength={8}
						className={`text-center text-3xl font-mono tracking-[0.5em] h-16 ${error ? "border-destructive" : ""}`}
						autoFocus
					/>
					{error && (
						<p className="text-destructive text-sm">
							El código debe tener 8 dígitos
						</p>
					)}
					<Button
						type="submit"
						className="w-full"
						size="lg"
						disabled={code.length !== 8}
					>
						Conectar
					</Button>
				</form>
			</div>
		</div>
	);
};

export default PairDisplayPage;
