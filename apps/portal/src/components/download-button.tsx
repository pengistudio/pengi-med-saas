import { useText } from "@pengi/shared";
import { Button, Spinner } from "@pengi/ui";
import { Download } from "lucide-react";
import React from "react";
import { saveBlob } from "@/lib/save-blob";

/**
 * Fetches a file with the Bearer token (the service) and saves it. With a
 * `label` it shows the text (e.g. "PDF"); without one, only the icon.
 */
export function DownloadButton({
	name,
	label,
	fileName,
	fetchFile,
}: {
	/** What is downloaded, for screen readers: "Certificado, 27 sept". */
	name: string;
	label?: string;
	fileName: string;
	fetchFile: () => Promise<{ success: boolean; data?: unknown }>;
}) {
	const { textGet } = useText();
	const [busy, setBusy] = React.useState(false);

	const onClick = async () => {
		setBusy(true);
		const res = await fetchFile();
		setBusy(false);
		if (res.success && res.data instanceof Blob) saveBlob(res.data, fileName);
	};

	const icon = busy ? <Spinner /> : <Download />;
	return (
		<Button
			type="button"
			variant="secondary"
			size={label ? "sm" : "icon"}
			className="rounded-full"
			disabled={busy}
			onClick={onClick}
			aria-label={`${textGet("portal.download")}: ${name}${label ? ` (${label})` : ""}`}
		>
			{icon}
			{label}
		</Button>
	);
}
