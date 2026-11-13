import { Wifi, WifiOff } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useUiText } from "../context/text-context";
import { useOnlineStatus } from "../hooks/use-online-status";
import { cn } from "../lib/utils";

const RESTORED_MS = 3000;

/**
 * Slim fixed bar shown while the browser is offline, and briefly after the
 * connection comes back. Mount once near the root, inside the text provider.
 */
export function OfflineBanner() {
	const { textGet } = useUiText();
	const online = useOnlineStatus();
	const [restored, setRestored] = useState(false);
	const wasOffline = useRef(false);

	useEffect(() => {
		if (!online) {
			wasOffline.current = true;
			setRestored(false);
			return;
		}
		if (!wasOffline.current) return;
		wasOffline.current = false;
		setRestored(true);
		const timer = setTimeout(() => setRestored(false), RESTORED_MS);
		return () => clearTimeout(timer);
	}, [online]);

	if (online && !restored) return null;

	const Icon = online ? Wifi : WifiOff;
	return (
		<div
			role="status"
			aria-live="polite"
			className={cn(
				"fixed inset-x-0 top-0 z-[100] flex items-center justify-center gap-2 px-4 py-1.5 text-sm font-medium",
				online
					? "bg-emerald-600 text-white"
					: "bg-destructive text-destructive-foreground",
			)}
		>
			<Icon className="size-4 shrink-0" aria-hidden="true" />
			<span>
				{textGet(online ? "offline.banner.restored" : "offline.banner.title")}
			</span>
		</div>
	);
}
