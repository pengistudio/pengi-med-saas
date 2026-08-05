import {
	Badge,
	Button,
	Popover,
	PopoverContent,
	PopoverTitle,
	PopoverTrigger,
	Text,
} from "@pengi/ui";
import { Loader2, RefreshCw } from "lucide-react";
import { useState } from "react";
import { useText } from "@/hooks/use-text";

interface InvoiceStatusBadgeProps {
	status: string;
	errorMessage?: string | null;
	onRetry?: () => void | Promise<void>;
}

export function InvoiceStatusBadge({
	status,
	errorMessage,
	onRetry,
}: InvoiceStatusBadgeProps) {
	switch (status) {
		case "pending":
			return (
				<Badge className="bg-amber-500 hover:bg-amber-600 text-white">
					<Text uuid="billing.status.pending" />
				</Badge>
			);
		case "processing":
			return (
				<Badge className="bg-slate-500 hover:bg-slate-600 text-white">
					<Loader2 className="animate-spin" />
					<Text uuid="billing.status.processing" />
				</Badge>
			);
		case "signed":
		case "validated":
			return (
				<Badge className="bg-blue-500 hover:bg-blue-600 text-white">
					<Text uuid={`billing.status.${status}`} />
				</Badge>
			);
		case "authorized":
			return (
				<Badge className="bg-green-500 hover:bg-green-600 text-white">
					<Text uuid="billing.status.authorized" />
				</Badge>
			);
		case "failed":
			return (
				<RetryableStatusBadge
					badgeClassName="cursor-pointer"
					badgeVariant="destructive"
					statusKey="billing.status.failed"
					detailTitleKey="billing.status.failed.detail.title"
					detailUnknownKey="billing.status.failed.detail.unknown"
					errorMessage={errorMessage}
					onRetry={onRetry}
				/>
			);
		case "connection_error":
			return (
				<RetryableStatusBadge
					badgeClassName="cursor-pointer bg-amber-500 hover:bg-amber-600 text-white"
					statusKey="billing.status.connection_error"
					detailTitleKey="billing.status.connection_error.detail.title"
					detailUnknownKey="billing.status.connection_error.detail.unknown"
					errorMessage={errorMessage}
					onRetry={onRetry}
				/>
			);
		default:
			return (
				<Badge variant="outline">
					<Text uuid={`billing.status.${status}`} />
				</Badge>
			);
	}
}

interface RetryableStatusBadgeProps
	extends Pick<InvoiceStatusBadgeProps, "errorMessage" | "onRetry"> {
	badgeClassName: string;
	badgeVariant?: "destructive";
	statusKey: string;
	detailTitleKey: string;
	detailUnknownKey: string;
}

function RetryableStatusBadge({
	badgeClassName,
	badgeVariant,
	statusKey,
	detailTitleKey,
	detailUnknownKey,
	errorMessage,
	onRetry,
}: RetryableStatusBadgeProps) {
	const { textGet } = useText();
	const [retrying, setRetrying] = useState(false);

	async function handleRetry() {
		if (!onRetry) return;
		setRetrying(true);
		await onRetry();
		setRetrying(false);
	}

	return (
		<div className="flex items-center gap-1">
			<Popover>
				<PopoverTrigger
					render={
						<Badge variant={badgeVariant} className={badgeClassName}>
							<Text uuid={statusKey} />
						</Badge>
					}
				/>
				<PopoverContent>
					<PopoverTitle>
						<Text uuid={detailTitleKey} />
					</PopoverTitle>
					<p className="text-muted-foreground whitespace-pre-wrap break-words">
						{errorMessage ?? textGet(detailUnknownKey)}
					</p>
				</PopoverContent>
			</Popover>
			{onRetry && (
				<Button
					size="icon-sm"
					variant="ghost"
					disabled={retrying}
					onClick={handleRetry}
					title={textGet("billing.status.retry")}
				>
					{retrying ? (
						<Loader2 className="h-3.5 w-3.5 animate-spin" />
					) : (
						<RefreshCw className="h-3.5 w-3.5" />
					)}
				</Button>
			)}
		</div>
	);
}
