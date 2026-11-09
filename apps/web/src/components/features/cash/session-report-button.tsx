import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { FileDown, Loader2 } from "lucide-react";
import { useState } from "react";
import { downloadSessionReport } from "@/api/cash-service";

/** Downloads the closing report (PDF) of a closed session. */
export function SessionReportButton({
	sessionId,
	variant = "outline",
	size,
}: {
	sessionId: number;
	variant?: "outline" | "ghost";
	size?: "sm" | "default";
}) {
	const { textGet } = useText();
	const [loading, setLoading] = useState(false);

	async function handleDownload() {
		setLoading(true);
		const res = await downloadSessionReport(sessionId);
		setLoading(false);
		if (!res.success || !res.data) return;
		const url = window.URL.createObjectURL(res.data);
		const link = document.createElement("a");
		link.href = url;
		link.download = res.filename || `cierre_caja_${sessionId}.pdf`;
		document.body.appendChild(link);
		link.click();
		document.body.removeChild(link);
		window.URL.revokeObjectURL(url);
	}

	return (
		<Button
			variant={variant}
			size={size}
			onClick={handleDownload}
			disabled={loading}
		>
			{loading ? (
				<Loader2 className="h-4 w-4 animate-spin" />
			) : (
				<FileDown className="h-4 w-4" />
			)}
			{textGet("cash.session.report.download")}
		</Button>
	);
}
