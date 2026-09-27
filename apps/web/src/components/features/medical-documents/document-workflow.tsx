import { useText } from "@pengi/shared";
import { Button, Text } from "@pengi/ui";
import { Check, Loader2, Printer, Save } from "lucide-react";
import React from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";
import type { DocumentSignature } from "@/api/signature-service";
import { SendEmailPopover } from "@/components/custom/send-email-popover";
import { SignDocumentButton } from "@/components/custom/sign-document-button";
import { cn } from "@/lib/utils";

/**
 * After each save the saved values become the form's new baseline, so a
 * later edit shows as unsaved and blocks signing or printing a stale copy.
 */
export function useSavedBaseline<T extends FieldValues>(
	field: UseFormReturn<T>,
	savedVersion: number,
) {
	React.useEffect(() => {
		if (savedVersion > 0) field.reset(field.getValues());
	}, [savedVersion, field]);
}

/** Who signed and when, as wrapping text: signer names are often long. */
function SignedBy({ signature }: { signature: DocumentSignature }) {
	const { textGet } = useText();
	return (
		<>
			{textGet("signature.signed_by")}{" "}
			<span className="font-medium break-words text-foreground">
				{signature.signer_name}
			</span>
			{signature.signed_at && (
				<span className="block tabular-nums">
					{new Date(signature.signed_at).toLocaleString("es-EC", {
						dateStyle: "short",
						timeStyle: "short",
					})}
				</span>
			)}
		</>
	);
}

type StepState = "done" | "current" | "pending";

function Step({
	number,
	state,
	title,
	hint,
	children,
}: {
	number: number;
	state: StepState;
	title: React.ReactNode;
	hint?: React.ReactNode;
	children?: React.ReactNode;
}) {
	return (
		<li className="grid grid-cols-[1.75rem_minmax(0,1fr)] gap-x-3">
			<span
				className={cn(
					"grid size-7 place-items-center rounded-full border text-xs font-semibold tabular-nums",
					state === "done" &&
						"border-primary bg-primary text-primary-foreground",
					state === "current" && "border-primary text-primary",
					state === "pending" && "text-muted-foreground",
				)}
			>
				{state === "done" ? <Check className="size-3.5" /> : number}
			</span>
			<div className="min-w-0 space-y-2 pb-5">
				<div className="pt-1">
					<p
						className={cn(
							"text-sm font-medium",
							state === "pending" && "text-muted-foreground",
						)}
					>
						{title}
					</p>
					{hint && <p className="text-xs text-muted-foreground">{hint}</p>}
				</div>
				{children}
			</div>
		</li>
	);
}

export interface DocumentWorkflowProps {
	/** Label of the first save, e.g. "Guardar informe". */
	saveLabel: React.ReactNode;
	saving: boolean;
	printing: boolean;
	isSaved: boolean;
	isDirty: boolean;
	signature: DocumentSignature | null;
	patientEmail: string;
	onSign: () => Promise<boolean>;
	onPrint: () => void;
	onSendEmail: (email: string) => Promise<void>;
}

/**
 * A medical document's workflow: save, optionally sign, then print or send.
 * A sticky panel on desktop; the next action pinned to the bottom on mobile.
 * Must render inside the document's <form> (the save button submits it).
 */
export function DocumentWorkflow({
	saveLabel,
	saving,
	printing,
	isSaved,
	isDirty,
	signature,
	patientEmail,
	onSign,
	onPrint,
	onSendEmail,
}: DocumentWorkflowProps) {
	const saved = isSaved && !isDirty;
	const signed = Boolean(signature);

	const saveButton = (
		<Button type="submit" disabled={saving || signed} className="w-full">
			{saving ? (
				<Loader2 className="mr-2 size-4 animate-spin" />
			) : (
				<Save className="mr-2 size-4" />
			)}
			{isSaved ? <Text uuid="medical_document.save_new_version" /> : saveLabel}
		</Button>
	);

	const deliverButtons = (
		<div className="grid grid-cols-2 gap-2 lg:grid-cols-1">
			<Button
				type="button"
				variant={saved ? "default" : "outline"}
				disabled={!saved || printing}
				onClick={onPrint}
			>
				{printing ? (
					<Loader2 className="mr-2 size-4 animate-spin" />
				) : (
					<Printer className="mr-2 size-4" />
				)}
				<Text uuid="medical_document.print" />
			</Button>
			<SendEmailPopover
				defaultEmail={patientEmail}
				onSend={onSendEmail}
				disabled={!saved}
				label={<Text uuid="medical_document.email_short" />}
				className="w-full"
			/>
		</div>
	);

	return (
		<>
			<aside className="hidden lg:sticky lg:top-6 lg:block">
				<ol className="rounded-xl border bg-card p-5">
					<Step
						number={1}
						state={saved ? "done" : "current"}
						title={<Text uuid="medical_document.steps.save" />}
						hint={
							<Text
								uuid={
									saved
										? "medical_document.steps.save.done"
										: isDirty && isSaved
											? "medical_document.steps.save.dirty"
											: "medical_document.steps.save.pending"
								}
							/>
						}
					>
						{!saved && saveButton}
					</Step>
					<Step
						number={2}
						state={signed ? "done" : saved ? "current" : "pending"}
						title={<Text uuid="medical_document.steps.sign" />}
						hint={
							signature ? (
								<SignedBy signature={signature} />
							) : (
								<Text uuid="medical_document.steps.sign.hint" />
							)
						}
					>
						{saved && !signed && (
							<SignDocumentButton
								signature={signature}
								onSign={onSign}
								className="w-full"
							/>
						)}
					</Step>
					<Step
						number={3}
						state={saved ? "current" : "pending"}
						title={<Text uuid="medical_document.steps.deliver" />}
						hint={!saved && <Text uuid="medical_document.steps.deliver.hint" />}
					>
						{deliverButtons}
					</Step>
				</ol>
			</aside>

			{/* The layout's <main> has p-4 md:p-6; offset it so the bar meets the screen edge. */}
			<div className="sticky -bottom-4 z-10 -mx-4 space-y-2 border-t bg-background/95 px-4 pt-3 pb-4 backdrop-blur md:-bottom-6 md:-mx-6 md:px-6 md:pb-6 lg:hidden">
				{isDirty && isSaved && (
					<p className="text-xs text-muted-foreground">
						<Text uuid="medical_document.steps.save.dirty" />
					</p>
				)}
				{!saved ? (
					saveButton
				) : (
					<div className="space-y-2">
						{!signed && (
							<SignDocumentButton
								signature={signature}
								onSign={onSign}
								className="w-full"
							/>
						)}
						{deliverButtons}
					</div>
				)}
			</div>
		</>
	);
}
