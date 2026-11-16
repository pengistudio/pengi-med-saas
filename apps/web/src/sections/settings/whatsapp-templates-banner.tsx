import { useText } from "@pengi/shared";
import { Button, cn, Text } from "@pengi/ui";
import { AlertTriangle, Clock, Loader2, RefreshCw } from "lucide-react";
import type {
	WhatsAppAccount,
	WhatsAppTemplatesState,
} from "@/api/whatsapp-service";

/** Statuses that are neither approved, in review nor missing. */
// Statuses with a label; any other Meta status (e.g. FLAGGED) shows "unknown".
function statusKey(status: string): string {
	switch (status) {
		case "APPROVED":
			return "settings.whatsapp.template.status.APPROVED";
		case "PENDING":
			return "settings.whatsapp.template.status.PENDING";
		case "REJECTED":
			return "settings.whatsapp.template.status.REJECTED";
		case "PAUSED":
			return "settings.whatsapp.template.status.PAUSED";
		case "DISABLED":
			return "settings.whatsapp.template.status.DISABLED";
		case "":
			return "settings.whatsapp.template.status.none";
		default:
			return "settings.whatsapp.template.status.unknown";
	}
}

const OK_OR_WAITING = new Set(["APPROVED", "PENDING", "IN_APPEAL", ""]);

/**
 * The account's templates summed up; falls back to the missing list when
 * the API didn't send templates_state.
 */
export function templatesState(
	account: WhatsAppAccount | null | undefined,
): WhatsAppTemplatesState {
	if (!account?.connected) return "";
	if (account.templates_state) return account.templates_state;
	return (account.missing_templates?.length ?? 0) > 0 ? "missing" : "";
}

/**
 * The next step while the clinic's templates aren't all approved by Meta:
 * create them (missing, only on click), wait for Meta's review (pending) or
 * see why Meta rejected or paused one (rejected). Nothing for "ready".
 */
export function WhatsAppTemplatesBanner({
	account,
	busy,
	onCreate,
	onSync,
}: {
	account: WhatsAppAccount;
	busy: "sync" | "create" | null;
	onCreate: () => void;
	onSync: () => void;
}) {
	const { textGet } = useText();
	const state = templatesState(account);
	if (state === "" || state === "ready") return null;

	const missing = account.missing_templates?.length ?? 0;
	const rejected = (account.templates ?? []).filter(
		(t) => !OK_OR_WAITING.has(t.status),
	);
	const warn = state !== "pending";

	const refresh = (
		<Button
			variant="outline"
			size="sm"
			disabled={busy !== null}
			onClick={onSync}
		>
			<RefreshCw
				className={cn("mr-2 h-4 w-4", busy === "sync" && "animate-spin")}
			/>
			{textGet("settings.whatsapp.templates_state.refresh")}
		</Button>
	);

	return (
		<div
			role="status"
			data-state={state}
			className={cn(
				"grid gap-3 rounded-lg border p-4",
				warn
					? "border-amber-500/40 bg-amber-500/10"
					: "border-primary/30 bg-primary/5",
			)}
		>
			<div className="grid gap-1">
				<p
					className={cn(
						"flex items-start gap-1.5 text-sm font-semibold",
						warn && "text-amber-700 dark:text-amber-400",
					)}
				>
					{warn ? (
						<AlertTriangle className="mt-0.5 size-4 shrink-0" />
					) : (
						<Clock className="mt-0.5 size-4 shrink-0" />
					)}
					<Text uuid={`settings.whatsapp.templates_state.${state}.title`} />
				</p>
				<p className="text-sm text-muted-foreground">
					<Text
						uuid={`settings.whatsapp.templates_state.${state}.description`}
					/>
				</p>
			</div>

			{state === "rejected" && rejected.length > 0 && (
				<ul className="grid gap-1 text-sm">
					{rejected.map((t) => (
						<li key={t.name} className="flex flex-wrap items-baseline gap-2">
							<span className="font-mono text-xs">{t.name}</span>
							<span className="text-xs font-medium">
								<Text uuid={statusKey(t.status)} />
							</span>
							{t.reason && (
								<span className="text-xs text-muted-foreground">
									{t.reason}
								</span>
							)}
						</li>
					))}
				</ul>
			)}

			<div className="flex flex-wrap items-center gap-2">
				{state === "missing" ? (
					<>
						{/* Only on an explicit click: it writes to Meta. */}
						<Button size="sm" disabled={busy !== null} onClick={onCreate}>
							{busy === "create" && (
								<Loader2 className="mr-2 h-4 w-4 shrink-0 animate-spin" />
							)}
							{textGet("settings.whatsapp.template.create_missing")}
						</Button>
						{missing > 0 && (
							<span className="text-sm text-muted-foreground">
								{textGet("settings.whatsapp.template.missing_count", {
									count: missing,
								})}
							</span>
						)}
					</>
				) : (
					refresh
				)}
			</div>
		</div>
	);
}
