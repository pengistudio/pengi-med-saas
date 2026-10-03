import { useText } from "@pengi/shared";
import { AlertTriangle, CheckCircle2, Loader2, XCircle } from "lucide-react";
import type { UploadItem } from "./upload-batch";

/** One row per file of a batch with its state: pending, converting, % , done... */
export function UploadProgressList({ items }: { items: UploadItem[] }) {
	const { textGet, formatDate } = useText();

	const detail = (item: UploadItem) => {
		switch (item.status) {
			case "pending":
				return textGet("clinical.attachment.upload.status.pending");
			case "converting":
				return textGet("clinical.attachment.upload.status.converting");
			case "uploading":
				return textGet("clinical.attachment.upload.status.uploading", {
					percent: item.progress,
				});
			case "done":
				return textGet("clinical.attachment.upload.status.done");
			case "duplicate":
				return textGet("clinical.attachment.upload.status.duplicate", {
					date: item.duplicate ? formatDate(item.duplicate.created_at) : "",
				});
			case "skipped":
				return textGet("clinical.attachment.upload.status.skipped");
			case "error":
				return item.errorKey ? textGet(item.errorKey) : item.errorMessage;
		}
	};

	return (
		<ul className="space-y-2" aria-live="polite">
			{items.map((item) => (
				<li key={item.id} className="flex items-start gap-2 text-sm">
					<span className="mt-0.5 shrink-0">
						{item.status === "done" && (
							<CheckCircle2 className="h-4 w-4 text-green-600" />
						)}
						{item.status === "duplicate" && (
							<AlertTriangle className="h-4 w-4 text-amber-600" />
						)}
						{(item.status === "error" || item.status === "skipped") && (
							<XCircle className="h-4 w-4 text-destructive" />
						)}
						{(item.status === "converting" || item.status === "uploading") && (
							<Loader2 className="h-4 w-4 animate-spin" />
						)}
					</span>
					<div className="min-w-0 flex-1">
						<p className="truncate font-medium">{item.name}</p>
						<p className="text-xs text-muted-foreground">{detail(item)}</p>
						{item.status === "uploading" && (
							<progress
								className="mt-1 h-1.5 w-full"
								max={100}
								value={item.progress}
							/>
						)}
					</div>
				</li>
			))}
		</ul>
	);
}
