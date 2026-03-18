import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, UploadCloud } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import * as z from "zod";
import { uploadSriSignature } from "@/api/tenant-service";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";

const sriSchema = z.object({
	password: z
		.string()
		.min(1)
		.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
	file: z.any().refine((file) => file instanceof File),
});

export function SriSignatureForm({ onSuccess }: { onSuccess?: () => void }) {
	const { textGet } = useText();

	const [loading, setLoading] = useState(false);

	const form = useForm<z.infer<typeof sriSchema>>({
		resolver: zodResolver(sriSchema),
		defaultValues: {
			password: "",
			file: undefined,
		},
	});

	async function onSubmit(values: z.infer<typeof sriSchema>) {
		setLoading(true);
		try {
			const response = await uploadSriSignature(values.file, values.password);
			if (response.success) {
				form.reset();
				if (onSuccess) onSuccess();
			}
		} finally {
			setLoading(false);
		}
	}

	return (
		<form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
			<div className="space-y-2">
				<Label>
					<Text uuid="billing.sri.file.label" />
				</Label>
				<Input
					type="file"
					accept=".p12"
					onChange={(event) => form.setValue("file", event.target.files?.[0])}
				/>
				<p className="text-sm text-muted-foreground">
					<Text uuid="billing.sri.file.description" />
				</p>
				{form.formState.errors.file && (
					<p className="text-sm font-medium text-destructive">
						{textGet(form.formState.errors.file.message as string)}
					</p>
				)}
			</div>

			<div className="space-y-2">
				<Label>
					<Text uuid="billing.sri.password.label" />
				</Label>
				<Input
					type="password"
					placeholder="••••••••"
					{...form.register("password")}
				/>
				<p className="text-sm text-muted-foreground">
					<Text uuid="billing.sri.password.description" />
				</p>
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
				<Text uuid="billing.sri.button.save" />
			</Button>
		</form>
	);
}
