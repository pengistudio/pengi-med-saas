import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Input,
	Spinner,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@pengi/ui";
import React from "react";
import {
	type AIFunction,
	type AIFunctionSettings,
	type AIUsageReport,
	getAISettings,
	getAIUsage,
	updateAISettings,
} from "@/api/ai-service";

/** Server settings plus the write-only key being typed (or cleared). */
type Draft = AIFunctionSettings & { api_key: string; clear_api_key: boolean };

const toDraft = (s: AIFunctionSettings): Draft => ({
	...s,
	api_key: "",
	clear_api_key: false,
});

function Field({
	id,
	label,
	hint,
	children,
}: React.PropsWithChildren<{ id: string; label: string; hint?: string }>) {
	return (
		<div className="grid gap-1.5">
			<label htmlFor={id} className="text-sm font-medium">
				{label}
			</label>
			{children}
			{hint && <p className="text-xs text-muted-foreground">{hint}</p>}
		</div>
	);
}

/** Current month as YYYY-MM (local time; the server reads it in its zone). */
function currentMonth(): string {
	const now = new Date();
	return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
}

/**
 * `/ai`: provider (base URL, write-only key, model, reasoning effort) and
 * prices per AI function, empty = server env default; and the month's usage
 * per clinic with tokens and cost (clinics only see their use count).
 */
const AIPage = () => {
	const { textGet, formatMoney } = useText();
	const [settings, setSettings] = React.useState<Draft[]>([]);
	const [saving, setSaving] = React.useState(false);
	const [month, setMonth] = React.useState(currentMonth);
	const [usage, setUsage] = React.useState<AIUsageReport | null>(null);
	const [usageLoading, setUsageLoading] = React.useState(true);

	React.useEffect(() => {
		getAISettings().then((res) => {
			if (res.success) setSettings(res.data.map(toDraft));
		});
	}, []);

	React.useEffect(() => {
		let cancelled = false;
		getAIUsage(month).then((res) => {
			if (cancelled) return;
			setUsage(res.success ? res.data : null);
			setUsageLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [month]);

	function update(index: number, patch: Partial<Draft>) {
		setSettings((prev) =>
			prev.map((s, i) => (i === index ? { ...s, ...patch } : s)),
		);
	}

	async function handleSave() {
		setSaving(true);
		const res = await updateAISettings(
			settings.map((s) => ({
				function: s.function,
				base_url: s.base_url,
				model: s.model,
				reasoning_effort: s.reasoning_effort,
				api_key: s.api_key,
				clear_api_key: s.clear_api_key,
				price_input: s.price_input,
				price_output: s.price_output,
			})),
		);
		setSaving(false);
		// The typed key is dropped either way: it is never shown again.
		if (res.success) setSettings(res.data.map(toDraft));
	}

	// Literal keys, so the message-keys test checks them.
	const keyHint = (s: Draft) => {
		if (s.clear_api_key) return textGet("backoffice.ai.api_key.will_clear");
		if (s.api_key_source === "db")
			return textGet("backoffice.ai.api_key.source.db");
		if (s.api_key_source === "env")
			return textGet("backoffice.ai.api_key.source.env");
		return s.env_has_api_key
			? textGet("backoffice.ai.api_key.source.env_other_host")
			: textGet("backoffice.ai.api_key.source.none");
	};

	const totalCost = usage?.rows.reduce((sum, r) => sum + r.cost_usd, 0) ?? 0;

	// Literal keys, so the message-keys test checks them.
	const functionLabel = (fn: AIFunction) =>
		fn === "summary"
			? textGet("backoffice.ai.function.summary")
			: fn === "scribe"
				? textGet("backoffice.ai.function.scribe")
				: textGet("backoffice.ai.function.bot");

	return (
		<div className="grid gap-6">
			<Card>
				<CardHeader>
					<CardTitle>{textGet("backoffice.ai.settings.title")}</CardTitle>
					<CardDescription>
						{textGet("backoffice.ai.settings.description")}
					</CardDescription>
				</CardHeader>
				<CardContent className="grid gap-4">
					{settings.some((s) => !s.encryption_available) && (
						<p className="rounded-md bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400">
							{textGet("backoffice.ai.encryption_missing")}
						</p>
					)}
					{settings.map((s, i) => (
						<fieldset
							key={s.function}
							className="grid gap-3 rounded-md border p-4 sm:grid-cols-2"
						>
							<legend className="px-1 text-sm font-medium">
								{functionLabel(s.function)}
							</legend>
							<Field
								id={`ai-${s.function}-base-url`}
								label={textGet("backoffice.ai.base_url")}
								hint={textGet("backoffice.ai.default_hint", {
									value: s.default_base_url,
								})}
							>
								<Input
									id={`ai-${s.function}-base-url`}
									type="url"
									value={s.base_url}
									placeholder={s.default_base_url}
									onChange={(e) => update(i, { base_url: e.target.value })}
								/>
							</Field>
							<Field
								id={`ai-${s.function}-api-key`}
								label={textGet("backoffice.ai.api_key")}
								hint={keyHint(s)}
							>
								<div className="flex gap-2">
									<Input
										id={`ai-${s.function}-api-key`}
										type="password"
										autoComplete="new-password"
										value={s.api_key}
										disabled={!s.encryption_available || s.clear_api_key}
										placeholder={
											s.has_api_key
												? textGet("backoffice.ai.api_key.placeholder_keep")
												: textGet("backoffice.ai.api_key.placeholder_new")
										}
										onChange={(e) => update(i, { api_key: e.target.value })}
									/>
									{s.has_api_key && (
										<Button
											type="button"
											variant="outline"
											onClick={() =>
												update(i, {
													clear_api_key: !s.clear_api_key,
													api_key: "",
												})
											}
										>
											{s.clear_api_key
												? textGet("backoffice.ai.api_key.undo")
												: textGet("backoffice.ai.api_key.clear")}
										</Button>
									)}
								</div>
							</Field>
							<Field
								id={`ai-${s.function}-model`}
								label={textGet("backoffice.ai.model")}
								hint={textGet("backoffice.ai.default_hint", {
									value: s.default_model,
								})}
							>
								<Input
									id={`ai-${s.function}-model`}
									value={s.model}
									placeholder={s.default_model}
									onChange={(e) => update(i, { model: e.target.value })}
								/>
							</Field>
							<Field
								id={`ai-${s.function}-effort`}
								label={textGet("backoffice.ai.reasoning_effort")}
								hint={
									s.default_reasoning_effort
										? textGet("backoffice.ai.default_hint", {
												value: s.default_reasoning_effort,
											})
										: textGet("backoffice.ai.reasoning_effort.hint")
								}
							>
								<Input
									id={`ai-${s.function}-effort`}
									value={s.reasoning_effort}
									placeholder={s.default_reasoning_effort}
									onChange={(e) =>
										update(i, { reasoning_effort: e.target.value })
									}
								/>
							</Field>
							<Field
								id={`ai-${s.function}-price-in`}
								label={textGet("backoffice.ai.price_input")}
							>
								<Input
									id={`ai-${s.function}-price-in`}
									type="number"
									min={0}
									step="any"
									value={s.price_input}
									onChange={(e) =>
										update(i, {
											price_input: Math.max(0, Number(e.target.value) || 0),
										})
									}
								/>
							</Field>
							<Field
								id={`ai-${s.function}-price-out`}
								label={textGet("backoffice.ai.price_output")}
							>
								<Input
									id={`ai-${s.function}-price-out`}
									type="number"
									min={0}
									step="any"
									value={s.price_output}
									onChange={(e) =>
										update(i, {
											price_output: Math.max(0, Number(e.target.value) || 0),
										})
									}
								/>
							</Field>
						</fieldset>
					))}
					<div className="flex justify-end">
						<Button
							onClick={handleSave}
							disabled={saving || settings.length === 0}
						>
							{saving && <Spinner />}
							{textGet("backoffice.ai.settings.save")}
						</Button>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader className="flex flex-row flex-wrap items-start justify-between gap-4">
					<div className="grid gap-1.5">
						<CardTitle>{textGet("backoffice.ai.usage.title")}</CardTitle>
						<CardDescription>
							{textGet("backoffice.ai.usage.description")}
						</CardDescription>
					</div>
					<Input
						type="month"
						className="w-44"
						aria-label={textGet("backoffice.ai.usage.month")}
						value={month}
						onChange={(e) => {
							if (!e.target.value) return;
							setUsageLoading(true);
							setMonth(e.target.value);
						}}
					/>
				</CardHeader>
				<CardContent>
					{usageLoading ? (
						<div className="flex justify-center py-8">
							<Spinner />
						</div>
					) : !usage || usage.rows.length === 0 ? (
						<p className="py-8 text-center text-sm text-muted-foreground">
							{textGet("backoffice.ai.usage.empty")}
						</p>
					) : (
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>{textGet("backoffice.ai.usage.clinic")}</TableHead>
									<TableHead>{textGet("backoffice.ai.function")}</TableHead>
									<TableHead className="text-right">
										{textGet("backoffice.ai.usage.uses")}
									</TableHead>
									<TableHead className="text-right">
										{textGet("backoffice.ai.usage.prompt_tokens")}
									</TableHead>
									<TableHead className="text-right">
										{textGet("backoffice.ai.usage.completion_tokens")}
									</TableHead>
									<TableHead className="text-right">
										{textGet("backoffice.ai.usage.cost")}
									</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{usage.rows.map((r) => (
									<TableRow key={`${r.tenant_id}-${r.function}`}>
										<TableCell>{r.tenant_name || `#${r.tenant_id}`}</TableCell>
										<TableCell>{functionLabel(r.function)}</TableCell>
										<TableCell className="text-right tabular-nums">
											{r.uses}
										</TableCell>
										<TableCell className="text-right tabular-nums">
											{r.prompt_tokens}
										</TableCell>
										<TableCell className="text-right tabular-nums">
											{r.completion_tokens}
										</TableCell>
										<TableCell className="text-right tabular-nums">
											{formatMoney(r.cost_usd)}
										</TableCell>
									</TableRow>
								))}
								<TableRow>
									<TableCell colSpan={5} className="font-medium">
										{textGet("backoffice.ai.usage.total")}
									</TableCell>
									<TableCell className="text-right font-medium tabular-nums">
										{formatMoney(totalCost)}
									</TableCell>
								</TableRow>
							</TableBody>
						</Table>
					)}
				</CardContent>
			</Card>
		</div>
	);
};

export default AIPage;
