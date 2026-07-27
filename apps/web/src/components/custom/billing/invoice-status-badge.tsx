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
				<FailedStatusBadge errorMessage={errorMessage} onRetry={onRetry} />
			);
		default:
			return (
				<Badge variant="outline">
					<Text uuid={`billing.status.${status}`} />
				</Badge>
			);
	}
}

function FailedStatusBadge({
	errorMessage,
	onRetry,
}: Pick<InvoiceStatusBadgeProps, "errorMessage" | "onRetry">) {
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
						<Badge variant="destructive" className="cursor-pointer">
							<Text uuid="billing.status.failed" />
						</Badge>
					}
				/>
				<PopoverContent>
					<PopoverTitle>
						<Text uuid="billing.status.failed.detail.title" />
					</PopoverTitle>
					<p className="text-muted-foreground whitespace-pre-wrap break-words">
						{errorMessage ?? textGet("billing.status.failed.detail.unknown")}
					</p>
				</PopoverContent>
			</Popover>
			{onRetry && (
				<Button
					size="icon-sm"
					variant="ghost"
					disabled={retrying}
					onClick={handleRetry}
					title={textGet("billing.status.failed.retry")}
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
