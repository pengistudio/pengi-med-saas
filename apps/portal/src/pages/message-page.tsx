import { useText } from "@pengi/shared";

/** A centered title and sentence, for the pages that have nothing to do. */
export function MessagePage({
	titleKey,
	bodyKey,
}: {
	titleKey: string;
	bodyKey: string;
}) {
	const { textGet } = useText();
	return (
		<div className="flex min-h-screen items-center justify-center px-6">
			<div className="max-w-sm space-y-2 text-center">
				<h1 className="text-xl font-semibold">{textGet(titleKey)}</h1>
				<p className="text-muted-foreground text-sm">{textGet(bodyKey)}</p>
			</div>
		</div>
	);
}

export const LandingPage = () => (
	<MessagePage titleKey="portal.landing.title" bodyKey="portal.landing.body" />
);

export const UnavailablePage = () => (
	<MessagePage
		titleKey="portal.unavailable.title"
		bodyKey="portal.unavailable.body"
	/>
);
