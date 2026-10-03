import { useText } from "@pengi/shared";
import { cn } from "@pengi/ui";
import { AlertTriangle, HardDrive } from "lucide-react";
import {
	type AttachmentUsage,
	isStorageFull,
} from "@/api/patient-attachment-service";
import { Alert, AlertDescription } from "@/components/ui/alert";

/** Share of the quota in use, 0–100 (100 when the plan has no quota). */
export const storagePercent = (usage: AttachmentUsage) =>
	usage.quota_bytes > 0
		? Math.min(100, Math.floor((usage.used_bytes * 100) / usage.quota_bytes))
		: 100;

/**
 * The tenant's attachment storage against its plan quota: a bar with
 * "X de Y", a warning from 80% on and, once full, why uploads are disabled.
 */
export function AttachmentStorageUsage({ usage }: { usage: AttachmentUsage }) {
	const { textGet, formatFileSize } = useText();
	const percent = storagePercent(usage);
	const full = isStorageFull(usage);

	const message = full
		? textGet(
				usage.quota_bytes === 0
					? "clinical.attachment.usage.not_included"
					: "clinical.attachment.usage.full",
			)
		: usage.warning
			? textGet("clinical.attachment.usage.warning", { percent })
			: null;

	return (
		<div className="space-y-2">
			{usage.quota_bytes > 0 && (
				<div className="space-y-1">
					<div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
						<span className="flex items-center gap-1.5">
							<HardDrive className="h-3.5 w-3.5" />
							{textGet("clinical.attachment.usage.title")}
						</span>
						<span>
							{textGet("clinical.attachment.usage.amount", {
								used: formatFileSize(usage.used_bytes),
								quota: formatFileSize(usage.quota_bytes),
							})}
						</span>
					</div>
					<div
						role="progressbar"
						aria-label={textGet("clinical.attachment.usage.title")}
						aria-valuemin={0}
						aria-valuemax={100}
						aria-valuenow={percent}
						className="h-2 w-full overflow-hidden rounded-full bg-muted"
					>
						<div
							className={cn(
								"h-full rounded-full transition-all",
								full
									? "bg-destructive"
									: usage.warning
										? "bg-amber-500"
										: "bg-primary",
							)}
							style={{ width: `${percent}%` }}
						/>
					</div>
				</div>
			)}
			{message && (
				<Alert variant={full ? "destructive" : "default"}>
					<AlertTriangle />
					<AlertDescription
						className={cn(!full && "text-amber-700 dark:text-amber-400")}
					>
						{message}
					</AlertDescription>
				</Alert>
			)}
		</div>
	);
}
