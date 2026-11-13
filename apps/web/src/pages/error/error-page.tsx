import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Illustration,
} from "@pengi/ui";
import { isRouteErrorResponse, useRouteError } from "react-router";

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

// A lazy chunk that fails to load (flaky network, or a deploy replaced it) or
// a failed fetch: reloading is the fix, so say "connection" rather than "bug".
const NETWORK_ERROR =
	/dynamically imported module|Failed to fetch|NetworkError|Load failed|importing a module script/i;

function isNetworkError(error: unknown): boolean {
	return error instanceof Error && NETWORK_ERROR.test(error.message);
}

export default function ErrorPage() {
	const error = useRouteError();
	const { textGet } = useText();
	const isNotFound = isRouteErrorResponse(error) && error.status === 404;
	const isMaintenance =
		isRouteErrorResponse(error) && [502, 503, 504].includes(error.status);
	const isNetwork = !isNotFound && !isMaintenance && isNetworkError(error);

	console.error("Unhandled route error:", error);

	const title = isNotFound
		? textGet("error_page.not_found.title")
		: isMaintenance
			? textGet("error_page.maintenance.title")
			: isNetwork
				? textGet("error_page.network.title")
				: textGet("error_page.title");
	const description = isNotFound
		? textGet("error_page.not_found.description")
		: isMaintenance
			? textGet("error_page.maintenance.description")
			: isNetwork
				? textGet("error_page.network.description")
				: textGet("error_page.description");

	return (
		<div className="min-h-screen bg-muted/30 flex items-center justify-center p-4">
			<Card className="w-full max-w-md shadow-lg">
				<CardHeader className="text-center space-y-2">
					<div className="flex justify-center mb-2">
						<Illustration
							name={
								isNotFound
									? "not-found"
									: isMaintenance
										? "maintenance"
										: isNetwork
											? "no-connection"
											: "error"
							}
						/>
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
