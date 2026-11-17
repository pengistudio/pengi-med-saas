import type { ServiceResponse } from "@pengi/shared";
import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Download, FileCode } from "lucide-react";
import { saveBlob } from "@/pages/reports/report-utils";

type FetchFile = () => Promise<ServiceResponse<Blob>>;

async function save(download: FetchFile, fallbackName: string) {
	const res = await download();
	if (res.success && res.data) saveBlob(res.data, res.filename || fallbackName);
}

/**
 * "Descargar RIDE" / "Descargar XML" actions of an authorized comprobante;
 * renders nothing for any other status. Errors are toasted by the service.
 */
export function SriDocumentDownloads({
	status,
	fileBase,
	accessKey,
	downloadRide,
	downloadXml,
}: {
	status: string;
	/** Fallback RIDE file name without extension, e.g. nota_credito_001-001-000000001. */
	fileBase: string;
	accessKey?: string;
	downloadRide?: FetchFile;
	downloadXml: FetchFile;
}) {
	const { textGet } = useText();
	if (status !== "authorized") return null;

	return (
		<>
			{downloadRide && (
				<Button
					variant="ghost"
					size="sm"
					onClick={() => save(downloadRide, `${fileBase}.pdf`)}
				>
					<Download className="mr-2 h-4 w-4" />
					{textGet("billing.invoice.ride.download")}
				</Button>
			)}
			<Button
				variant="ghost"
				size="sm"
				onClick={() => save(downloadXml, `${accessKey || fileBase}.xml`)}
			>
				<FileCode className="mr-2 h-4 w-4" />
				{textGet("billing.document.xml.download")}
			</Button>
		</>
	);
}
