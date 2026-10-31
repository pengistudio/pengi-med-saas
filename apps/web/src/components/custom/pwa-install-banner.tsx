import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Share, X } from "lucide-react";
import { useEffect, useState } from "react";

const DISMISSED_KEY = "pwa-install-dismissed";

// Chromium's install prompt event; not in lib.dom.
interface InstallPromptEvent extends Event {
	prompt: () => Promise<void>;
}

const isStandalone = () =>
	window.matchMedia("(display-mode: standalone)").matches ||
	(navigator as Navigator & { standalone?: boolean }).standalone === true;

// iPadOS reports a Mac user agent, so touch support tells them apart.
const isIos = () =>
	/iPhone|iPad|iPod/.test(navigator.userAgent) ||
	(navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);

const wasDismissed = () => {
	try {
		return localStorage.getItem(DISMISSED_KEY) === "1";
	} catch {
		return false;
	}
};

export function PwaInstallBanner() {
	const { textGet } = useText();
	const [prompt, setPrompt] = useState<InstallPromptEvent | null>(null);
	const [ios, setIos] = useState(false);
	const [hidden, setHidden] = useState(true);

	useEffect(() => {
		if (!import.meta.env.PROD || isStandalone() || wasDismissed()) return;
		// iOS has no install event: the only way is to explain Share → Add to Home Screen.
		if (isIos()) {
			setIos(true);
			setHidden(false);
		}
		const onPrompt = (e: Event) => {
			e.preventDefault();
			setPrompt(e as InstallPromptEvent);
			setHidden(false);
		};
		window.addEventListener("beforeinstallprompt", onPrompt);
		return () => window.removeEventListener("beforeinstallprompt", onPrompt);
	}, []);

	if (hidden) return null;

	const dismiss = () => {
		try {
			localStorage.setItem(DISMISSED_KEY, "1");
		} catch {
			// private mode: it just shows again next visit
		}
		setHidden(true);
	};

	const install = async () => {
		await prompt?.prompt();
		setHidden(true);
	};

	return (
		<div
			role="dialog"
			className="fixed inset-x-3 bottom-3 z-40 flex items-start gap-3 rounded-xl border bg-card p-3 text-sm shadow-lg md:left-auto md:max-w-sm"
			style={{ marginBottom: "env(safe-area-inset-bottom)" }}
		>
			<div className="flex-1 space-y-1">
				<p className="font-medium">{textGet("pwa.install.title")}</p>
				{ios && (
					<p className="flex items-center gap-1 text-muted-foreground">
						<Share className="size-4 shrink-0" />
						{textGet("pwa.install.ios_steps")}
					</p>
				)}
				{prompt && (
					<Button size="sm" onClick={install}>
						{textGet("pwa.install.action")}
					</Button>
				)}
			</div>
			<button
				type="button"
				onClick={dismiss}
				aria-label={textGet("pwa.install.dismiss")}
				className="text-muted-foreground"
			>
				<X className="size-4" />
			</button>
		</div>
	);
}
