import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import { AlertCircle, Download, Loader2 } from "lucide-react";
import React from "react";
import {
	type PatientAttachment,
	viewPatientAttachment,
} from "@/api/patient-attachment-service";

interface AttachmentViewerProps {
	patientId: number;
	/** The attachment to show; null keeps the dialog closed. */
	attachment: PatientAttachment | null;
	onClose: () => void;
	onDownload: (attachment: PatientAttachment) => void;
}

// pdf.js (and its worker) load only when a PDF is opened.
const PdfPages = React.lazy(() => import("./pdf-pages"));

type ViewState =
	| { status: "loading" }
	| { status: "error" }
	| { status: "ready"; url: string | null; blob: Blob };

/**
 * Shows an image or PDF inside the app. The file is fetched through the service
 * (auth + tenant headers) as a blob and shown from an object URL, revoked when
 * the dialog closes. PDFs are drawn page by page on canvases with pdf.js.
 */
export function AttachmentViewer({
	patientId,
	attachment,
	onClose,
	onDownload,
}: AttachmentViewerProps) {
	const { textGet } = useText();
	const [state, setState] = React.useState<ViewState>({ status: "loading" });
	const attachmentId = attachment?.ID;

	React.useEffect(() => {
		if (attachmentId === undefined) return;
		let cancelled = false;
		let objectUrl: string | null = null;
		setState({ status: "loading" });
		viewPatientAttachment(patientId, attachmentId).then((res) => {
			if (cancelled) return;
			if (!res.success || !res.data) {
				setState({ status: "error" });
				return;
			}
			// Only images need an object URL; PDFs are drawn from the blob.
			if (res.data.type.startsWith("image/")) {
				objectUrl = window.URL.createObjectURL(res.data);
			}
			setState({ status: "ready", url: objectUrl, blob: res.data });
		});
		return () => {
			cancelled = true;
			if (objectUrl) window.URL.revokeObjectURL(objectUrl);
		};
	}, [patientId, attachmentId]);

	return (
		<Dialog
			open={attachment !== null}
			onOpenChange={(open) => {
				if (!open) onClose();
			}}
		>
			<DialogContent className="flex h-[85vh] flex-col sm:max-w-4xl">
				<DialogHeader>
					<DialogTitle className="truncate pr-8">
						{attachment?.file_name}
					</DialogTitle>
					<DialogDescription>
						{textGet("clinical.attachment.viewer.description")}
					</DialogDescription>
				</DialogHeader>
				<div className="flex min-h-0 flex-1 items-center justify-center overflow-auto rounded-md border bg-muted/30">
					{state.status === "loading" && (
						<Loader2
							className="h-6 w-6 animate-spin text-muted-foreground"
							aria-label={textGet("clinical.attachment.viewer.loading")}
						/>
					)}
					{state.status === "error" && (
						<div
							role="alert"
							className="flex flex-col items-center gap-2 text-sm text-muted-foreground"
						>
							<AlertCircle className="h-8 w-8" />
							{textGet("clinical.attachment.viewer.error")}
						</div>
					)}
					{state.status === "ready" &&
						(state.url ? (
							<img
								src={state.url}
								alt={attachment?.file_name ?? ""}
								className="max-h-full max-w-full object-contain"
							/>
						) : (
							<React.Suspense
								fallback={
									<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
								}
							>
								<PdfPages blob={state.blob} />
							</React.Suspense>
						))}
				</div>
				<div className="flex justify-end">
					<Button
						type="button"
						variant="outline"
						size="sm"
						disabled={!attachment}
						onClick={() => attachment && onDownload(attachment)}
					>
						<Download className="mr-2 h-4 w-4" />
						{textGet("clinical.attachment.download")}
					</Button>
				</div>
			</DialogContent>
		</Dialog>
	);
}
