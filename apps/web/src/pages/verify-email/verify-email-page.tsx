import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import { verifyEmail } from "@/api/auth-service";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { Text } from "@/components/ui/text";

type State = "loading" | "success" | "error";

const VerifyEmailPage = () => {
	const [state, setState] = React.useState<State>("loading");
	const [searchParams] = useSearchParams();
	const navigate = useNavigate();
	const token = searchParams.get("token") ?? "";

	React.useEffect(() => {
		if (!token) {
			setState("error");
			return;
		}
		verifyEmail(token).then((res) => {
			setState(res.success ? "success" : "error");
		});
	}, [token]);

	return (
		<div className="flex min-h-screen items-center justify-center p-4">
			<Card className="max-w-md w-full">
				{state === "loading" && (
					<CardHeader className="items-center">
						<Spinner />
						<CardTitle className="mt-4">
							<Text uuid="verify_email.loading" />
						</CardTitle>
					</CardHeader>
				)}
				{state === "success" && (
					<>
						<CardHeader>
							<CardTitle>
								<Text uuid="verify_email.success.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="verify_email.success.description" />
							</CardDescription>
						</CardHeader>
						<CardFooter>
							<Button className="w-full" onClick={() => navigate("/login")}>
								<Text uuid="verify_email.go_to_login" />
							</Button>
						</CardFooter>
					</>
				)}
				{state === "error" && (
					<>
						<CardHeader>
							<CardTitle>
								<Text uuid="verify_email.error.title" />
							</CardTitle>
							<CardDescription>
								<Text uuid="verify_email.error.description" />
							</CardDescription>
						</CardHeader>
						<CardFooter>
							<Button
								variant="outline"
								className="w-full"
								onClick={() => navigate("/register")}
							>
								<Text uuid="register.go_to_login" />
							</Button>
						</CardFooter>
					</>
				)}
			</Card>
		</div>
	);
};

export default VerifyEmailPage;
