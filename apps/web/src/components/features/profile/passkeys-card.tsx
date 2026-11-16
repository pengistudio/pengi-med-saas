import { useText } from "@pengi/shared";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
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
	useToast,
} from "@pengi/ui";
import {
	browserSupportsWebAuthn,
	startRegistration,
} from "@simplewebauthn/browser";
import { KeyRound, Trash } from "lucide-react";
import React from "react";
import z from "zod";
import {
	beginPasskeyRegistration,
	deletePasskey,
	finishPasskeyRegistration,
	getPasskeys,
	type Passkey,
} from "@/api/passkey-service";
import { getTwoFactorStatus } from "@/api/two-factor-service";

// Field names (= input ids) differ from the other cards on the profile page:
// the dialog labels must point at their inputs.
const buildSchema = (needsCode: boolean) =>
	z.object({
		passkey_name: z.string().trim().min(1).max(100),
		passkey_password: z.string().min(1),
		passkey_code: needsCode ? z.string().trim().min(6).max(64) : z.string(),
	});

/**
 * Passkeys (WebAuthn) of the signed-in user: list, add (password, plus a
 * 2FA code when 2FA is on) and remove. Lives in the profile's security
 * section.
 */
export function PasskeysCard() {
	const { textGet, formatDate, formatRelative } = useText();
	const [passkeys, setPasskeys] = React.useState<Passkey[]>();
	const [adding, setAdding] = React.useState(false);
	const [toDelete, setToDelete] = React.useState<Passkey | null>(null);
	const supported = browserSupportsWebAuthn();

	const load = React.useCallback(() => {
		getPasskeys().then((res) => {
			if (res.success) setPasskeys(res.data);
		});
	}, []);

	React.useEffect(() => {
		load();
	}, [load]);

	async function confirmDelete() {
		if (!toDelete) return;
		const res = await deletePasskey(toDelete.id);
		setToDelete(null);
		if (res.success) load();
	}

	return (
		<Card>
			<CardHeader>
				<CardTitle>{textGet("profile.passkeys.title")}</CardTitle>
				<CardDescription>
					{textGet("profile.passkeys.description")}
				</CardDescription>
			</CardHeader>
			<CardContent className="space-y-4">
				{!passkeys ? (
					<Spinner />
				) : passkeys.length === 0 ? (
					<p className="text-sm text-muted-foreground">
						{textGet("profile.passkeys.empty")}
					</p>
				) : (
					<ul className="divide-y rounded-md border">
						{passkeys.map((passkey) => (
							<li
								key={passkey.id}
								className="flex items-center justify-between gap-3 p-3"
							>
								<div className="min-w-0">
									<p className="truncate text-sm font-medium">{passkey.name}</p>
									<p className="text-xs text-muted-foreground">
										{textGet("profile.passkeys.created", {
											date: formatDate(passkey.created_at, "long"),
										})}
										{" · "}
										{passkey.last_used_at
											? textGet("profile.passkeys.last_used", {
													date: formatRelative(passkey.last_used_at),
												})
											: textGet("profile.passkeys.never_used")}
									</p>
								</div>
								<Button
									type="button"
									variant="ghost"
									size="icon"
									className="text-destructive"
									aria-label={textGet("profile.passkeys.delete", {
										name: passkey.name,
									})}
									onClick={() => setToDelete(passkey)}
								>
									<Trash className="h-4 w-4" />
								</Button>
							</li>
						))}
					</ul>
				)}
				{!supported && (
					<p role="alert" className="text-sm text-muted-foreground">
						{textGet("profile.passkeys.unsupported")}
					</p>
				)}
				<div className="flex justify-end">
					<Button disabled={!supported} onClick={() => setAdding(true)}>
						<KeyRound className="h-4 w-4" />
						{textGet("profile.passkeys.add")}
					</Button>
				</div>
			</CardContent>

			{adding && (
				<AddDialog
					onClose={() => setAdding(false)}
					onAdded={() => {
						setAdding(false);
						load();
					}}
				/>
			)}

			<AlertDialog
				open={toDelete !== null}
				onOpenChange={(open) => {
					if (!open) setToDelete(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{textGet("profile.passkeys.delete.title")}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{textGet("profile.passkeys.delete.description", {
								name: toDelete?.name ?? "",
							})}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>
							{textGet("profile.passkeys.cancel")}
						</AlertDialogCancel>
						<AlertDialogAction onClick={confirmDelete}>
							{textGet("profile.passkeys.delete.confirm")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</Card>
	);
}

/**
 * Add a passkey: name + current password (+ code when 2FA is on) -> begin ->
 * browser ceremony -> finish. A cancelled prompt (NotAllowedError) is silent.
 */
function AddDialog({
	onClose,
	onAdded,
}: {
	onClose: () => void;
	onAdded: () => void;
}) {
	const { textGet } = useText();
	const { errorToast } = useToast();
	const [needsCode, setNeedsCode] = React.useState<boolean>();
	const [loading, setLoading] = React.useState(false);

	React.useEffect(() => {
		getTwoFactorStatus().then((res) => {
			// If the status can't be read, ask for the code: the server decides.
			setNeedsCode(res.success ? res.data.enabled : true);
		});
	}, []);

	const schema = React.useMemo(
		() => buildSchema(needsCode === true),
		[needsCode],
	);

	async function onSubmit(values: z.infer<ReturnType<typeof buildSchema>>) {
		setLoading(true);
		try {
			const begin = await beginPasskeyRegistration(
				values.passkey_password,
				needsCode ? values.passkey_code : undefined,
			);
			if (!begin.success) return;
			let credential: Awaited<ReturnType<typeof startRegistration>>;
			try {
				credential = await startRegistration({
					optionsJSON: begin.data.options,
				});
			} catch (err) {
				if (!(err instanceof Error && err.name === "NotAllowedError")) {
					errorToast(null, textGet("auth.passkey.browser_error"));
				}
				return;
			}
			const res = await finishPasskeyRegistration(
				begin.data.challenge_id,
				values.passkey_name,
				credential,
			);
			if (res.success) onAdded();
		} finally {
			setLoading(false);
		}
	}

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open) onClose();
			}}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>{textGet("profile.passkeys.add.title")}</DialogTitle>
					<DialogDescription>
						{textGet("profile.passkeys.add.description")}
					</DialogDescription>
				</DialogHeader>
				{needsCode === undefined ? (
					<Spinner />
				) : (
					<Form<ReturnType<typeof buildSchema>>
						key={String(needsCode)}
						schema={schema}
						onSubmit={onSubmit}
						defaultValues={{
							passkey_name: "",
							passkey_password: "",
							passkey_code: "",
						}}
					>
						{(field) => (
							<div className="space-y-4">
								<FormInput
									field={field}
									name="passkey_name"
									type="text"
									label={textGet("profile.passkeys.name_label")}
									placeholder={textGet("profile.passkeys.name_placeholder")}
									autoComplete="off"
									autoFocus
								/>
								<FormPasswordInput
									field={field}
									name="passkey_password"
									label={textGet("profile.security.current_password")}
								/>
								{needsCode && (
									<FormInput
										field={field}
										name="passkey_code"
										type="text"
										label={textGet("profile.two_factor.proof.code_label")}
										description={textGet(
											"profile.two_factor.proof.code_description",
										)}
										autoComplete="one-time-code"
										spellCheck={false}
									/>
								)}
								<DialogFooter>
									<Button type="button" variant="outline" onClick={onClose}>
										{textGet("profile.passkeys.cancel")}
									</Button>
									<Button type="submit" disabled={loading}>
										{loading && <Spinner />}
										{textGet("profile.passkeys.add.submit")}
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
