import { useText } from "@pengi/shared";
import { Spinner } from "@pengi/ui";

/** The whole page while the session or its first data loads. */
export function PageSpinner() {
	const { textGet } = useText();
	return (
		<div className="flex min-h-screen items-center justify-center">
			<Spinner aria-label={textGet("portal.loading")} />
		</div>
	);
}
