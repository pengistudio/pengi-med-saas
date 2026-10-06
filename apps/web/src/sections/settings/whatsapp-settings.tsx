import { useText } from "@pengi/shared";
import { Button, cn, Text } from "@pengi/ui";
import {
	AlertTriangle,
	ArrowRight,
	Loader2,
	MessageCircle,
	RefreshCw,
	Send,
	Unplug,
} from "lucide-react";
import React from "react";
import { Link } from "react-router";
import {
	disconnectWhatsApp,
	getWhatsAppAccount,
	getWhatsAppConfig,
	MAX_REMINDER_OFFSETS,
	REMINDER_OFFSET_CHOICES,
	syncWhatsAppTemplate,
	toggleReminderOffset,
	updateWhatsAppSettings,
	type WhatsAppAccount,
	type WhatsAppConfig,
} from "@/api/whatsapp-service";
import { ConfirmActionDialog } from "@/components/features/exam-orders/confirm-action-dialog";
import { ConnectManualDialog } from "@/components/features/whatsapp/connect-manual-dialog";
import { SendTestDialog } from "@/components/features/whatsapp/send-test-dialog";
import { Switch } from "@/components/ui/switch";
import { useEmbeddedSignup } from "@/hooks/use-embedded-signup";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import {
	StatusPill,
	WhatsAppTemplateList,
	WhatsAppUsageBar,
} from "./whatsapp-templates-usage";

// ─── Card ────────────────────────────────────────────────────────────────────

export function WhatsAppSettings() {
	const { textGet } = useText();
	const { checkPermission } = usePermission();
	const canUseInbox = checkPermission([
		PERMISSIONS.WHATSAPP.PERMISSION_USE_WHATSAPP_INBOX,
	]);
	const [config, setConfig] = React.useState<WhatsAppConfig | null>(null);
	const [account, setAccount] = React.useState<WhatsAppAccount | null>(null);
	const [manualOpen, setManualOpen] = React.useState(false);
	const [testOpen, setTestOpen] = React.useState(false);
	const [disconnectOpen, setDisconnectOpen] = React.useState(false);
	const [busy, setBusy] = React.useState<"sync" | "save" | "disconnect" | null>(
		null,
	);
	// Bumped after a sync: the template list and the usage re-read.
	const [reloadKey, setReloadKey] = React.useState(0);
	// Reminder settings being edited; reset whenever the account changes.
	const [enabled, setEnabled] = React.useState(false);
	const [offsets, setOffsets] = React.useState<number[]>([]);

	const applyAccount = React.useCallback((next: WhatsAppAccount) => {
		setAccount(next);
		setEnabled(next.reminders_enabled);
		setOffsets([...(next.reminder_offsets ?? [])].sort((a, b) => b - a));
	}, []);

	React.useEffect(() => {
		getWhatsAppConfig().then((res) => {
			if (res.success && res.data) setConfig(res.data);
		});
		getWhatsAppAccount().then((res) => {
			if (res.success && res.data) applyAccount(res.data);
		});
	}, [applyAccount]);

	const embedded = useEmbeddedSignup(config, applyAccount);

	async function handleSync() {
		setBusy("sync");
		const res = await syncWhatsAppTemplate();
		if (res.success && res.data) applyAccount(res.data);
		setReloadKey((k) => k + 1);
		setBusy(null);
	}

	async function handleSave() {
		setBusy("save");
		const res = await updateWhatsAppSettings({
			reminders_enabled: enabled,
			reminder_offsets: offsets,
		});
		if (res.success && res.data) applyAccount(res.data);
		setBusy(null);
	}

	async function handleDisconnect() {
		setBusy("disconnect");
		const res = await disconnectWhatsApp();
		if (res.success) {
			applyAccount({
				connected: false,
				template_status: "",
				reminders_enabled: false,
				reminder_offsets: [],
			});
		}
		setBusy(null);
	}

	const connected = !!account?.connected;
	const tokenInvalid = account?.status === "token_invalid";
	const savedOffsets = [...(account?.reminder_offsets ?? [])].sort(
		(a, b) => b - a,
	);
	const dirty =
		enabled !== !!account?.reminders_enabled ||
		offsets.join(",") !== savedOffsets.join(",");
	const offsetsMissing = enabled && offsets.length === 0;
	const templateStatus = account?.template_status ?? "";

	const connectActions = (
		<div className="flex flex-wrap items-center gap-2">
			{config?.embedded_signup_available && (
				<Button size="sm" disabled={embedded.running} onClick={embedded.start}>
					{embedded.running ? (
						<Loader2 className="mr-2 h-4 w-4 animate-spin" />
					) : (
						<MessageCircle className="mr-2 h-4 w-4" />
					)}
					<Text uuid="settings.whatsapp.connect" />
				</Button>
			)}
			<Button
				size="sm"
				variant={config?.embedded_signup_available ? "ghost" : "outline"}
				onClick={() => setManualOpen(true)}
			>
				<Text uuid="settings.whatsapp.connect_manual" />
			</Button>
		</div>
	);

	return (
		<div className="grid gap-4 border-t pt-6">
			<div className="flex flex-wrap items-start justify-between gap-4">
				<div className="grid gap-1">
					<p className="text-sm font-medium">
						<Text uuid="settings.whatsapp.title" />
					</p>
					<p className="text-sm text-muted-foreground">
						<Text uuid="settings.whatsapp.description" />
					</p>
					{account && (
						<span className="inline-flex w-fit items-center gap-1.5 text-xs font-medium text-muted-foreground">
							<span
								className={cn(
									"size-1.5 rounded-full",
									connected && !tokenInvalid
										? "bg-primary"
										: connected
											? "bg-amber-500"
											: "bg-muted-foreground/50",
								)}
							/>
							<Text
								uuid={
									!connected
										? "settings.whatsapp.status.disconnected"
										: tokenInvalid
											? "settings.whatsapp.status.token_invalid"
											: "settings.whatsapp.status.connected"
								}
							/>
						</span>
					)}
				</div>
				{connected ? (
					<Button
						variant="outline"
						size="sm"
						disabled={busy === "disconnect"}
						onClick={() => setDisconnectOpen(true)}
					>
						<Unplug className="mr-2 h-4 w-4" />
						<Text uuid="settings.whatsapp.disconnect" />
					</Button>
				) : (
					account && connectActions
				)}
			</div>

			{!connected && account && (
				<p className="rounded-lg bg-muted/50 p-4 text-sm text-muted-foreground">
					<Text
						uuid={
							config?.embedded_signup_available
								? "settings.whatsapp.hint.embedded"
								: "settings.whatsapp.hint.manual"
						}
					/>
				</p>
			)}

			{connected && account && (
				<>
					{tokenInvalid && (
						<div className="grid gap-3 rounded-lg border border-amber-500/40 bg-amber-500/10 p-4">
							<p className="flex items-start gap-1.5 text-sm text-amber-700 dark:text-amber-400">
								<AlertTriangle className="mt-0.5 size-4 shrink-0" />
								<Text uuid="settings.whatsapp.token_invalid.hint" />
							</p>
							{connectActions}
						</div>
					)}

					<dl className="grid gap-3 text-sm sm:grid-cols-2">
						<div className="grid gap-0.5">
							<dt className="text-xs text-muted-foreground">
								<Text uuid="settings.whatsapp.number" />
							</dt>
							<dd className="font-medium tabular-nums">
								{account.display_phone || account.phone_number_id}
							</dd>
							{account.verified_name && (
								<dd className="text-muted-foreground">
									{account.verified_name}
								</dd>
							)}
						</div>
						<div className="grid gap-0.5">
							<dt className="text-xs text-muted-foreground">
								<Text uuid="settings.whatsapp.mode" />
							</dt>
							<dd>
								<Text uuid={`settings.whatsapp.mode.${account.mode}`} />
							</dd>
						</div>
						{/* The catalog needs the inbox permission; without it, the reminder's status. */}
						{canUseInbox ? (
							<WhatsAppTemplateList
								reloadKey={reloadKey}
								syncing={busy === "sync"}
								onSync={handleSync}
							/>
						) : (
							<div className="grid gap-1 sm:col-span-2">
								<dt className="text-xs text-muted-foreground">
									<Text uuid="settings.whatsapp.template" />
								</dt>
								<dd className="flex flex-wrap items-center gap-2">
									<span className="font-mono text-xs">
										{account.template_name}
									</span>
									<StatusPill status={templateStatus} />
									<Button
										variant="ghost"
										size="sm"
										disabled={busy === "sync"}
										onClick={handleSync}
									>
										<RefreshCw
											className={cn(
												"mr-2 h-4 w-4",
												busy === "sync" && "animate-spin",
											)}
										/>
										<Text uuid="settings.whatsapp.template.sync" />
									</Button>
								</dd>
								{account.template_reason && (
									<dd className="text-xs text-muted-foreground">
										{account.template_reason}
									</dd>
								)}
								{templateStatus !== "APPROVED" && (
									<dd className="text-xs text-muted-foreground">
										<Text uuid="settings.whatsapp.template.not_approved_hint" />
									</dd>
								)}
							</div>
						)}
						<WhatsAppUsageBar reloadKey={reloadKey} />
					</dl>

					<div className="grid gap-3 rounded-lg bg-muted/50 p-4">
						<div className="flex items-center justify-between gap-4">
							<label
								htmlFor="whatsapp-reminders"
								className="cursor-pointer text-sm font-medium"
							>
								<Text uuid="settings.whatsapp.reminders" />
							</label>
							<Switch
								id="whatsapp-reminders"
								checked={enabled}
								onCheckedChange={(checked) => setEnabled(checked)}
							/>
						</div>
						<p className="text-xs text-muted-foreground">
							<Text uuid="settings.whatsapp.reminders.opt_in_hint" />
						</p>
						<div className="grid gap-2">
							<p className="text-xs text-muted-foreground">
								<Text
									uuid="settings.whatsapp.offsets.hint"
									values={{ max: MAX_REMINDER_OFFSETS }}
								/>
							</p>
							<div className="flex flex-wrap gap-2">
								{REMINDER_OFFSET_CHOICES.map((hours) => {
									const selected = offsets.includes(hours);
									const full =
										!selected && offsets.length >= MAX_REMINDER_OFFSETS;
									return (
										<button
											key={hours}
											type="button"
											aria-pressed={selected}
											disabled={!enabled || full}
											onClick={() =>
												setOffsets((prev) => toggleReminderOffset(prev, hours))
											}
											className={cn(
												"rounded-full border px-3 py-1 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50",
												selected
													? "border-primary bg-primary text-primary-foreground"
													: "bg-background hover:bg-muted",
											)}
										>
											{textGet("settings.whatsapp.offset_before", {
												count: hours,
											})}
										</button>
									);
								})}
							</div>
							{offsetsMissing && (
								<p className="text-xs text-destructive">
									<Text uuid="settings.whatsapp.offsets.required" />
								</p>
							)}
						</div>
						<div className="flex flex-wrap justify-end gap-2">
							<Button
								variant="outline"
								size="sm"
								disabled={templateStatus !== "APPROVED" || tokenInvalid}
								onClick={() => setTestOpen(true)}
							>
								<Send className="mr-2 h-4 w-4" />
								<Text uuid="settings.whatsapp.test.open" />
							</Button>
							<Button
								size="sm"
								disabled={!dirty || offsetsMissing || busy === "save"}
								onClick={handleSave}
							>
								{busy === "save" && (
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
								)}
								<Text uuid="form.save" />
							</Button>
						</div>
					</div>

					{canUseInbox && (
						<Link
							to="/whatsapp"
							className="inline-flex w-fit items-center gap-1 text-sm font-medium text-primary hover:underline"
						>
							<Text uuid="settings.whatsapp.inbox_link" />
							<ArrowRight className="h-4 w-4" />
						</Link>
					)}
				</>
			)}

			<ConnectManualDialog
				open={manualOpen}
				onOpenChange={setManualOpen}
				onConnected={applyAccount}
			/>
			<SendTestDialog open={testOpen} onOpenChange={setTestOpen} />
			<ConfirmActionDialog
				open={disconnectOpen}
				onOpenChange={setDisconnectOpen}
				title={<Text uuid="settings.whatsapp.disconnect.title" />}
				description={<Text uuid="settings.whatsapp.disconnect.description" />}
				onConfirm={handleDisconnect}
			/>
		</div>
	);
}
