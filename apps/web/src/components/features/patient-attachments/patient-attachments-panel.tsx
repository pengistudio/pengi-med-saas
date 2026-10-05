import { parseDateOnly, toDateOnlyString, useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "@pengi/ui";
import {
	ArchiveRestore,
	Download,
	Eye,
	Files,
	FileUp,
	Loader2,
	Pencil,
	Trash2,
} from "lucide-react";
import React from "react";
import { Link } from "react-router";
import {
	ATTACHMENT_CATEGORIES,
	type AttachmentCategory,
	type AttachmentUsage,
	type DeletedPatientAttachment,
	deletePatientAttachment,
	getDeletedPatientAttachments,
	getPatientAttachments,
	isStorageFull,
	type PatientAttachment,
	restorePatientAttachment,
	updatePatientAttachment,
} from "@/api/patient-attachment-service";
import {
	saveAttachmentFile,
	toastBatchResult,
} from "@/components/features/patient-attachments/attachment-actions";
import { AttachmentStorageUsage } from "@/components/features/patient-attachments/attachment-storage-usage";
import { AttachmentViewer } from "@/components/features/patient-attachments/attachment-viewer";
import {
	type UploadItem,
	uploadBatch,
} from "@/components/features/patient-attachments/upload-batch";
import { UploadProgressList } from "@/components/features/patient-attachments/upload-progress-list";
import PatientAttachmentForm, {
	PatientAttachmentDeleteForm,
	type PatientAttachmentDeleteValues,
	PatientAttachmentEditForm,
	type PatientAttachmentEditValues,
	type PatientAttachmentFormValues,
} from "@/sections/forms/clinical/patient-attachment-form";

type CategoryFilter = AttachmentCategory | "all";

interface PatientAttachmentsPanelProps {
	patientId: number;
	canRead: boolean;
	canUpload: boolean;
	/** Delete with a reason, see the deleted ones and restore them. */
	canDelete?: boolean;
}

/** The "Archivos" tab of a patient: upload, list by exam date, filter, download. */
export function PatientAttachmentsPanel({
	patientId,
	canRead,
	canUpload,
	canDelete = false,
}: PatientAttachmentsPanelProps) {
	const { textGet, formatDate, formatDateTime, formatFileSize } = useText();
	const [category, setCategory] = React.useState<CategoryFilter>("all");
	const [attachments, setAttachments] = React.useState<PatientAttachment[]>([]);
	const [usage, setUsage] = React.useState<AttachmentUsage | null>(null);
	const [loading, setLoading] = React.useState(canRead);
	const [uploadOpen, setUploadOpen] = React.useState(false);
	const [uploading, setUploading] = React.useState(false);
	// Progress of the running/last batch; null while the form is showing.
	const [batch, setBatch] = React.useState<UploadItem[] | null>(null);
	const [downloadingId, setDownloadingId] = React.useState<number | null>(null);

	const [editing, setEditing] = React.useState<PatientAttachment | null>(null);
	const [saving, setSaving] = React.useState(false);

	const [viewing, setViewing] = React.useState<PatientAttachment | null>(null);

	const [deleting, setDeleting] = React.useState<PatientAttachment | null>(
		null,
	);
	const [deletingBusy, setDeletingBusy] = React.useState(false);
	const [showDeleted, setShowDeleted] = React.useState(false);
	const [deletedItems, setDeletedItems] = React.useState<
		DeletedPatientAttachment[]
	>([]);
	const [deletedLoading, setDeletedLoading] = React.useState(false);
	const [restoringId, setRestoringId] = React.useState<number | null>(null);

	const loadDeleted = React.useCallback(() => {
		if (!canDelete || !patientId) return;
		getDeletedPatientAttachments(patientId).then((res) => {
			setDeletedItems(res.success ? (res.data ?? []) : []);
			setDeletedLoading(false);
		});
	}, [canDelete, patientId]);

	React.useEffect(() => {
		if (showDeleted) loadDeleted();
	}, [showDeleted, loadDeleted]);

	const load = React.useCallback(() => {
		if (!canRead || !patientId) return;
		// The spinner is turned on by whoever asks for a reload (initial state,
		// filter change, upload), never synchronously here.
		getPatientAttachments(
			patientId,
			category === "all" ? undefined : category,
		).then((res) => {
			setAttachments(res.success ? (res.data?.items ?? []) : []);
			setUsage(res.success ? (res.data?.usage ?? null) : null);
			setLoading(false);
		});
	}, [canRead, patientId, category]);

	React.useEffect(() => {
		load();
	}, [load]);

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
			},
			(id, patch) =>
				setBatch(
					(items) =>
						items?.map((item) =>
							item.id === id ? { ...item, ...patch } : item,
						) ?? null,
				),
		);
		setUploading(false);
		toastBatchResult(result, textGet);
		if (result.uploaded > 0) {
			setLoading(true);
			load();
		}
	}

	function closeUpload() {
		if (uploading) return;
		setUploadOpen(false);
		setBatch(null);
	}

	async function handleEdit(values: PatientAttachmentEditValues) {
		if (!editing) return;
		setSaving(true);
		const res = await updatePatientAttachment(patientId, editing.ID, {
			category: values.category,
			// Required here: the form prefills it, and clearing it keeps the old day.
			taken_at: values.taken_at
				? toDateOnlyString(values.taken_at)
				: editing.taken_at.slice(0, 10),
			description: values.description?.trim() ?? "",
		});
		setSaving(false);
		if (res.success) {
			setEditing(null);
			setLoading(true);
			load();
		}
	}

	async function handleDelete(values: PatientAttachmentDeleteValues) {
		if (!deleting) return;
		setDeletingBusy(true);
		const res = await deletePatientAttachment(
			patientId,
			deleting.ID,
			values.reason.trim(),
		);
		setDeletingBusy(false);
		if (res.success) {
			setDeleting(null);
			setLoading(true);
			load();
		}
	}

	async function handleRestore(attachment: DeletedPatientAttachment) {
		setRestoringId(attachment.ID);
		const res = await restorePatientAttachment(patientId, attachment.ID);
		setRestoringId(null);
		if (res.success) {
			setDeletedLoading(true);
			loadDeleted();
			setLoading(true);
			load();
		}
	}

	async function handleDownload(attachment: PatientAttachment) {
		setDownloadingId(attachment.ID);
		await saveAttachmentFile(patientId, attachment);
		setDownloadingId(null);
	}

	const categoryLabel = (value: CategoryFilter) =>
		value === "all"
			? textGet("clinical.attachment.filter.all")
			: textGet(`clinical.attachment.category.${value}`);

	return (
		<div className="space-y-4">
			<div className="flex flex-wrap items-center gap-2">
				{canRead && !showDeleted && (
					<Select
						value={category}
						onValueChange={(value) => {
							const next = (value as CategoryFilter | null) ?? "all";
							if (next === category) return;
							setLoading(true);
							setCategory(next);
						}}
					>
						<SelectTrigger
							className="w-56"
							aria-label={textGet("clinical.attachment.column.category")}
						>
							<SelectValue>{categoryLabel(category)}</SelectValue>
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="all">{categoryLabel("all")}</SelectItem>
							{ATTACHMENT_CATEGORIES.map((value) => (
								<SelectItem key={value} value={value}>
									{categoryLabel(value)}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				)}
				<div className="ml-auto flex items-center gap-2">
					{canDelete && (
						<Button
							type="button"
							size="sm"
							variant="ghost"
							className="text-muted-foreground"
							onClick={() => {
								setDeletedLoading(!showDeleted);
								setShowDeleted((v) => !v);
							}}
						>
							{showDeleted ? (
								<Files className="mr-2 h-4 w-4" />
							) : (
								<ArchiveRestore className="mr-2 h-4 w-4" />
							)}
							{textGet(
								showDeleted
									? "clinical.attachment.deleted.hide"
									: "clinical.attachment.deleted.show",
							)}
						</Button>
					)}
					{canUpload && !showDeleted && (
						<Button
							type="button"
							size="sm"
							disabled={usage ? isStorageFull(usage) : false}
							onClick={() => setUploadOpen(true)}
						>
							<FileUp className="mr-2 h-4 w-4" />
							{textGet("clinical.attachment.upload")}
						</Button>
					)}
				</div>
			</div>

			{usage && <AttachmentStorageUsage usage={usage} />}

			{showDeleted ? (
				deletedLoading ? (
					<div className="flex justify-center py-8">
						<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
					</div>
				) : deletedItems.length === 0 ? (
					<p className="py-8 text-center text-sm text-muted-foreground">
						{textGet("clinical.attachment.deleted.empty")}
					</p>
				) : (
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>
									{textGet("clinical.attachment.column.file")}
								</TableHead>
								<TableHead>
									{textGet("clinical.attachment.deleted.column.by")}
								</TableHead>
								<TableHead>
									{textGet("clinical.attachment.deleted.column.at")}
								</TableHead>
								<TableHead>
									{textGet("clinical.attachment.deleted.column.reason")}
								</TableHead>
								<TableHead className="text-right">
									{textGet("table.actions")}
								</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{deletedItems.map((item) => (
								<TableRow key={item.ID}>
									<TableCell className="max-w-xs truncate font-medium">
										{item.file_name}
									</TableCell>
									<TableCell>{item.deleted_by_name ?? ""}</TableCell>
									<TableCell className="whitespace-nowrap">
										{item.DeletedAt ? formatDateTime(item.DeletedAt) : ""}
									</TableCell>
									<TableCell className="max-w-xs">
										{item.delete_reason}
									</TableCell>
									<TableCell className="text-right">
										<Button
											type="button"
											variant="ghost"
											size="sm"
											disabled={restoringId === item.ID}
											onClick={() => handleRestore(item)}
										>
											{restoringId === item.ID ? (
												<Loader2 className="mr-2 h-4 w-4 animate-spin" />
											) : (
												<ArchiveRestore className="mr-2 h-4 w-4" />
											)}
											{textGet("clinical.attachment.restore")}
										</Button>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				)
			) : !canRead ? (
				<p className="py-8 text-center text-sm text-muted-foreground">
					{textGet("clinical.attachment.no_read")}
				</p>
			) : loading ? (
				<div className="flex justify-center py-8">
					<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
				</div>
			) : attachments.length === 0 ? (
				category === "all" ? (
					<div className="flex flex-col items-center gap-2 py-10 text-center">
						<Files className="h-10 w-10 text-muted-foreground" />
						<p className="font-medium">
							{textGet("clinical.attachment.empty.title")}
						</p>
						<p className="max-w-md text-sm text-muted-foreground">
							{textGet("clinical.attachment.empty.description")}
						</p>
					</div>
				) : (
					<p className="py-8 text-center text-sm text-muted-foreground">
						{textGet("clinical.attachment.empty.filtered")}
					</p>
				)
			) : (
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>
								{textGet("clinical.attachment.column.taken_at")}
							</TableHead>
							<TableHead>
								{textGet("clinical.attachment.column.category")}
							</TableHead>
							<TableHead>
								{textGet("clinical.attachment.column.file")}
							</TableHead>
							<TableHead>
								{textGet("clinical.attachment.column.size")}
							</TableHead>
							<TableHead className="text-right">
								{textGet("table.actions")}
							</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{attachments.map((attachment) => (
							<TableRow key={attachment.ID}>
								<TableCell className="whitespace-nowrap">
									{formatDate(parseDateOnly(attachment.taken_at))}
								</TableCell>
								<TableCell>
									<Badge variant="secondary">
										{categoryLabel(attachment.category)}
									</Badge>
								</TableCell>
								<TableCell className="max-w-xs">
									<p className="truncate font-medium">{attachment.file_name}</p>
									{attachment.description && (
										<p className="truncate text-xs text-muted-foreground">
											{attachment.description}
										</p>
									)}
									{attachment.medical_record_id &&
										attachment.medical_record_date && (
											<Link
												to={`/clinical/medical-records/view/${attachment.medical_record_id}`}
												className="text-xs text-primary hover:underline"
											>
												{textGet("clinical.attachment.linked_consultation", {
													date: formatDate(attachment.medical_record_date),
												})}
											</Link>
										)}
								</TableCell>
								<TableCell className="whitespace-nowrap">
									{formatFileSize(attachment.size)}
								</TableCell>
								<TableCell className="text-right">
									<Button
										type="button"
										variant="ghost"
										size="icon"
										aria-label={textGet("clinical.attachment.view")}
										onClick={() => setViewing(attachment)}
									>
										<Eye className="h-4 w-4" />
									</Button>
									{canUpload && (
										<Button
											type="button"
											variant="ghost"
											size="icon"
											aria-label={textGet("clinical.attachment.edit")}
											onClick={() => setEditing(attachment)}
										>
											<Pencil className="h-4 w-4" />
										</Button>
									)}
									{canDelete &&
										(attachment.locked_by_review ? (
											<TooltipProvider>
												<Tooltip>
													<TooltipTrigger
														render={<span className="inline-flex" />}
													>
														<Button
															type="button"
															variant="ghost"
															size="icon"
															disabled
															aria-label={textGet(
																"clinical.attachment.delete.locked_by_review",
															)}
														>
															<Trash2 className="h-4 w-4" />
														</Button>
													</TooltipTrigger>
													<TooltipContent>
														{textGet(
															"clinical.attachment.delete.locked_by_review",
														)}
													</TooltipContent>
												</Tooltip>
											</TooltipProvider>
										) : (
											<Button
												type="button"
												variant="ghost"
												size="icon"
												aria-label={textGet("clinical.attachment.delete")}
												onClick={() => setDeleting(attachment)}
											>
												<Trash2 className="h-4 w-4" />
											</Button>
										))}
									<Button
										type="button"
										variant="ghost"
										size="icon"
										aria-label={textGet("clinical.attachment.download")}
										disabled={downloadingId === attachment.ID}
										onClick={() => handleDownload(attachment)}
									>
										{downloadingId === attachment.ID ? (
											<Loader2 className="h-4 w-4 animate-spin" />
										) : (
											<Download className="h-4 w-4" />
										)}
									</Button>
								</TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
			)}

			<AttachmentViewer
				patientId={patientId}
				attachment={viewing}
				onClose={() => setViewing(null)}
				onDownload={handleDownload}
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

			<Dialog
				open={editing !== null}
				onOpenChange={(open) => {
					if (!open) setEditing(null);
				}}
			>
				<DialogContent className="sm:max-w-[525px]">
					<DialogHeader>
						<DialogTitle>
							{textGet("clinical.attachment.edit.title")}
						</DialogTitle>
						<DialogDescription>{editing?.file_name}</DialogDescription>
					</DialogHeader>
					{editing && (
						<PatientAttachmentEditForm
							attachment={editing}
							loading={saving}
							onSubmit={handleEdit}
						/>
					)}
				</DialogContent>
			</Dialog>

			<Dialog
				open={deleting !== null}
				onOpenChange={(open) => {
					if (!open) setDeleting(null);
				}}
			>
				<DialogContent className="sm:max-w-[525px]">
					<DialogHeader>
						<DialogTitle>
							{textGet("clinical.attachment.delete.title")}
						</DialogTitle>
						<DialogDescription>
							{textGet("clinical.attachment.delete.description")}
						</DialogDescription>
					</DialogHeader>
					{deleting && (
						<PatientAttachmentDeleteForm
							loading={deletingBusy}
							onSubmit={handleDelete}
						/>
					)}
				</DialogContent>
			</Dialog>
		</div>
	);
}
