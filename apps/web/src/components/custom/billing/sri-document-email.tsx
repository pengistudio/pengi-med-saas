import type { ServiceResponse } from "@pengi/shared";
import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Form,
	FormInput,
	Spinner,
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@pengi/ui";
import { Mail } from "lucide-react";
import { useState } from "react";
import { z } from "zod";
import type { SriDocumentEmailResult } from "@/api/billing-service";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";

/** Delivery fields the list endpoints add to every comprobante. */
export interface SriDocumentEmailState {
	status: string;
	emailed_at?: string | null;
	emailed_to?: string | null;
	email_error?: string | null;
}

/** "Correo" badge of an authorized comprobante; nothing for other statuses. */
export function SriDocumentEmailStatus({
	doc,
}: {
	doc: SriDocumentEmailState;
}) {
	const { textGet, formatDateTime } = useText();
	if (doc.status !== "authorized") return null;

	let className = "text-muted-foreground";
	let label = textGet("billing.document.email.status.not_sent");
	let detail = label;
	const lastSent = doc.emailed_at
		? textGet("billing.document.email.status.sent_to", {
				email: doc.emailed_to ?? "",
				date: formatDateTime(doc.emailed_at),
			})
		: "";
	// A failed resend keeps the previous emailed_at, so the error wins: a newer
	// failure must not hide behind an older success.
	if (doc.email_error) {
		className = "border-destructive text-destructive";
		label = textGet("billing.document.email.status.failed");
		detail = lastSent
			? `${textGet(doc.email_error)} ${lastSent}`
			: textGet(doc.email_error);
	} else if (doc.emailed_at) {
		className = "border-emerald-500 text-emerald-600";
		label = textGet("billing.document.email.status.sent");
		detail = lastSent;
	}

	return (
		<Tooltip>
			<TooltipTrigger render={<span className="inline-flex" />}>
				<Badge variant="outline" className={className}>
					<Mail />
					{label}
				</Badge>
				<span className="sr-only">{detail}</span>
			</TooltipTrigger>
			<TooltipContent side="top" className="max-w-64 text-xs">
				{detail}
			</TooltipContent>
		</Tooltip>
	);
}

const formSchema = z.object({
	email: z
		.string()
		.trim()
		.min(1, "billing.document.email.field.error")
		.pipe(z.email("billing.document.email.field.error")),
});

type FormValues = z.infer<typeof formSchema>;

interface EmailDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	initialEmail: string;
	send: (email: string) => Promise<ServiceResponse<SriDocumentEmailResult>>;
	onSent: () => void;
}

export function SriDocumentEmailDialog({
	open,
	onOpenChange,
	initialEmail,
	send,
	onSent,
}: EmailDialogProps) {
	const { textGet } = useText();
	const [sending, setSending] = useState(false);

	async function onSubmit(values: FormValues) {
		setSending(true);
		const res = await send(values.email);
		setSending(false);
		// The backend records a failed send on the document too, so refresh the
		// list either way; only close the dialog on success.
		if (res.success) onOpenChange(false);
		onSent();
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>
						{textGet("billing.document.email.dialog.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("billing.document.email.dialog.description")}
					</DialogDescription>
				</DialogHeader>
				<Form
					schema={formSchema}
					defaultValues={{ email: initialEmail }}
					onSubmit={onSubmit}
				>
					{(field) => (
						<>
							<FormInput
								field={field}
								name="email"
								type="email"
								label={textGet("billing.document.email.field.label")}
								placeholder={textGet(
									"billing.document.email.field.placeholder",
								)}
							/>
							<DialogFooter className="mt-6">
								<Button
									type="button"
									variant="outline"
									onClick={() => onOpenChange(false)}
								>
									{textGet("common.cancel")}
								</Button>
								<Button type="submit" disabled={sending}>
									{sending && <Spinner />}
									{textGet("billing.document.email.submit")}
								</Button>
							</DialogFooter>
						</>
					)}
				</Form>
			</DialogContent>
		</Dialog>
	);
}

/**
 * "Enviar / Reenviar por correo" row action of an authorized comprobante
 * (UPDATE_BILLING only). The dialog is prefilled with the address the document
 * was last sent to, else `buyerEmail`. API errors are toasted by the service.
 */
export function SriDocumentEmailAction({
	doc,
	buyerEmail,
	send,
	onSent,
}: {
	doc: SriDocumentEmailState;
	buyerEmail?: string | null;
	send: (email: string) => Promise<ServiceResponse<SriDocumentEmailResult>>;
	onSent?: () => void;
}) {
	const { textGet } = useText();
	const { checkPermission } = usePermission();
	const [open, setOpen] = useState(false);

	if (
		doc.status !== "authorized" ||
		!checkPermission([PERMISSIONS.BILLING.PERMISSION_UPDATE_BILLING])
	) {
		return null;
	}

	return (
		<>
			<Button variant="ghost" size="sm" onClick={() => setOpen(true)}>
				<Mail className="mr-2 h-4 w-4" />
				{textGet(
					doc.emailed_at
						? "billing.document.email.resend"
						: "billing.document.email.send",
				)}
			</Button>
			{open && (
				<SriDocumentEmailDialog
					open={open}
					onOpenChange={setOpen}
					initialEmail={doc.emailed_to || buyerEmail || ""}
					send={send}
					onSent={() => onSent?.()}
				/>
			)}
		</>
	);
}
