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
	Badge,
	Button,
	Spinner,
} from "@pengi/ui";
import { Fingerprint, ShieldOff } from "lucide-react";
import React from "react";
import {
	type BackofficeUser,
	resetPasskeys,
	resetTwoFactor,
} from "@/api/user-service";
import { useSession } from "@/lib/session";

type Kind = "two_factor" | "passkeys";

/** The second-factor state of an admin, as badges. */
export function SecondFactorBadges({ user }: { user: BackofficeUser }) {
	const { textGet } = useText();
	return (
		<div className="flex flex-wrap gap-1.5">
			<Badge variant={user.two_factor_enabled ? "default" : "outline"}>
				{textGet(
					user.two_factor_enabled
						? "backoffice.users.two_factor.on"
						: "backoffice.users.two_factor.off",
				)}
			</Badge>
			<Badge variant={user.passkey_count > 0 ? "default" : "outline"}>
				{textGet("backoffice.users.passkeys.count", {
					count: user.passkey_count,
				})}
			</Badge>
		</div>
	);
}

/**
 * "Reset 2FA" / "Reset passkeys" of another admin, each behind a confirmation.
 * Only offered for what the admin has, and never for yourself (the API
 * refuses it too, E-BO-012).
 */
export function SecondFactorResetActions({
	user,
	onChanged,
}: {
	user: BackofficeUser;
	onChanged: () => void;
}) {
	const { textGet } = useText();
	const { userId } = useSession();
	const [confirming, setConfirming] = React.useState<Kind | null>(null);
	const [busy, setBusy] = React.useState(false);

	if (user.ID === userId) return null;
	const canReset2fa = user.two_factor_enabled;
	const canResetPasskeys = user.passkey_count > 0;
	if (!canReset2fa && !canResetPasskeys) return null;

	async function confirm() {
		if (!confirming) return;
		setBusy(true);
		const res = await (confirming === "two_factor"
			? resetTwoFactor(user.ID)
			: resetPasskeys(user.ID));
		setBusy(false);
		setConfirming(null);
		if (res.success) onChanged();
	}

	return (
		<>
			{canReset2fa && (
				<Button
					variant="outline"
					size="sm"
					onClick={() => setConfirming("two_factor")}
				>
					<ShieldOff className="h-4 w-4" />
					{textGet("backoffice.users.reset_two_factor")}
				</Button>
			)}
			{canResetPasskeys && (
				<Button
					variant="outline"
					size="sm"
					onClick={() => setConfirming("passkeys")}
				>
					<Fingerprint className="h-4 w-4" />
					{textGet("backoffice.users.reset_passkeys")}
				</Button>
			)}
			<AlertDialog
				open={confirming !== null}
				onOpenChange={(open) => {
					if (!open && !busy) setConfirming(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{textGet(
								confirming === "passkeys"
									? "backoffice.users.reset_passkeys.title"
									: "backoffice.users.reset_two_factor.title",
							)}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{textGet(
								confirming === "passkeys"
									? "backoffice.users.reset_passkeys.description"
									: "backoffice.users.reset_two_factor.description",
								{ name: user.user_name },
							)}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel disabled={busy}>
							{textGet("backoffice.common.cancel")}
						</AlertDialogCancel>
						<AlertDialogAction
							onClick={confirm}
							disabled={busy}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							{busy && <Spinner />}
							{textGet("backoffice.users.reset.confirm")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}
