import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@pengi/ui";
import { AlertTriangle } from "lucide-react";
import { isRouteErrorResponse, useRouteError } from "react-router";
import { useText } from "@/hooks/use-text";

// `String(error)` gives "[object Object]" for anything that isn't an Error
// instance (e.g. React Router's ErrorResponse for loader/route errors, or a
// plain thrown object) — this covers those shapes explicitly instead.
function formatErrorDetail(error: unknown): string {
	if (isRouteErrorResponse(error)) {
		return `${error.status} ${error.statusText}\n${JSON.stringify(error.data, null, 2)}`;
	}
	if (error instanceof Error) {
		return error.stack ?? error.message;
	}
	try {
		return JSON.stringify(error, null, 2);
	} catch {
		return String(error);
	}
}

export default function ErrorPage() {
	const error = useRouteError();
	const { textGet } = useText();
	const isNotFound = isRouteErrorResponse(error) && error.status === 404;

	console.error("Unhandled route error:", error);

	const title = isNotFound
		? textGet("error_page.not_found.title")
		: textGet("error_page.title");
	const description = isNotFound
		? textGet("error_page.not_found.description")
		: textGet("error_page.description");

	return (
		<div className="min-h-screen bg-muted/30 flex items-center justify-center p-4">
			<Card className="w-full max-w-md shadow-lg">
				<CardHeader className="text-center space-y-2">
					<div className="flex justify-center mb-2">
						<div className="h-12 w-12 rounded-full bg-destructive/10 flex items-center justify-center">
							<AlertTriangle className="h-6 w-6 text-destructive" />
						</div>
					</div>
					<CardTitle className="text-2xl font-bold">{title}</CardTitle>
					<CardDescription>{description}</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="flex flex-col gap-2 sm:flex-row">
						<Button className="flex-1" onClick={() => window.location.reload()}>
							{textGet("error_page.retry_button")}
						</Button>
						<Button
							variant="outline"
							className="flex-1"
							onClick={() => {
								window.location.href = "/";
							}}
						>
							{textGet("error_page.home_button")}
						</Button>
					</div>
					{import.meta.env.DEV && (
						<details className="rounded-md border bg-muted/50 p-3 text-xs">
							<summary className="cursor-pointer font-medium text-muted-foreground">
								Detalle técnico (solo dev)
							</summary>
							<pre className="mt-2 whitespace-pre-wrap break-words text-destructive">
								{formatErrorDetail(error)}
							</pre>
						</details>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
