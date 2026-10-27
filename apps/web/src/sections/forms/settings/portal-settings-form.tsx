import { useText } from "@pengi/shared";
import {
	Button,
	Form,
	FormInput,
	FormRadioGroup,
	FormRow,
	FormSelect,
} from "@pengi/ui";
import { Check, Copy, Loader2, Save } from "lucide-react";
import React from "react";
import { z } from "zod";
import {
	type PortalSections,
	type PortalSettings,
	REQUEST_EXPIRY_HOURS,
	SLOT_INTERVAL_MINUTES,
} from "@/api/portal-settings-service";
import { SettingsSection } from "@/components/custom/settings-section";
import { FormSwitch } from "@/components/forms/form-switch";

const SECTION_KEYS: (keyof PortalSections)[] = [
	"appointments",
	"results",
	"documents",
	"invoices",
];

const hours = (max: number, key: string) =>
	z.number({ error: key }).int(key).min(0, key).max(max, key);

const formSchema = z.object({
	enabled: z.boolean(),
	sections: z.object({
		appointments: z.boolean(),
		results: z.boolean(),
		documents: z.boolean(),
		invoices: z.boolean(),
	}),
	booking: z.object({
		enabled: z.boolean(),
		choose_by: z.enum(["doctor", "specialty"]),
		auto_confirm: z.boolean(),
		request_expiry_hours: z.enum(["12", "24", "48"]),
		slot_interval_minutes: z.enum(["0", "10", "15", "20", "30", "60"]),
		min_notice_hours: hours(72, "settings.portal.booking.error.min_notice"),
		max_days_ahead: z
			.number({ error: "settings.portal.booking.error.max_days" })
			.int("settings.portal.booking.error.max_days")
			.min(1, "settings.portal.booking.error.max_days")
			.max(180, "settings.portal.booking.error.max_days"),
		cancel_notice_hours: hours(
			168,
			"settings.portal.booking.error.cancel_notice",
		),
	}),
});

type PortalSettingsValues = z.infer<typeof formSchema>;

const toValues = (settings: PortalSettings): PortalSettingsValues => ({
	...settings,
	booking: {
		...settings.booking,
		request_expiry_hours: String(
			settings.booking.request_expiry_hours,
		) as PortalSettingsValues["booking"]["request_expiry_hours"],
		slot_interval_minutes: String(
			settings.booking.slot_interval_minutes ?? 0,
		) as PortalSettingsValues["booking"]["slot_interval_minutes"],
	},
});

const toSettings = (values: PortalSettingsValues): PortalSettings => ({
	...values,
	booking: {
		...values.booking,
		request_expiry_hours: Number(
			values.booking.request_expiry_hours,
		) as PortalSettings["booking"]["request_expiry_hours"],
		slot_interval_minutes: Number(
			values.booking.slot_interval_minutes,
		) as PortalSettings["booking"]["slot_interval_minutes"],
	},
});

/**
 * The patient portal settings: access and link, what patients see, and
 * booking from the portal. The page loads and saves them.
 */
export function PortalSettingsForm({
	initialData,
	portalUrl,
	loading,
	onSubmit,
}: {
	initialData: PortalSettings;
	portalUrl: string;
	loading?: boolean;
	onSubmit: (settings: PortalSettings) => void;
}) {
	const { textGet } = useText();

	return (
		<Form
			schema={formSchema}
			defaultValues={toValues(initialData)}
			onSubmit={(values: PortalSettingsValues) => onSubmit(toSettings(values))}
			className="grid grid-cols-1 items-start gap-4"
		>
			{(field) => {
				const enabled = field.watch("enabled");
				const appointments = field.watch("sections.appointments");
				const bookingOn = field.watch("booking.enabled");
				const bookingAvailable = enabled && appointments;
				const bookingFields = !bookingAvailable || !bookingOn;
				return (
					<>
						<SettingsSection
							title={textGet("settings.portal.access.title")}
							description={<p>{textGet("settings.portal.access.help")}</p>}
						>
							<FormSwitch
								field={field}
								name="enabled"
								label={textGet("settings.portal.enabled")}
							/>
							<PortalLink url={portalUrl} />
						</SettingsSection>

						<SettingsSection
							title={textGet("settings.portal.sections.title")}
							description={<p>{textGet("settings.portal.sections.help")}</p>}
						>
							<div className="grid gap-4">
								{SECTION_KEYS.map((key) => (
									<FormSwitch
										key={key}
										field={field}
										name={`sections.${key}`}
										label={textGet(`settings.portal.sections.${key}`)}
										disabled={!enabled}
									/>
								))}
							</div>
						</SettingsSection>

						<SettingsSection
							title={textGet("settings.portal.booking.title")}
							description={
								<>
									<p>{textGet("settings.portal.booking.help")}</p>
									<p>{textGet("settings.portal.booking.opt_in_help")}</p>
								</>
							}
						>
							<FormSwitch
								field={field}
								name="booking.enabled"
								label={textGet("settings.portal.booking.enabled")}
								description={
									bookingAvailable
										? undefined
										: textGet("settings.portal.booking.needs_appointments")
								}
								disabled={!bookingAvailable}
							/>
							<fieldset
								disabled={bookingFields}
								className="grid gap-6 disabled:opacity-60"
							>
								<FormRadioGroup
									field={field}
									name="booking.choose_by"
									label={textGet("settings.portal.booking.choose_by")}
									options={[
										{
											value: "doctor",
											label: textGet(
												"settings.portal.booking.choose_by.doctor",
											),
										},
										{
											value: "specialty",
											label: textGet(
												"settings.portal.booking.choose_by.specialty",
											),
										},
									]}
								/>
								<FormSwitch
									field={field}
									name="booking.auto_confirm"
									label={textGet("settings.portal.booking.auto_confirm")}
									description={textGet(
										"settings.portal.booking.auto_confirm.description",
									)}
									disabled={bookingFields}
								/>
								<FormSelect
									field={field}
									name="booking.request_expiry_hours"
									label={textGet("settings.portal.booking.expiry")}
									description={textGet(
										"settings.portal.booking.expiry.description",
									)}
									disabled={
										bookingFields || field.watch("booking.auto_confirm")
									}
									options={REQUEST_EXPIRY_HOURS.map((h) => ({
										value: String(h),
										label: textGet("settings.portal.booking.hours", {
											count: h,
										}),
									}))}
								/>
								<FormSelect
									field={field}
									name="booking.slot_interval_minutes"
									label={textGet("settings.portal.booking.slot_interval")}
									description={textGet(
										"settings.portal.booking.slot_interval.description",
									)}
									disabled={bookingFields}
									options={SLOT_INTERVAL_MINUTES.map((m) => ({
										value: String(m),
										label:
											m === 0
												? textGet("settings.portal.booking.slot_interval.type")
												: textGet(
														"settings.portal.booking.slot_interval.minutes",
														{
															count: m,
														},
													),
									}))}
								/>
								<FormRow cols={3}>
									<FormInput
										field={field}
										name="booking.min_notice_hours"
										type="number"
										min={0}
										max={72}
										label={textGet("settings.portal.booking.min_notice")}
										description={textGet(
											"settings.portal.booking.min_notice.description",
										)}
									/>
									<FormInput
										field={field}
										name="booking.max_days_ahead"
										type="number"
										min={1}
										max={180}
										label={textGet("settings.portal.booking.max_days")}
										description={textGet(
											"settings.portal.booking.max_days.description",
										)}
									/>
									<FormInput
										field={field}
										name="booking.cancel_notice_hours"
										type="number"
										min={0}
										max={168}
										label={textGet("settings.portal.booking.cancel_notice")}
										description={textGet(
											"settings.portal.booking.cancel_notice.description",
										)}
									/>
								</FormRow>
							</fieldset>
						</SettingsSection>

						<div className="flex justify-end">
							<Button type="submit" disabled={loading}>
								{loading ? (
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
								) : (
									<Save className="mr-2 h-4 w-4" />
								)}
								{textGet("settings.portal.save")}
							</Button>
						</div>
					</>
				);
			}}
		</Form>
	);
}

/** The portal's public URL with a copy button. */
function PortalLink({ url }: { url: string }) {
	const { textGet } = useText();
	const [copied, setCopied] = React.useState(false);

	const handleCopy = async () => {
		await navigator.clipboard.writeText(url);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<div className="grid gap-1.5">
			<p className="text-sm font-medium">{textGet("settings.portal.link")}</p>
			<div className="flex items-center gap-2">
				<code className="min-w-0 flex-1 truncate rounded-md border bg-muted px-3 py-2 text-sm">
					{url}
				</code>
				<Button
					type="button"
					variant="outline"
					disabled={!url}
					onClick={handleCopy}
				>
					{copied ? (
						<Check className="mr-2 h-4 w-4" />
					) : (
						<Copy className="mr-2 h-4 w-4" />
					)}
					{textGet(copied ? "settings.portal.copied" : "settings.portal.copy")}
				</Button>
			</div>
		</div>
	);
}
