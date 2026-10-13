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
	AlertDialogTrigger,
	Badge,
	Button,
	Text,
} from "@pengi/ui";
import { AlertTriangle, Loader2, PenLine, ShieldCheck } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import type { DocumentSignature } from "@/api/signature-service";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { useDoctorStatus } from "@/store/doctors-store";
import { useMySignature } from "@/store/signature-store";

interface SignDocumentButtonProps {
	/** The document's current signature; when signed, a badge replaces the button. */
	signature?: DocumentSignature | null;
	/** Signs the document; resolves `true` on success. */
	onSign: () => Promise<boolean>;
	disabled?: boolean;
	/** Icon-only variant for table rows. */
	compact?: boolean;
	className?: string;
}

/**
 * "Firmar" action for a medical document: signs it with the current user's
 * electronic signature (P12), or sends them to their profile to upload one.
 */
export function SignDocumentButton({
	signature,
	onSign,
	disabled,
	compact,
	className,
}: SignDocumentButtonProps) {
	const { textGet, formatDateTime } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const canSign = checkPermission([
		PERMISSIONS.MEDICAL_RECORD.PERMISSION_SIGN_MEDICAL_DOCUMENT,
	]);
	const isSigned = Boolean(signature?.signed_at);
	const mySignature = useMySignature(canSign && !isSigned);
	// Only the doctor's own user signs, so the signer's profile is the
	// document's doctor: warn (don't block) when it has no registry number.
	const doctorStatus = useDoctorStatus(canSign && !isSigned);
	const missingRegistry =
		!!doctorStatus?.doctor && !doctorStatus.doctor.professional_registry;
	const [signing, setSigning] = React.useState(false);

	if (isSigned) {
		const signedAt = formatDateTime(signature?.signed_at);
		const label = `${textGet("signature.signed_by")} ${signature?.signer_name ?? ""}`;
		return (
			<Badge
				variant="outline"
				className={cn(
					className,
					"gap-1.5 border-green-600/40 text-green-700 dark:text-green-400",
					// Signer names are often long: wrap instead of clipping.
					!compact &&
						"h-auto max-w-full items-start justify-start py-1 text-left whitespace-normal",
				)}
				title={`${label} · ${signedAt}`}
			>
				<ShieldCheck className="mt-px h-3.5 w-3.5 shrink-0" />
				{!compact && <span className="min-w-0 break-words">{label}</span>}
			</Badge>
		);
	}

	if (!canSign) return null;

	if (mySignature && (!mySignature.configured || mySignature.expired)) {
		const setupLabel = textGet(
			mySignature.expired ? "signature.expired.setup" : "signature.setup",
		);
		return (
			<Button
				type="button"
				variant="outline"
				size={compact ? "icon" : "default"}
				className={className}
				title={setupLabel}
				onClick={() => navigate("/profile")}
			>
				<PenLine className={compact ? "h-4 w-4" : "mr-2 h-4 w-4"} />
				{!compact && setupLabel}
			</Button>
		);
	}

	async function handleSign() {
		setSigning(true);
		await onSign();
		setSigning(false);
	}

	return (
		<AlertDialog>
			<AlertDialogTrigger
				render={
					<Button
						type="button"
						variant={compact ? "ghost" : "outline"}
						size={compact ? "icon" : "default"}
						className={className}
						title={textGet("signature.sign.button")}
						disabled={disabled || signing || !mySignature}
					>
						{signing ? (
							<Loader2
								className={
									compact ? "h-4 w-4 animate-spin" : "mr-2 h-4 w-4 animate-spin"
								}
							/>
						) : (
							<PenLine className={compact ? "h-4 w-4" : "mr-2 h-4 w-4"} />
						)}
						{!compact && <Text uuid="signature.sign.button" />}
					</Button>
				}
			/>
			<AlertDialogContent>
				<AlertDialogHeader>
					<AlertDialogTitle>
						<Text uuid="signature.sign.confirm.title" />
					</AlertDialogTitle>
					<AlertDialogDescription>
						<Text uuid="signature.sign.confirm.description" />
					</AlertDialogDescription>
				</AlertDialogHeader>
				{mySignature?.subject_name && (
					<p className="text-sm">
						<span className="text-muted-foreground">
							<Text uuid="signature.holder" />:
						</span>{" "}
						<span className="font-medium">{mySignature.subject_name}</span>
					</p>
				)}
				{missingRegistry && (
					<p className="flex items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 p-2 text-sm text-amber-800 dark:text-amber-300">
						<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
						<Text uuid="doctors.sign.missing_registry" type="span" />
					</p>
				)}
				<AlertDialogFooter>
					<AlertDialogCancel>
						<Text uuid="form.cancel" />
					</AlertDialogCancel>
					<AlertDialogAction onClick={handleSign}>
						<Text uuid="signature.sign.button" />
					</AlertDialogAction>
				</AlertDialogFooter>
			</AlertDialogContent>
		</AlertDialog>
	);
}
