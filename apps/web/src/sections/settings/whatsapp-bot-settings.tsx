import { useText } from "@pengi/shared";
import { Button, cn, Input, Text, Textarea } from "@pengi/ui";
import {
	ArrowRight,
	Check,
	FlaskConical,
	Loader2,
	Plus,
	Trash2,
	UserRound,
	X,
} from "lucide-react";
import React, { type ReactNode } from "react";
import { Link } from "react-router";
import {
	type WhatsAppBotSettings as BotSettings,
	getWhatsAppBotSettings,
	testWhatsAppBot,
	updateWhatsAppBotSettings,
	type WhatsAppBotFAQEntry,
	type WhatsAppBotPreview,
} from "@/api/whatsapp-service";
import { AI_CARD_CLASS, AIMark } from "@/components/features/ai/ai-mark";
import { BotReason } from "@/components/features/whatsapp/bot-reason";
import { Switch } from "@/components/ui/switch";
import { useEnabledFeatures } from "@/hooks/use-enabled-features";
import { aiBudgetNotice } from "@/lib/ai-budget";
import {
	BOT_READINESS_ITEMS,
	botReady,
	cleanFAQ,
	faqDraftValid,
	readinessWithDraft,
} from "@/lib/whatsapp-bot";

/** Checklist labels, literal so the message catalog test checks them. */
const READINESS_LABELS: Record<
	(typeof BOT_READINESS_ITEMS)[number]["key"],
	ReactNode
> = {
	whatsapp_connected: (
		<Text uuid="settings.whatsapp.bot.readiness.whatsapp_connected" />
	),
	ai_opt_in: <Text uuid="settings.whatsapp.bot.readiness.ai_opt_in" />,
	provider_configured: (
		<Text uuid="settings.whatsapp.bot.readiness.provider_configured" />
	),
	office_hours: <Text uuid="settings.whatsapp.bot.readiness.office_hours" />,
	address: <Text uuid="settings.whatsapp.bot.readiness.address" />,
	faq: <Text uuid="settings.whatsapp.bot.readiness.faq" />,
};

/** Longest sample message the test panel sends (the server's limit). */
const MAX_TEST_MESSAGE = 1000;

/**
 * The WhatsApp AI assistant (inside the WhatsApp settings card): readiness
 * checklist, switches, the FAQ it answers from and the "Probar" panel.
 * Hidden when the plan has no AI.
 */
export function WhatsAppBotSettings() {
	const planHasAI = useEnabledFeatures().ai === true;
	if (!planHasAI) return null;
	return <BotSettingsCard />;
}

function BotSettingsCard() {
	const { textGet, formatDate } = useText();
	const [settings, setSettings] = React.useState<BotSettings | null>(null);
	const [enabled, setEnabled] = React.useState(false);
	const [duringHours, setDuringHours] = React.useState(false);
	const [faq, setFaq] = React.useState<WhatsAppBotFAQEntry[]>([]);
	const [saving, setSaving] = React.useState(false);

	const apply = React.useCallback((next: BotSettings) => {
		setSettings(next);
		setEnabled(next.enabled);
		setDuringHours(next.during_office_hours);
		setFaq(next.faq.map((e) => ({ ...e })));
	}, []);

	React.useEffect(() => {
		getWhatsAppBotSettings().then((res) => {
			if (res.success && res.data) apply(res.data);
		});
	}, [apply]);

	if (!settings?.readiness.plan_whatsapp || !settings.readiness.plan_ai)
		return null;

	const limits = settings.limits;
	const readiness = readinessWithDraft(settings.readiness, faq);
	const ready = botReady(readiness);
	const faqValid = faqDraftValid(faq, limits);
	const dirty =
		enabled !== settings.enabled ||
		duringHours !== settings.during_office_hours ||
		JSON.stringify(cleanFAQ(faq)) !== JSON.stringify(settings.faq);
	// Turning it off is always possible; on needs every requirement.
	const canToggle = enabled || ready;

	function updateEntry(index: number, patch: Partial<WhatsAppBotFAQEntry>) {
		setFaq((prev) =>
			prev.map((e, i) => (i === index ? { ...e, ...patch } : e)),
		);
	}

	async function handleSave() {
		setSaving(true);
		const res = await updateWhatsAppBotSettings({
			enabled: enabled && ready,
			during_office_hours: duringHours,
			faq: cleanFAQ(faq),
		});
		if (res.success && res.data) apply(res.data);
		setSaving(false);
	}

	return (
		<section
			className={cn(AI_CARD_CLASS, "grid gap-4 p-4")}
			aria-labelledby="whatsapp-bot-title"
		>
			<div className="flex items-start gap-3">
				<AIMark />
				<div className="grid gap-1">
					<p id="whatsapp-bot-title" className="text-sm font-medium">
						<Text uuid="settings.whatsapp.bot.title" />
					</p>
					<p className="text-sm text-muted-foreground">
						<Text uuid="settings.whatsapp.bot.description" />
					</p>
				</div>
			</div>

			{!ready && (
				<div className="grid gap-2 rounded-lg bg-muted/50 p-3">
					<p className="text-xs font-medium">
						<Text uuid="settings.whatsapp.bot.readiness.title" />
					</p>
					<ul className="grid gap-1.5">
						{BOT_READINESS_ITEMS.map(({ key, to }) => {
							const ok = readiness[key];
							return (
								<li
									key={key}
									data-testid={`bot-readiness-${key}`}
									data-ok={ok ? "true" : "false"}
									className="flex items-start gap-2 text-xs"
								>
									{ok ? (
										<Check className="mt-0.5 size-3.5 shrink-0 text-emerald-600" />
									) : (
										<X className="mt-0.5 size-3.5 shrink-0 text-destructive" />
									)}
									<span className={cn(ok && "text-muted-foreground")}>
										{READINESS_LABELS[key]}
										{!ok && to && (
											<Link
												to={to}
												className="ml-1 inline-flex items-center gap-0.5 font-medium text-violet-700 hover:underline dark:text-violet-300"
											>
												<Text uuid="settings.whatsapp.bot.readiness.fix" />
												<ArrowRight className="size-3" />
											</Link>
										)}
									</span>
								</li>
							);
						})}
					</ul>
				</div>
			)}

			<div className="grid gap-3">
				<div className="flex items-center justify-between gap-4">
					<label
						htmlFor="whatsapp-bot-enabled"
						className={cn(
							"text-sm font-medium",
							canToggle ? "cursor-pointer" : "text-muted-foreground",
						)}
					>
						<Text uuid="settings.whatsapp.bot.enabled" />
					</label>
					<Switch
						id="whatsapp-bot-enabled"
						checked={enabled}
						disabled={!canToggle}
						onCheckedChange={(checked) => setEnabled(checked)}
					/>
				</div>
				<p className="text-xs text-muted-foreground">
					<Text uuid="settings.whatsapp.bot.enabled.hint" />
				</p>
				<div className="flex items-center justify-between gap-4">
					<label
						htmlFor="whatsapp-bot-hours"
						className="cursor-pointer text-sm font-medium"
					>
						<Text uuid="settings.whatsapp.bot.during_hours" />
					</label>
					<Switch
						id="whatsapp-bot-hours"
						checked={duringHours}
						onCheckedChange={(checked) => setDuringHours(checked)}
					/>
				</div>
				<p className="text-xs text-muted-foreground">
					<Text uuid="settings.whatsapp.bot.during_hours.hint" />
				</p>
				{aiBudgetNotice(settings.budget) && (
					<p
						className={cn(
							"text-xs",
							settings.budget.reached
								? "text-destructive"
								: "text-muted-foreground",
						)}
					>
						{settings.budget.reached ? (
							<Text
								uuid="ai.budget.reached"
								values={{
									percent: settings.budget.percent,
									date: formatDate(settings.budget.period_end, "day-month"),
								}}
							/>
						) : (
							<Text
								uuid="ai.budget.low"
								values={{ percent: settings.budget.percent }}
							/>
						)}
					</p>
				)}
			</div>

			<dl className="grid gap-3 border-t pt-3 text-sm sm:grid-cols-2">
				<div className="grid gap-0.5">
					<dt className="text-xs text-muted-foreground">
						<Text uuid="settings.whatsapp.bot.material.address" />
					</dt>
					<dd>
						{settings.address || (
							<span className="text-muted-foreground">
								<Text uuid="settings.whatsapp.bot.material.empty" />
							</span>
						)}
					</dd>
					{settings.clinic_phone && (
						<>
							<dt className="mt-2 text-xs text-muted-foreground">
								<Text uuid="settings.whatsapp.bot.material.phone" />
							</dt>
							<dd className="tabular-nums">{settings.clinic_phone}</dd>
						</>
					)}
					<dd className="mt-2 text-xs text-muted-foreground">
						{settings.booking_link ? (
							<Text uuid="settings.whatsapp.bot.material.booking_on" />
						) : (
							<Text uuid="settings.whatsapp.bot.material.booking_off" />
						)}
					</dd>
				</div>
				<div className="grid gap-0.5">
					<dt className="text-xs text-muted-foreground">
						<Text uuid="settings.whatsapp.bot.material.hours" />
					</dt>
					<dd className="whitespace-pre-line text-xs tabular-nums">
						{settings.office_hours || (
							<span className="text-muted-foreground">
								<Text uuid="settings.whatsapp.bot.material.empty" />
							</span>
						)}
					</dd>
				</div>
			</dl>

			<div className="grid gap-3 border-t pt-3">
				<div className="grid gap-1">
					<p className="text-sm font-medium">
						<Text uuid="settings.whatsapp.bot.faq.title" />
					</p>
					<p className="text-xs text-muted-foreground">
						<Text uuid="settings.whatsapp.bot.faq.hint" />
					</p>
				</div>
				{faq.length === 0 && (
					<p className="rounded-lg bg-muted/50 p-3 text-xs text-muted-foreground">
						<Text uuid="settings.whatsapp.bot.faq.empty" />
					</p>
				)}
				<ol className="grid gap-3">
					{faq.map((entry, index) => (
						<FAQEntryEditor
							key={index}
							index={index}
							entry={entry}
							maxQuestion={limits.max_question}
							maxAnswer={limits.max_answer}
							onChange={(patch) => updateEntry(index, patch)}
							onRemove={() =>
								setFaq((prev) => prev.filter((_, i) => i !== index))
							}
						/>
					))}
				</ol>
				<div className="flex flex-wrap items-center justify-between gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={faq.length >= limits.max_entries}
						onClick={() =>
							setFaq((prev) => [...prev, { question: "", answer: "" }])
						}
					>
						<Plus className="mr-2 h-4 w-4" />
						<Text uuid="settings.whatsapp.bot.faq.add" />
					</Button>
					<span className="text-xs text-muted-foreground tabular-nums">
						{textGet("settings.whatsapp.bot.faq.count", {
							total: faq.length,
							max: limits.max_entries,
						})}
					</span>
				</div>
				{!faqValid && (
					<p className="text-xs text-destructive">
						<Text uuid="settings.whatsapp.bot.faq.invalid" />
					</p>
				)}
				<div className="flex justify-end">
					<Button
						size="sm"
						disabled={!dirty || !faqValid || saving}
						onClick={handleSave}
					>
						{saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
						<Text uuid="form.save" />
					</Button>
				</div>
			</div>

			<BotTestPanel />
		</section>
	);
}

function Counter({ value, max }: { value: number; max: number }) {
	return (
		<span
			className={cn(
				"text-[11px] tabular-nums",
				value > max ? "text-destructive" : "text-muted-foreground",
			)}
		>
			{value}/{max}
		</span>
	);
}

function FAQEntryEditor({
	index,
	entry,
	maxQuestion,
	maxAnswer,
	onChange,
	onRemove,
}: {
	index: number;
	entry: WhatsAppBotFAQEntry;
	maxQuestion: number;
	maxAnswer: number;
	onChange: (patch: Partial<WhatsAppBotFAQEntry>) => void;
	onRemove: () => void;
}) {
	const { textGet } = useText();
	const qId = `bot-faq-q-${index}`;
	const aId = `bot-faq-a-${index}`;
	return (
		<li className="grid gap-2 rounded-lg border p-3">
			<div className="grid gap-1">
				<div className="flex items-center justify-between gap-2">
					<label htmlFor={qId} className="text-xs font-medium">
						<Text uuid="settings.whatsapp.bot.faq.question" />
					</label>
					<Counter value={entry.question.length} max={maxQuestion} />
				</div>
				<Input
					id={qId}
					value={entry.question}
					maxLength={maxQuestion}
					placeholder={textGet(
						"settings.whatsapp.bot.faq.question.placeholder",
					)}
					onChange={(e) => onChange({ question: e.target.value })}
				/>
			</div>
			<div className="grid gap-1">
				<div className="flex items-center justify-between gap-2">
					<label htmlFor={aId} className="text-xs font-medium">
						<Text uuid="settings.whatsapp.bot.faq.answer" />
					</label>
					<Counter value={entry.answer.length} max={maxAnswer} />
				</div>
				<Textarea
					id={aId}
					value={entry.answer}
					maxLength={maxAnswer}
					rows={3}
					placeholder={textGet("settings.whatsapp.bot.faq.answer.placeholder")}
					onChange={(e) => onChange({ answer: e.target.value })}
				/>
			</div>
			<div className="flex justify-end">
				<Button
					variant="ghost"
					size="sm"
					onClick={onRemove}
					aria-label={textGet("settings.whatsapp.bot.faq.remove")}
				>
					<Trash2 className="mr-2 h-4 w-4" />
					<Text uuid="settings.whatsapp.bot.faq.remove" />
				</Button>
			</div>
		</li>
	);
}

/** "Probar": what the assistant would answer; nothing is sent or counted. */
function BotTestPanel() {
	const { textGet } = useText();
	const [message, setMessage] = React.useState("");
	const [running, setRunning] = React.useState(false);
	const [preview, setPreview] = React.useState<WhatsAppBotPreview | null>(null);

	async function handleTest() {
		setRunning(true);
		const res = await testWhatsAppBot(message.trim());
		setPreview(res.success && res.data ? res.data : null);
		setRunning(false);
	}

	const answered = preview?.action === "answer";
	return (
		<div className="grid gap-3 border-t pt-3">
			<div className="grid gap-1">
				<label htmlFor="whatsapp-bot-test" className="text-sm font-medium">
					<Text uuid="settings.whatsapp.bot.test.title" />
				</label>
				<p className="text-xs text-muted-foreground">
					<Text uuid="settings.whatsapp.bot.test.hint" />
				</p>
			</div>
			<Textarea
				id="whatsapp-bot-test"
				value={message}
				maxLength={MAX_TEST_MESSAGE}
				rows={2}
				placeholder={textGet("settings.whatsapp.bot.test.placeholder")}
				onChange={(e) => setMessage(e.target.value)}
			/>
			<div className="flex justify-end">
				<Button
					variant="outline"
					size="sm"
					disabled={message.trim() === "" || running}
					onClick={handleTest}
				>
					{running ? (
						<Loader2 className="mr-2 h-4 w-4 animate-spin" />
					) : (
						<FlaskConical className="mr-2 h-4 w-4" />
					)}
					<Text uuid="settings.whatsapp.bot.test.run" />
				</Button>
			</div>
			{preview && (
				<div className="grid gap-2" data-testid="bot-test-result">
					<span
						className={cn(
							"inline-flex w-fit items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
							answered
								? "bg-violet-100 text-violet-700 dark:bg-violet-950 dark:text-violet-300"
								: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
						)}
					>
						{answered ? (
							<Check className="size-3" />
						) : (
							<UserRound className="size-3" />
						)}
						{answered ? (
							<Text uuid="settings.whatsapp.bot.test.answer" />
						) : (
							<Text uuid="settings.whatsapp.bot.test.handoff" />
						)}
					</span>
					{!answered && preview.reason && (
						<p className="text-xs text-muted-foreground">
							<BotReason reason={preview.reason} />
						</p>
					)}
					<p className="whitespace-pre-wrap break-words rounded-2xl rounded-br-sm bg-emerald-100 px-3 py-2 text-sm text-emerald-950 dark:bg-emerald-900/50 dark:text-emerald-50">
						{preview.message}
					</p>
					{!preview.would_reply_now && (
						<p className="text-xs text-muted-foreground">
							<Text uuid="settings.whatsapp.bot.test.in_hours" />
						</p>
					)}
					<p className="text-xs text-muted-foreground">
						<Text uuid="settings.whatsapp.bot.test.nothing_sent" />
					</p>
				</div>
			)}
		</div>
	);
}
