import { Button, Label, Text } from "@pengi/ui";
import { Loader2, UploadCloud } from "lucide-react";
import type React from "react";
import { useEffect, useState } from "react";
import { getLogo, uploadLogo } from "@/api/tenant-service";
import { FilePicker } from "@/components/custom/file-picker";

export function LogoUploadForm({
	hasLogo,
	onSuccess,
}: {
	hasLogo: boolean;
	onSuccess?: () => void;
}) {
	const [file, setFile] = useState<File | null>(null);
	const [loading, setLoading] = useState(false);
	const [previewUrl, setPreviewUrl] = useState<string | null>(null);

	// Drop the preview once the logo is gone (adjust state during render).
	if (!hasLogo && previewUrl !== null) setPreviewUrl(null);

	useEffect(() => {
		if (!hasLogo) return;

		let objectUrl: string | null = null;
		getLogo().then((res) => {
			if (res.success && res.data) {
				objectUrl = window.URL.createObjectURL(res.data);
				setPreviewUrl(objectUrl);
			}
		});

		return () => {
			if (objectUrl) window.URL.revokeObjectURL(objectUrl);
		};
	}, [hasLogo]);

	async function handleSubmit(event: React.FormEvent) {
		event.preventDefault();
		if (!file) return;

		setLoading(true);
		try {
			const res = await uploadLogo(file);
			if (res.success) {
				setFile(null);
				onSuccess?.();
			}
		} finally {
			setLoading(false);
		}
	}

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			{previewUrl && (
				<div className="space-y-1">
					<p className="text-sm text-muted-foreground">
						<Text uuid="billing.sri.logo.current" />
					</p>
					<img
						src={previewUrl}
						alt="Logo"
						className="h-16 max-w-[200px] rounded border object-contain"
					/>
				</div>
			)}
			<div className="space-y-2">
				<Label>
					<Text uuid="billing.sri.logo.file.label" />
				</Label>
				<FilePicker accept=".png,.jpg,.jpeg" file={file} onChange={setFile} />
			</div>
			<Button type="submit" className="w-fit" disabled={loading || !file}>
				{loading ? (
					<Loader2 className="mr-2 h-4 w-4 animate-spin" />
				) : (
					<UploadCloud className="mr-2 h-4 w-4" />
				)}
				<Text uuid="billing.sri.logo.button.save" />
			</Button>
		</form>
	);
}
