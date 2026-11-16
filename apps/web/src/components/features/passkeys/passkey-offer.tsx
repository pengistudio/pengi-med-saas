import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Form,
	FormInput,
	FormPasswordInput,
	Spinner,
} from "@pengi/ui";
import { browserSupportsWebAuthn } from "@simplewebauthn/browser";
import { KeyRound } from "lucide-react";
import React from "react";
import z from "zod";
import { getPasskeys } from "@/api/passkey-service";
import { getTwoFactorStatus } from "@/api/two-factor-service";
import { usePasskeyRegistration } from "@/hooks/use-passkey-registration";
import { deviceLabel } from "@/lib/device-label";
import {
	dismissPasskeyOffer,
	isPasskeyOfferDismissed,
	snoozePasskeyOffer,
	takePasskeyOfferPending,
} from "@/lib/passkey-offer-storage";
import { selectMfaRequired, useSessionStore } from "@/store/session-store";

// Field names (= input ids) are unique on the page: the labels point at them.
const buildSchema = (needsPassword: boolean, needsCode: boolean) =>
	z.object({
		offer_name: z.string().trim().min(1).max(100),
		offer_password: needsPassword ? z.string().min(1) : z.string(),
		offer_code: needsCode ? z.string().trim().min(6).max(64) : z.string(),
	});

/**
 * Offers a passkey on the first screen after a PASSWORD login (the login form
 * leaves the mark; a passkey login does not). Shows only when the browser
 * supports WebAuthn, the user has no passkey yet and has not chosen "don't
 * ask again" on this device. It never blocks navigation, and stays away while
 * the clinic's MFA-required redirect is active.
 */
export function PasskeyOffer() {
	const mfaRequired = useSessionStore(selectMfaRequired);
	const [userId, setUserId] = React.useState<number>();
	const started = React.useRef(false);

	React.useEffect(() => {
		// Once per page load; the mark is consumed so a reload does not ask again.
		if (started.current) return;
		started.current = true;
		const id = takePasskeyOfferPending();
		if (
			id === null ||
			!browserSupportsWebAuthn() ||
			isPasskeyOfferDismissed(id)
		)
			return;
		getPasskeys(false).then((res) => {
			if (res.success && res.data.length === 0) setUserId(id);
		});
	}, []);

	if (userId === undefined || mfaRequired) return null;
	return <OfferDialog userId={userId} onClose={() => setUserId(undefined)} />;
}

function OfferDialog({
	userId,
	onClose,
}: {
	userId: number;
	onClose: () => void;
}) {
	const { textGet } = useText();
	const register = usePasskeyRegistration();
	const [reauth, setReauth] = React.useState(false);
	const [needsCode, setNeedsCode] = React.useState<boolean>();
	const [name, setName] = React.useState(
		() => deviceLabel() ?? textGet("auth.passkey_offer.default_name"),
	);
	const [loading, setLoading] = React.useState(false);

	// The code is only asked of users with 2FA, and only once the password is.
	React.useEffect(() => {
		if (!reauth) return;
		getTwoFactorStatus().then((res) => {
			// If the status can't be read, ask for the code: the server decides.
			setNeedsCode(res.success ? res.data.enabled : true);
		});
	}, [reauth]);

	const schema = React.useMemo(
		() => buildSchema(reauth, needsCode === true),
		[reauth, needsCode],
	);

	async function onSubmit(values: z.infer<ReturnType<typeof buildSchema>>) {
		setName(values.offer_name);
		setLoading(true);
		try {
			const result = await register({
				name: values.offer_name,
				...(reauth
					? {
							password: values.offer_password,
							code: needsCode ? values.offer_code : undefined,
						}
					: {}),
			});
			if (result === "done") onClose();
			else if (result === "reauth") setReauth(true);
		} finally {
			setLoading(false);
		}
	}

	function dontAskAgain() {
		dismissPasskeyOffer(userId);
		onClose();
	}

	// "Not now", Esc and the X all mean "not now": snooze, don't nag.
	function notNow() {
		snoozePasskeyOffer(userId);
		onClose();
	}

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open) notNow();
			}}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>{textGet("auth.passkey_offer.title")}</DialogTitle>
					<DialogDescription>
						{reauth
							? textGet("auth.passkey_offer.reauth_description")
							: textGet("auth.passkey_offer.description")}
					</DialogDescription>
				</DialogHeader>
				{reauth && needsCode === undefined ? (
					<Spinner />
				) : (
					<Form<ReturnType<typeof buildSchema>>
						key={`${reauth}-${needsCode}`}
						schema={schema}
						onSubmit={onSubmit}
						defaultValues={{
							offer_name: name,
							offer_password: "",
							offer_code: "",
						}}
					>
						{(field) => (
							<div className="space-y-4">
								<FormInput
									field={field}
									name="offer_name"
									type="text"
									label={textGet("profile.passkeys.name_label")}
									placeholder={textGet("profile.passkeys.name_placeholder")}
									autoComplete="off"
								/>
								{reauth && (
									<FormPasswordInput
										field={field}
										name="offer_password"
										label={textGet("profile.security.current_password")}
										autoFocus
									/>
								)}
								{reauth && needsCode && (
									<FormInput
										field={field}
										name="offer_code"
										type="text"
										label={textGet("profile.two_factor.proof.code_label")}
										description={textGet(
											"profile.two_factor.proof.code_description",
										)}
										autoComplete="one-time-code"
										spellCheck={false}
									/>
								)}
								<DialogFooter className="gap-2 sm:flex-wrap">
									<Button
										type="button"
										variant="ghost"
										onClick={dontAskAgain}
										disabled={loading}
									>
										{textGet("auth.passkey_offer.dont_ask")}
									</Button>
									<Button
										type="button"
										variant="outline"
										onClick={notNow}
										disabled={loading}
									>
										{textGet("auth.passkey_offer.not_now")}
									</Button>
									<Button type="submit" disabled={loading}>
										{loading ? <Spinner /> : <KeyRound className="h-4 w-4" />}
										{textGet("auth.passkey_offer.create")}
									</Button>
								</DialogFooter>
							</div>
						)}
					</Form>
				)}
			</DialogContent>
		</Dialog>
	);
}
