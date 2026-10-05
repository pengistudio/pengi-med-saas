import { useText } from "@pengi/shared";
import { Loader2 } from "lucide-react";
import type { PDFDocumentLoadingTask } from "pdfjs-dist";
import workerUrl from "pdfjs-dist/build/pdf.worker.min.mjs?url";
import React from "react";

interface PdfPagesProps {
	blob: Blob;
}

/**
 * Renders every page of a PDF to a canvas, scaled to the container width.
 * No scripts, annotations or text layers: pdf.js only draws the pages.
 * Loaded lazily (see attachment-viewer) so pdf.js stays out of the main bundle.
 */
export default function PdfPages({ blob }: PdfPagesProps) {
	const { textGet } = useText();
	const containerRef = React.useRef<HTMLDivElement>(null);
	const [status, setStatus] = React.useState<"loading" | "ready" | "error">(
		"loading",
	);

	React.useEffect(() => {
		const container = containerRef.current;
		if (!container) return;
		let cancelled = false;
		let task: PDFDocumentLoadingTask | null = null;

		(async () => {
			try {
				const pdfjs = await import("pdfjs-dist");
				pdfjs.GlobalWorkerOptions.workerSrc = workerUrl;
				const data = await blob.arrayBuffer();
				if (cancelled) return;
				task = pdfjs.getDocument({ data });
				const doc = await task.promise;
				if (cancelled) return;
				const width = container.clientWidth || 800;
				const pixelRatio = window.devicePixelRatio || 1;
				for (let n = 1; n <= doc.numPages; n++) {
					const page = await doc.getPage(n);
					if (cancelled) return;
					const base = page.getViewport({ scale: 1 });
					const viewport = page.getViewport({
						scale: (width / base.width) * pixelRatio,
					});
					const canvas = document.createElement("canvas");
					canvas.width = viewport.width;
					canvas.height = viewport.height;
					canvas.className = "mb-3 h-auto w-full bg-white shadow";
					container.appendChild(canvas);
					await page.render({ canvas, viewport }).promise;
					if (n === 1) setStatus("ready");
				}
				setStatus("ready");
			} catch {
				if (!cancelled) setStatus("error");
			}
		})();

		return () => {
			cancelled = true;
			container.replaceChildren();
			void task?.destroy();
		};
	}, [blob]);

	return (
		<div className="h-full w-full overflow-auto p-2">
			{status === "loading" && (
				<div className="flex justify-center py-8">
					<Loader2
						className="h-6 w-6 animate-spin text-muted-foreground"
						aria-label={textGet("clinical.attachment.viewer.loading")}
					/>
				</div>
			)}
			{status === "error" && (
				<p
					role="alert"
					className="py-8 text-center text-sm text-muted-foreground"
				>
					{textGet("clinical.attachment.viewer.error")}
				</p>
			)}
			<div ref={containerRef} data-testid="pdf-pages" />
		</div>
	);
}
