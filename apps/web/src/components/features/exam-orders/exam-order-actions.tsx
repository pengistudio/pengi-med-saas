import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import {
	Ban,
	CheckCircle2,
	Download,
	Loader2,
	MessageCircle,
	Pencil,
} from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	closeExamOrder,
	downloadExamOrderPdf,
	type ExamOrder,
	emailExamOrder,
	signExamOrder,
	voidExamOrder,
} from "@/api/exam-order-service";
import { SendEmailPopover } from "@/components/custom/send-email-popover";
import { SignDocumentButton } from "@/components/custom/sign-document-button";
import { buildExamOrderWhatsAppMessage } from "@/lib/exam-order-whatsapp";
import { isOrderVoided, patientDisplayName } from "@/lib/exam-orders";
import { generateWhatsAppLink } from "@/lib/utils";
import { ExamOrderVoidForm } from "@/sections/forms/clinical/exam-order-form";
import { ConfirmActionDialog } from "./confirm-action-dialog";

/** Opens the order's PDF in a new tab. */
async function openOrderPdf(orderId: number) {
	const res = await downloadExamOrderPdf(orderId);
	if (!res.success || !res.data) return;
	const url = window.URL.createObjectURL(res.data);
	window.open(url, "_blank", "noopener,noreferrer");
	// The new tab has loaded the blob by then.
	setTimeout(() => window.URL.revokeObjectURL(url), 60_000);
}

function VoidOrderDialog({
	open,
	onOpenChange,
	onVoid,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	/** Resolves once the request finished. */
	onVoid: (reason: string) => Promise<void>;
}) {
	const { textGet } = useText();
	const [voiding, setVoiding] = React.useState(false);

	async function submit(reason: string) {
		setVoiding(true);
		try {
			await onVoid(reason.trim());
		} finally {
			setVoiding(false);
		}
	}

	return (
		<Dialog
			open={open}
			onOpenChange={(next) => {
				if (!next && !voiding) onOpenChange(false);
			}}
		>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>
						{textGet("clinical.exam_orders.void.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("clinical.exam_orders.void.description")}
					</DialogDescription>
				</DialogHeader>
				<ExamOrderVoidForm
					loading={voiding}
					onSubmit={(values) => submit(values.reason)}
				/>
			</DialogContent>
		</Dialog>
	);
}

interface ExamOrderActionsProps {
	order: ExamOrder;
	onChange: (order: ExamOrder) => void;
	/** CREATE_EXAM_ORDER: edit, sign, email, close and void. */
	canManage: boolean;
}

/**
 * The order's actions: edit, download PDF, sign, email, WhatsApp, close as
 * complete and void with a reason. A voided order only downloads (and shows
 * its signature, if any).
 */
export function ExamOrderActions({
	order,
	onChange,
	canManage,
}: ExamOrderActionsProps) {
	const { textGet, formatDate } = useText();
	const navigate = useNavigate();
	const [downloading, setDownloading] = React.useState(false);
	const [voidOpen, setVoidOpen] = React.useState(false);
	const editable = canManage && !isOrderVoided(order);
	const phone = isOrderVoided(order) ? undefined : order.patient?.phone;

	async function handleDownload() {
		setDownloading(true);
		try {
			await openOrderPdf(order.ID);
		} finally {
			setDownloading(false);
		}
	}

	async function handleSign() {
		const res = await signExamOrder(order.ID);
		if (res.success && res.data) onChange(res.data);
		return res.success;
	}

	async function handleClose() {
		const res = await closeExamOrder(order.ID);
		if (res.success && res.data) onChange(res.data);
	}

	async function handleVoid(reason: string) {
		const res = await voidExamOrder(order.ID, reason);
		if (res.success && res.data) {
			setVoidOpen(false);
			onChange(res.data);
		}
	}

	function handleWhatsApp() {
		if (!phone) return;
		const message = buildExamOrderWhatsAppMessage(
			{
				code: order.code,
				patientName: patientDisplayName(order.patient),
				doctorName: order.ordered_by_name || undefined,
				date: formatDate(order.CreatedAt),
				urgent: order.priority === "urgent",
				items: order.items.map((item) => ({
					name: item.name,
					indications: item.indications,
				})),
				destinationLab: order.destination_lab,
				notes: order.notes,
			},
			textGet,
		);
		window.open(
			generateWhatsAppLink(phone, message),
			"_blank",
			"noopener,noreferrer",
		);
	}

	return (
		<div className="flex flex-wrap items-center gap-2">
			{editable && (
				<Button
					variant="outline"
					onClick={() => navigate(`/clinical/exam-orders/${order.ID}/edit`)}
				>
					<Pencil className="mr-2 h-4 w-4" />
					{textGet("clinical.exam_orders.action.edit")}
				</Button>
			)}
			<Button variant="outline" onClick={handleDownload} disabled={downloading}>
				{downloading ? (
					<Loader2 className="mr-2 h-4 w-4 animate-spin" />
				) : (
					<Download className="mr-2 h-4 w-4" />
				)}
				{textGet("clinical.exam_orders.action.download")}
			</Button>
			{(order.signed_at || editable) && (
				<SignDocumentButton signature={order} onSign={handleSign} />
			)}
			{editable && (
				<SendEmailPopover
					defaultEmail={order.patient?.email ?? ""}
					onSend={async (email) => {
						await emailExamOrder(order.ID, email);
					}}
					label={textGet("clinical.exam_orders.action.email")}
				/>
			)}
			{phone && (
				<Button variant="outline" onClick={handleWhatsApp}>
					<MessageCircle className="mr-2 h-4 w-4" />
					{textGet("clinical.exam_orders.action.whatsapp")}
				</Button>
			)}
			{editable && order.status !== "complete_results" && (
				<ConfirmActionDialog
					trigger={
						<Button variant="outline">
							<CheckCircle2 className="mr-2 h-4 w-4" />
							{textGet("clinical.exam_orders.action.close")}
						</Button>
					}
					title={textGet("clinical.exam_orders.close.title")}
					description={textGet("clinical.exam_orders.close.description")}
					onConfirm={handleClose}
				/>
			)}
			{editable && (
				<Button
					variant="outline"
					className="text-destructive"
					onClick={() => setVoidOpen(true)}
				>
					<Ban className="mr-2 h-4 w-4" />
					{textGet("clinical.exam_orders.action.void")}
				</Button>
			)}
			<VoidOrderDialog
				open={voidOpen}
				onOpenChange={setVoidOpen}
				onVoid={handleVoid}
			/>
		</div>
	);
}
