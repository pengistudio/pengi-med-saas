import { Building2, ChevronRight, LogOut } from "lucide-react";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import { getEnvironmentsFromUser } from "@/api/user-service";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { useText } from "@/hooks/use-text";
import { useSessionStore } from "@/store/session-store";
import type { EnvironmentWithCompany } from "@/types/user-type";

export default function LoginEnvironments() {
	const [searchParams] = useSearchParams();
	const navigate = useNavigate();
	const { textGet } = useText();
	const setEnvironment = useSessionStore((state) => state.setEnvironment);
	const exchangeToken = searchParams.get("exchange_token");
	const [environments, setEnvironments] = React.useState<
		EnvironmentWithCompany[]
	>([]);

	const handleSelectEnvironment = React.useCallback(
		(envId: number) => {
			const selectedEnv = environments.find((e) => e.ID === envId);
			if (selectedEnv) {
				setEnvironment(selectedEnv);
			}
			navigate("/");
		},
		[environments, navigate, setEnvironment],
	);

	React.useEffect(() => {
		if (environments.length === 1) {
			handleSelectEnvironment(environments[0].ID);
		}
	}, [environments, handleSelectEnvironment]);

	React.useEffect(() => {
		if (!exchangeToken) {
			navigate("/login");
			return;
		}
		getEnvironmentsFromUser(exchangeToken).then((response) => {
			if (response.success) {
				setEnvironments(response.data);
			} else {
				navigate("/login");
			}
		});
	}, [exchangeToken, navigate]);

	const handleLogout = () => {
		navigate("/login");
	};

	if (environments.length <= 1) {
		return null;
	}

	return (
		<div className="min-h-screen bg-muted/30 flex items-center justify-center p-4">
			<Card className="w-full max-w-md shadow-lg">
				<CardHeader className="text-center space-y-2">
					<div className="flex justify-center mb-2">
						<div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center">
							<Building2 className="h-6 w-6 text-primary" />
						</div>
					</div>
					<CardTitle className="text-2xl font-bold">
						{textGet("login.environments.title")}
					</CardTitle>
					<CardDescription>
						{textGet("login.environments.subtitle")}
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="grid gap-3">
						{environments.map((env) => (
							<button
								type="button"
								key={env.ID}
								onClick={() => handleSelectEnvironment(env.ID)}
								className="flex items-center justify-between p-4 rounded-lg border bg-card hover:bg-accent hover:text-accent-foreground transition-all duration-200 text-left group hover:cursor-pointer"
							>
								<div className="flex items-center gap-4">
									<div className="h-10 w-10 rounded-md bg-primary/10 flex items-center justify-center shrink-0">
										<Building2 className="h-5 w-5 text-primary" />
									</div>
									<div>
										<h4 className="font-medium text-sm">
											{env.company.legal_name}
										</h4>
										{env.company.trade_name && (
											<p className="text-xs text-muted-foreground mt-0.5">
												{env.company.trade_name}
											</p>
										)}
									</div>
								</div>
								<ChevronRight className="h-5 w-5 text-muted-foreground group-hover:text-primary transition-colors" />
							</button>
						))}
					</div>

					<div className="pt-4 border-t mt-6 flex justify-center">
						<Button
							variant="ghost"
							className="text-muted-foreground hover:text-foreground"
							onClick={handleLogout}
						>
							<LogOut className="h-4 w-4 mr-2" />
							{textGet("login.environments.cancel")}
						</Button>
					</div>
				</CardContent>
			</Card>
		</div>
	);
}
