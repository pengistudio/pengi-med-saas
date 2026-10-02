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
	Text,
} from "@pengi/ui";
import { Download, Eye, Info, RotateCcw, Upload } from "lucide-react";
import React from "react";
import {
	type DocumentTemplate,
	type DocumentTemplateId,
	deleteDocumentTemplate,
	downloadCustomDocumentTemplate,
	downloadDefaultDocumentTemplate,
	getDocumentTemplates,
	previewDocumentTemplate,
	uploadDocumentTemplate,
} from "@/api/document-template-service";

function saveBlob(blob: Blob, filename: string) {
	const url = window.URL.createObjectURL(blob);
	const link = document.createElement("a");
	link.href = url;
	link.download = filename;
	document.body.appendChild(link);
	link.click();
	document.body.removeChild(link);
	window.URL.revokeObjectURL(url);
}

/**
 * One row per printable document of the tenant's plan: download the default or
 * the custom template, upload a new one, preview it with sample data, restore
 * the default.
 */
export function DocumentTemplatesSettings() {
	const { textGet } = useText();
	const [templates, setTemplates] = React.useState<DocumentTemplate[] | null>(
		null,
	);
	const [busyId, setBusyId] = React.useState<DocumentTemplateId | null>(null);
	const [restoreTarget, setRestoreTarget] =
		React.useState<DocumentTemplateId | null>(null);
	const uploadTarget = React.useRef<DocumentTemplateId | null>(null);
	const fileInputRef = React.useRef<HTMLInputElement>(null);

	React.useEffect(() => {
		getDocumentTemplates().then((res) => {
			if (res.success) setTemplates(res.data ?? []);
		});
	}, []);

	function replaceTemplate(updated: DocumentTemplate) {
		setTemplates(
			(current) =>
				current?.map((t) => (t.id === updated.id ? updated : t)) ?? null,
		);
	}

	async function run(id: DocumentTemplateId, action: () => Promise<void>) {
		setBusyId(id);
		await action();
		setBusyId(null);
	}

	function handleDownloadDefault(id: DocumentTemplateId) {
		run(id, async () => {
			const res = await downloadDefaultDocumentTemplate(id);
			if (res.success && res.data)
				saveBlob(res.data, res.filename ?? `${id}.html`);
		});
	}

	function handleDownloadCustom(id: DocumentTemplateId) {
		run(id, async () => {
			const res = await downloadCustomDocumentTemplate(id);
			if (res.success && res.data)
				saveBlob(res.data, res.filename ?? `custom_${id}.html`);
		});
	}

	function handlePreview(id: DocumentTemplateId) {
		run(id, async () => {
			const res = await previewDocumentTemplate(id);
			if (res.success && res.data) {
				window.open(window.URL.createObjectURL(res.data), "_blank");
			}
		});
	}

	function handlePickFile(id: DocumentTemplateId) {
		uploadTarget.current = id;
		fileInputRef.current?.click();
	}

	async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
		const file = e.target.files?.[0];
		const id = uploadTarget.current;
		if (fileInputRef.current) fileInputRef.current.value = "";
		if (!file || !id) return;
		await run(id, async () => {
			const res = await uploadDocumentTemplate(id, file);
			if (res.success && res.data) replaceTemplate(res.data);
		});
	}

	async function handleRestore() {
		const id = restoreTarget;
		if (!id) return;
		await run(id, async () => {
			const res = await deleteDocumentTemplate(id);
			if (res.success && res.data) replaceTemplate(res.data);
		});
		setRestoreTarget(null);
	}

	return (
		<div className="grid gap-3">
			<p className="flex items-start gap-1.5 text-xs text-muted-foreground">
				<Info className="mt-px size-3.5 shrink-0" />
				<Text uuid="settings.document_templates.note" />
			</p>

			<input
				ref={fileInputRef}
				type="file"
				accept=".html,text/html"
				className="hidden"
				onChange={handleUpload}
			/>

			{templates?.length === 0 && (
				<p className="text-sm text-muted-foreground">
					<Text uuid="settings.document_templates.empty" />
				</p>
			)}

			<div className="divide-y">
				{templates?.map((tpl) => {
					const busy = busyId === tpl.id;
					return (
						<div
							key={tpl.id}
							className="flex flex-wrap items-center justify-between gap-3 py-3"
						>
							<div className="grid gap-1">
								<div className="flex items-center gap-2">
									<span className="text-sm font-medium">
										{textGet(`document_templates.doc.${tpl.id}`)}
									</span>
									<Badge variant={tpl.has_custom ? "default" : "secondary"}>
										{tpl.has_custom
											? textGet("settings.document_templates.status.custom")
											: textGet("settings.document_templates.status.default")}
									</Badge>
								</div>
								<span className="text-xs text-muted-foreground">
									{textGet(`document_templates.paper.${tpl.paper}`)}
								</span>
							</div>

							<div className="flex flex-wrap gap-1.5">
								<Button
									variant="ghost"
									size="sm"
									disabled={busy}
									onClick={() => handleDownloadDefault(tpl.id)}
								>
									<Download />
									<Text uuid="settings.document_templates.download_default" />
								</Button>
								{tpl.has_custom && (
									<Button
										variant="ghost"
										size="sm"
										disabled={busy}
										onClick={() => handleDownloadCustom(tpl.id)}
									>
										<Download />
										<Text uuid="settings.document_templates.download_custom" />
									</Button>
								)}
								<Button
									variant="ghost"
									size="sm"
									disabled={busy}
									onClick={() => handlePreview(tpl.id)}
								>
									<Eye />
									<Text uuid="settings.document_templates.preview" />
								</Button>
								{tpl.has_custom && (
									<Button
										variant="ghost"
										size="sm"
										className="text-destructive hover:text-destructive"
										disabled={busy}
										onClick={() => setRestoreTarget(tpl.id)}
									>
										<RotateCcw />
										<Text uuid="settings.document_templates.restore" />
									</Button>
								)}
								<Button
									variant="outline"
									size="sm"
									disabled={busy}
									onClick={() => handlePickFile(tpl.id)}
								>
									<Upload />
									<Text uuid="settings.document_templates.upload" />
								</Button>
							</div>
						</div>
					);
				})}
			</div>

			<AlertDialog
				open={restoreTarget !== null}
				onOpenChange={(open) => {
					if (!open) setRestoreTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							<Text uuid="settings.document_templates.restore_confirm.title" />
						</AlertDialogTitle>
						<AlertDialogDescription>
							<Text uuid="settings.document_templates.restore_confirm.description" />
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>
							<Text uuid="common.cancel" />
						</AlertDialogCancel>
						<AlertDialogAction
							variant="destructive"
							disabled={busyId !== null}
							onClick={handleRestore}
						>
							<Text uuid="settings.document_templates.restore_confirm.action" />
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
