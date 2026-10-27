import { useText } from "@pengi/shared";
import { Button, cn, Input, Label, Spinner } from "@pengi/ui";
import { Mail, MessageCircle } from "lucide-react";
import React from "react";
import {
	type ContactChannel,
	requestNewPatientCode,
	verifyNewPatientCode,
} from "@/api/portal-service";
import {
	CaptchaField,
	CodeInput,
	Honeypot,
	useCaptcha,
} from "@/components/verification";
import {
	type ContactErrorKind,
	contactErrorKind,
	isValidContact,
	type VerifiedContact,
} from "@/lib/new-patient-flow";

const CHANNEL_ICON = { whatsapp: MessageCircle, email: Mail } as const;

/**
 * Step 1: which contact to verify (WhatsApp or email) and asking for the
 * code. Once well-formed, the API answers the same for any contact.
 */
export function ContactForm({
	slug,
	channels,
	initial,
	onCodeSent,
	onNotAccepted,
}: {
	slug: string;
	/** Available channels, preferred first. */
	channels: ContactChannel[];
	initial?: VerifiedContact;
	onCodeSent: (contact: VerifiedContact) => void;
	/** The clinic stopped taking new patients. */
	onNotAccepted: () => void;
}) {
	const { textGet } = useText();
	const [channel, setChannel] = React.useState<ContactChannel>(
		initial && channels.includes(initial.channel)
			? initial.channel
			: channels[0],
	);
	const [contact, setContact] = React.useState(initial?.contact ?? "");
	const [website, setWebsite] = React.useState("");
	const [busy, setBusy] = React.useState(false);
	const [error, setError] = React.useState<{
		kind: ContactErrorKind;
		message: string;
	}>();
	const captcha = useCaptcha();
	const trimmed = contact.trim();
	const inputId = "new-patient-contact";

	const submit = async (event: React.FormEvent) => {
		event.preventDefault();
		if (captcha.pending) return;
		if (!isValidContact(channel, trimmed)) {
			setError({ kind: "invalid_contact", message: "" });
			return;
		}
		setBusy(true);
		setError(undefined);
		const res = await requestNewPatientCode(slug, {
			channel,
			contact: trimmed,
			website,
			...captcha.payload(),
		});
		// A token works once: get a fresh one for the next request.
		captcha.reset();
		setBusy(false);
		if (res.success) {
			onCodeSent({ channel, contact: trimmed });
			return;
		}
		const kind = contactErrorKind(res);
		if (kind === "not_accepted") {
			onNotAccepted();
			return;
		}
		setError({ kind, message: res.message });
	};

	const errorText =
		error &&
		(error.kind === "invalid_contact"
			? textGet(
					channel === "email"
						? "portal.new_patient.error.contact_email"
						: "portal.new_patient.error.contact_phone",
				)
			: error.kind === "captcha"
				? textGet("portal.signin.captcha_error")
				: error.message);

	return (
		<form onSubmit={submit} className="max-w-md space-y-5" noValidate>
			{channels.length > 1 && (
				<fieldset>
					<legend className="pb-2 text-sm font-medium">
						{textGet("portal.new_patient.channel")}
					</legend>
					<div className="grid grid-cols-2 gap-1 rounded-full bg-muted p-1">
						{channels.map((c) => {
							const Icon = CHANNEL_ICON[c];
							const active = c === channel;
							return (
								<label
									key={c}
									className={cn(
										"flex h-10 cursor-pointer items-center justify-center gap-2 rounded-full text-sm font-medium transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring has-[:focus-visible]:ring-offset-2",
										active
											? "bg-card text-foreground shadow-sm"
											: "text-muted-foreground hover:text-foreground",
									)}
								>
									<input
										type="radio"
										name="channel"
										value={c}
										checked={active}
										onChange={() => {
											setChannel(c);
											setContact("");
											setError(undefined);
										}}
										className="sr-only"
									/>
									<Icon aria-hidden="true" className="size-4" />
									{textGet(`portal.new_patient.channel.${c}`)}
								</label>
							);
						})}
					</div>
				</fieldset>
			)}

			<div className="space-y-2">
				<Label htmlFor={inputId}>
					{textGet(
						channel === "email"
							? "portal.new_patient.contact.email"
							: "portal.new_patient.contact.whatsapp",
					)}
				</Label>
				<Input
					id={inputId}
					name={channel === "email" ? "email" : "phone"}
					type={channel === "email" ? "email" : "tel"}
					inputMode={channel === "email" ? "email" : "tel"}
					autoComplete={channel === "email" ? "email" : "tel"}
					placeholder={
						channel === "email"
							? textGet("portal.new_patient.contact.email.placeholder")
							: textGet("portal.new_patient.contact.whatsapp.placeholder")
					}
					value={contact}
					onChange={(e) => {
						setContact(e.target.value);
						if (error?.kind === "invalid_contact") setError(undefined);
					}}
					aria-invalid={error?.kind === "invalid_contact" ? true : undefined}
					aria-describedby={errorText ? `${inputId}-error` : undefined}
					className="h-11 text-base"
				/>
				{errorText && (
					<p
						id={`${inputId}-error`}
						role="alert"
						className="text-destructive text-sm"
					>
						{errorText}
					</p>
				)}
			</div>

			<Honeypot value={website} onChange={setWebsite} />
			<CaptchaField captcha={captcha} />

			<Button
				type="submit"
				className="h-12 w-full gap-2 rounded-full text-base"
				disabled={busy || trimmed === "" || captcha.pending}
			>
				{busy && <Spinner />}
				{textGet("portal.signin.send_code")}
			</Button>
			<p className="text-muted-foreground text-xs">
				{textGet("portal.new_patient.contact.hint")}
			</p>
		</form>
	);
}

/** Step 2: the 6-digit code; a right one trades for a guest token. */
export function CodeForm({
	slug,
	contact,
	onVerified,
	onChangeContact,
}: {
	slug: string;
	contact: VerifiedContact;
	onVerified: (guestToken: string) => void;
	onChangeContact: () => void;
}) {
	const { textGet } = useText();
	const [code, setCode] = React.useState("");
	const [busy, setBusy] = React.useState(false);
	const [error, setError] = React.useState<string>();

	const submit = async (event: React.FormEvent) => {
		event.preventDefault();
		setBusy(true);
		const res = await verifyNewPatientCode(slug, { ...contact, code });
		setBusy(false);
		if (res.success) {
			onVerified(res.data.guest_token);
			return;
		}
		setError(
			res.code === 401 || res.code === 400
				? textGet("portal.signin.code_invalid")
				: res.message,
		);
	};

	return (
		<form onSubmit={submit} className="max-w-md space-y-5">
			<p className="text-muted-foreground">
				{textGet(
					contact.channel === "email"
						? "portal.new_patient.code_sent.email"
						: "portal.new_patient.code_sent.whatsapp",
					{ contact: contact.contact },
				)}
			</p>
			<CodeInput
				id="new-patient-code"
				value={code}
				onChange={(value) => {
					setCode(value);
					setError(undefined);
				}}
				error={error}
			/>
			<Button
				type="submit"
				className="h-12 w-full gap-2 rounded-full text-base"
				disabled={busy || code.length !== 6}
			>
				{busy && <Spinner />}
				{textGet("portal.new_patient.verify")}
			</Button>
			<Button
				type="button"
				variant="ghost"
				className="h-11 w-full rounded-full"
				onClick={onChangeContact}
			>
				{textGet("portal.new_patient.change_contact")}
			</Button>
		</form>
	);
}
