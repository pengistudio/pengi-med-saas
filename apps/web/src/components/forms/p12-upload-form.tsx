import { zodResolver } from "@hookform/resolvers/zod";
import { type ServiceResponse, useText } from "@pengi/shared";
import { Button, Input, Label } from "@pengi/ui";
import { Loader2, UploadCloud } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useForm } from "react-hook-form";
import * as z from "zod";
import { FilePicker } from "@/components/custom/file-picker";

const p12Schema = z.object({
	// Sent exactly as typed: CAs may issue passwords with spaces, and a wrong
	// password now gets its own error from the server.
	password: z.string().min(1, { message: "signature.form.password_required" }),
	file: z
		.any()
		.refine((file) => file instanceof File, {
			message: "signature.form.file_required",
		})
		.refine((file) => /\.(p12|pfx)$/i.test(file?.name ?? ""), {
			message: "signature.form.file_type",
		}),
});

/** Server errors shown next to the field the user has to fix. */
const FIELD_BY_ERROR_CODE: Record<string, "file" | "password"> = {
	"E-SIGN-007": "password", // wrong password
	"E-SIGN-001": "file", // invalid file
	"E-SIGN-002": "file", // expired
	"E-SIGN-008": "file", // not valid yet
	"E-BILL-005": "file", // invalid SRI signature file
};

interface P12UploadFormProps<T> {
	/** Sends the certificate and its password; toasts belong to the service. */
	upload: (file: File, password: string) => Promise<ServiceResponse<T>>;
	onSuccess?: (data: T | undefined) => void;
	fileLabel: ReactNode;
	fileDescription: ReactNode;
	passwordLabel: ReactNode;
	passwordDescription: ReactNode;
	submitLabel: ReactNode;
}

/** Upload form for a .p12 electronic signature certificate and its password. */
export function P12UploadForm<T>({
	upload,
	onSuccess,
	fileLabel,
	fileDescription,
	passwordLabel,
	passwordDescription,
	submitLabel,
}: P12UploadFormProps<T>) {
	const { textGet } = useText();
	const [loading, setLoading] = useState(false);

	const form = useForm<z.infer<typeof p12Schema>>({
		resolver: zodResolver(p12Schema),
		defaultValues: { password: "", file: undefined },
	});

	async function onSubmit(values: z.infer<typeof p12Schema>) {
		setLoading(true);
		try {
			const response = await upload(values.file, values.password);
			if (response.success) {
				form.reset();
				onSuccess?.(response.data);
				return;
			}
			// Every error code is also an i18n key, rendered by the field below.
			const code = response.data?.error_code;
			const field = FIELD_BY_ERROR_CODE[code];
			if (field) form.setError(field, { message: code });
		} finally {
			setLoading(false);
		}
	}

	return (
		<form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
			<div className="space-y-2">
				<Label>{fileLabel}</Label>
				<FilePicker
					accept=".p12,.pfx"
					file={form.watch("file")}
					aria-invalid={!!form.formState.errors.file}
					onChange={(file) => {
						form.setValue("file", file ?? undefined);
						form.clearErrors("file");
					}}
				/>
				<p className="text-sm text-muted-foreground">{fileDescription}</p>
				{form.formState.errors.file && (
					<p className="text-sm font-medium text-destructive">
						{textGet(form.formState.errors.file.message as string)}
					</p>
				)}
			</div>

			<div className="space-y-2">
				<Label>{passwordLabel}</Label>
				<Input
					type="password"
					placeholder="••••••••"
					aria-invalid={!!form.formState.errors.password}
					{...form.register("password")}
				/>
				<p className="text-sm text-muted-foreground">{passwordDescription}</p>
				{form.formState.errors.password && (
					<p className="text-sm font-medium text-destructive">
						{textGet(form.formState.errors.password.message as string)}
					</p>
				)}
			</div>

			<Button type="submit" className="w-fit" disabled={loading}>
				{loading ? (
					<Loader2 className="mr-2 h-4 w-4 animate-spin" />
				) : (
					<UploadCloud className="mr-2 h-4 w-4" />
				)}
				{submitLabel}
			</Button>
		</form>
	);
}
