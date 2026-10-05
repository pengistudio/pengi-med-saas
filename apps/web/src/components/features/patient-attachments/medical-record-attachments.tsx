import { parseDateOnly, toDateOnlyString, useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import { Eye, FileUp, Link2, Loader2, Paperclip, Unlink } from "lucide-react";
import React from "react";
import {
	getPatientAttachments,
	linkPatientAttachment,
	type PatientAttachment,
} from "@/api/patient-attachment-service";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import PatientAttachmentForm, {
	type PatientAttachmentFormValues,
} from "@/sections/forms/clinical/patient-attachment-form";
import { saveAttachmentFile, toastBatchResult } from "./attachment-actions";
import { AttachmentViewer } from "./attachment-viewer";
import { type UploadItem, uploadBatch } from "./upload-batch";
import { UploadProgressList } from "./upload-progress-list";

interface MedicalRecordAttachmentsProps {
	patientId: number;
	/**
	 * The saved consultation. Uploads and links go straight to it. Without it
	 * (a consultation still being written) files are uploaded unlinked and kept
	 * as pending, for the form to link them once the consultation is saved.
	 */
	medicalRecordId?: number;
	/** Pending mode: the files to link, held by the form (survives tab switches). */
	pendingAttachments?: PatientAttachment[];
	onPendingChange?: (attachments: PatientAttachment[]) => void;
}

/**
 * The "Adjuntos" section of a consultation: upload files linked to it, see the
 * linked ones, open them, unlink them, and link an existing unlinked file of
 * the same patient. Hidden without attachment permissions.
 */
export function MedicalRecordAttachments({
	patientId,
	medicalRecordId,
	pendingAttachments,
	onPendingChange,
}: MedicalRecordAttachmentsProps) {
	const { textGet, formatDate } = useText();
	const { checkPermission } = usePermission();
	const canRead = checkPermission([
		PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_PATIENT_ATTACHMENT,
	]);
	// Linking and unlinking are metadata updates: same permission as upload.
	const canUpload = checkPermission([
		PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPLOAD_PATIENT_ATTACHMENT,
	]);
	const pending = medicalRecordId === undefined;

	const [linked, setLinked] = React.useState<PatientAttachment[]>([]);
	const items = pending ? (pendingAttachments ?? []) : linked;
	const [loading, setLoading] = React.useState(!pending && canRead);
	const [uploadOpen, setUploadOpen] = React.useState(false);
	const [uploading, setUploading] = React.useState(false);
	const [batch, setBatch] = React.useState<UploadItem[] | null>(null);
	const [pickerOpen, setPickerOpen] = React.useState(false);
	const [candidates, setCandidates] = React.useState<PatientAttachment[]>([]);
	const [candidatesLoading, setCandidatesLoading] = React.useState(false);
	const [busyId, setBusyId] = React.useState<number | null>(null);
	const [viewing, setViewing] = React.useState<PatientAttachment | null>(null);

	const load = React.useCallback(() => {
		if (pending || !canRead) return;
		getPatientAttachments(patientId, undefined, medicalRecordId).then((res) => {
			setLinked(res.success ? (res.data?.items ?? []) : []);
			setLoading(false);
		});
	}, [pending, canRead, patientId, medicalRecordId]);

	React.useEffect(() => {
		load();
	}, [load]);

	function setPending(next: PatientAttachment[]) {
		onPendingChange?.(next);
	}

	if (!canRead && !canUpload) return null;

	async function handleUpload(values: PatientAttachmentFormValues) {
		setUploading(true);
		setBatch(
			values.files.map((file, id) => ({
				id,
				name: file.name,
				status: "pending",
				progress: 0,
			})),
		);
		const result = await uploadBatch(
			patientId,
			values.files,
			{
				category: values.category,
				taken_at: values.taken_at
					? toDateOnlyString(values.taken_at)
					: undefined,
				description: values.description?.trim() || undefined,
				medical_record_id: medicalRecordId,
			},
			(id, patch) =>
				setBatch(
					(current) =>
						current?.map((item) =>
							item.id === id ? { ...item, ...patch } : item,
						) ?? null,
				),
		);
		setUploading(false);
		toastBatchResult(result, textGet);
		if (result.uploaded === 0) return;
		if (pending) {
			setPending([...result.attachments, ...items]);
		} else {
			setLoading(true);
			load();
		}
	}

	function closeUpload() {
		if (uploading) return;
		setUploadOpen(false);
		setBatch(null);
	}

	function openPicker() {
		setPickerOpen(true);
		setCandidatesLoading(true);
		getPatientAttachments(patientId).then((res) => {
			const all = res.success ? (res.data?.items ?? []) : [];
			setCandidates(
				all.filter(
					(a) =>
						!a.medical_record_id && !items.some((item) => item.ID === a.ID),
				),
			);
			setCandidatesLoading(false);
		});
	}

	async function handleLink(attachment: PatientAttachment) {
		if (pending) {
			setPending([attachment, ...items]);
			setCandidates((list) => list.filter((a) => a.ID !== attachment.ID));
			return;
		}
		setBusyId(attachment.ID);
		const res = await linkPatientAttachment(
			patientId,
			attachment.ID,
			medicalRecordId,
		);
		setBusyId(null);
		if (res.success) {
			setCandidates((list) => list.filter((a) => a.ID !== attachment.ID));
			setLoading(true);
			load();
		}
	}

	async function handleUnlink(attachment: PatientAttachment) {
		if (pending) {
			// Stays in the patient's files, just not linked to this consultation.
			setPending(items.filter((a) => a.ID !== attachment.ID));
			return;
		}
		setBusyId(attachment.ID);
		const res = await linkPatientAttachment(patientId, attachment.ID, null);
		setBusyId(null);
		if (res.success) {
			setLoading(true);
			load();
		}
	}

	const listed = pending || canRead;

	return (
		// The upload form below is a <form> rendered (through a portal) inside the
		// consultation's own form: keep its submit from bubbling up to it.
		<section
			aria-labelledby="medical-record-attachments-title"
			onSubmit={(e) => e.stopPropagation()}
		>
			<Card className="border-l-4 border-l-slate-400">
				<CardHeader>
					<div className="flex flex-wrap items-center justify-between gap-3">
						<div className="flex items-center gap-3">
							<div className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-500 text-primary-foreground">
								<Paperclip className="h-4 w-4" />
							</div>
							<div>
								<CardTitle
									id="medical-record-attachments-title"
									className="text-base"
								>
									{textGet("clinical.attachment.consultation.title")}
								</CardTitle>
								<CardDescription className="text-xs">
									{textGet(
										pending
											? "clinical.attachment.consultation.pending_hint"
											: "clinical.attachment.consultation.description",
									)}
								</CardDescription>
							</div>
						</div>
						{canUpload && (
							<div className="flex flex-wrap gap-2">
								{canRead && (
									<Button
										type="button"
										size="sm"
										variant="outline"
										onClick={openPicker}
									>
										<Link2 className="mr-2 h-4 w-4" />
										{textGet("clinical.attachment.consultation.link_existing")}
									</Button>
								)}
								<Button
									type="button"
									size="sm"
									onClick={() => setUploadOpen(true)}
								>
									<FileUp className="mr-2 h-4 w-4" />
									{textGet("clinical.attachment.upload")}
								</Button>
							</div>
						)}
					</div>
				</CardHeader>
				<CardContent>
					{!listed ? (
						<p className="py-4 text-center text-sm text-muted-foreground">
							{textGet("clinical.attachment.no_read")}
						</p>
					) : loading ? (
						<div className="flex justify-center py-4">
							<Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
						</div>
					) : items.length === 0 ? (
						<p className="py-4 text-center text-sm text-muted-foreground">
							{textGet("clinical.attachment.consultation.empty")}
						</p>
					) : (
						<ul className="divide-y rounded-md border">
							{items.map((attachment) => (
								<li
									key={attachment.ID}
									className="flex items-center gap-3 px-3 py-2 text-sm"
								>
									<div className="min-w-0 flex-1">
										<p className="truncate font-medium">
											{attachment.file_name}
										</p>
										<p className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
											<Badge variant="secondary">
												{textGet(
													`clinical.attachment.category.${attachment.category}`,
												)}
											</Badge>
											{formatDate(parseDateOnly(attachment.taken_at))}
										</p>
									</div>
									{canRead && (
										<Button
											type="button"
											variant="ghost"
											size="icon"
											aria-label={textGet("clinical.attachment.view")}
											onClick={() => setViewing(attachment)}
										>
											<Eye className="h-4 w-4" />
										</Button>
									)}
									{canUpload && (
										<Button
											type="button"
											variant="ghost"
											size="icon"
											aria-label={textGet(
												"clinical.attachment.consultation.unlink",
											)}
											disabled={busyId === attachment.ID}
											onClick={() => handleUnlink(attachment)}
										>
											{busyId === attachment.ID ? (
												<Loader2 className="h-4 w-4 animate-spin" />
											) : (
												<Unlink className="h-4 w-4" />
											)}
										</Button>
									)}
								</li>
							))}
						</ul>
					)}
				</CardContent>
			</Card>

			<AttachmentViewer
				patientId={patientId}
				attachment={viewing}
				onClose={() => setViewing(null)}
				onDownload={(attachment) => saveAttachmentFile(patientId, attachment)}
			/>

			<Dialog
				open={uploadOpen}
				onOpenChange={(open) => (open ? setUploadOpen(true) : closeUpload())}
			>
				<DialogContent className="sm:max-w-[525px]">
					<DialogHeader>
						<DialogTitle>
							{textGet("clinical.attachment.form.title")}
						</DialogTitle>
						<DialogDescription>
							{textGet("clinical.attachment.form.subtitle")}
						</DialogDescription>
					</DialogHeader>
					{batch ? (
						<div className="space-y-4">
							<UploadProgressList items={batch} />
							<div className="flex justify-end">
								<Button
									type="button"
									disabled={uploading}
									onClick={closeUpload}
								>
									{textGet("clinical.attachment.upload.close")}
								</Button>
							</div>
						</div>
					) : (
						<PatientAttachmentForm
							loading={uploading}
							onSubmit={handleUpload}
						/>
					)}
				</DialogContent>
			</Dialog>

			<Dialog open={pickerOpen} onOpenChange={setPickerOpen}>
				<DialogContent className="sm:max-w-[525px]">
					<DialogHeader>
						<DialogTitle>
							{textGet("clinical.attachment.consultation.picker.title")}
						</DialogTitle>
						<DialogDescription>
							{textGet("clinical.attachment.consultation.picker.description")}
						</DialogDescription>
					</DialogHeader>
					{candidatesLoading ? (
						<div className="flex justify-center py-4">
							<Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
						</div>
					) : candidates.length === 0 ? (
						<p className="py-4 text-center text-sm text-muted-foreground">
							{textGet("clinical.attachment.consultation.picker.empty")}
						</p>
					) : (
						<ul className="max-h-80 divide-y overflow-auto rounded-md border">
							{candidates.map((attachment) => (
								<li
									key={attachment.ID}
									className="flex items-center gap-3 px-3 py-2 text-sm"
								>
									<div className="min-w-0 flex-1">
										<p className="truncate font-medium">
											{attachment.file_name}
										</p>
										<p className="text-xs text-muted-foreground">
											{textGet(
												`clinical.attachment.category.${attachment.category}`,
											)}
											{" · "}
											{formatDate(parseDateOnly(attachment.taken_at))}
										</p>
									</div>
									<Button
										type="button"
										size="sm"
										variant="outline"
										disabled={busyId === attachment.ID}
										onClick={() => handleLink(attachment)}
									>
										{busyId === attachment.ID ? (
											<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										) : (
											<Link2 className="mr-2 h-4 w-4" />
										)}
										{textGet("clinical.attachment.consultation.link")}
									</Button>
								</li>
							))}
						</ul>
					)}
				</DialogContent>
			</Dialog>
		</section>
	);
}
