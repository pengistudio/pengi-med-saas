import { Text } from "@pengi/ui";
import { uploadSriSignature } from "@/api/tenant-service";
import { P12UploadForm } from "@/components/forms/p12-upload-form";

export function SriSignatureForm({ onSuccess }: { onSuccess?: () => void }) {
	return (
		<P12UploadForm
			upload={uploadSriSignature}
			onSuccess={() => onSuccess?.()}
			fileLabel={<Text uuid="billing.sri.file.label" />}
			fileDescription={<Text uuid="billing.sri.file.description" />}
			passwordLabel={<Text uuid="billing.sri.password.label" />}
			passwordDescription={<Text uuid="billing.sri.password.description" />}
			submitLabel={<Text uuid="billing.sri.button.save" />}
		/>
	);
}
