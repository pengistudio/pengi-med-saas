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
	AlertDialogTrigger,
	Badge,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Text,
} from "@pengi/ui";
import { PenLine, Trash } from "lucide-react";
import React from "react";
import {
	deleteMySignature,
	type MySignature,
	uploadMySignature,
} from "@/api/signature-service";
import { P12UploadForm } from "@/components/forms/p12-upload-form";
import { useMySignature, useSignatureStore } from "@/store/signature-store";

function SignatureStatus({ signature }: { signature: MySignature }) {
	const { textGet, formatDate } = useText();
	const notAfter = formatDate(signature.not_after);

	return (
		<div className="rounded-md border p-4 space-y-3">
			<div className="flex flex-wrap items-center gap-2">
				<p className="font-semibold">{signature.subject_name}</p>
				{signature.expired ? (
					<Badge variant="destructive">
						{textGet("signature.status.expired")}
					</Badge>
				) : signature.expiring_soon ? (
					<Badge variant="outline" className="border-amber-500 text-amber-600">
						{textGet("signature.status.expiring_soon")}
					</Badge>
				) : (
					<Badge
						variant="outline"
						className="border-green-600/40 text-green-700 dark:text-green-400"
					>
						{textGet("signature.status.active")}
					</Badge>
				)}
			</div>
			<div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-sm">
				{signature.subject_serial && (
					<div>
						<span className="text-muted-foreground">
							{textGet("signature.subject_serial")}:
						</span>{" "}
						{signature.subject_serial}
					</div>
				)}
				<div>
					<span className="text-muted-foreground">
						{textGet("signature.issuer")}:
					</span>{" "}
					{signature.issuer}
				</div>
				<div>
					<span className="text-muted-foreground">
						{textGet("signature.not_after")}:
					</span>{" "}
					{notAfter}
				</div>
			</div>
		</div>
	);
}

/** Profile card where the user uploads, replaces or removes their P12. */
export function ElectronicSignatureCard() {
	const signature = useMySignature();
	const setSignature = useSignatureStore((s) => s.setSignature);
	const [replacing, setReplacing] = React.useState(false);

	async function handleDelete() {
		const res = await deleteMySignature();
		if (res.success && res.data) setSignature(res.data);
	}

	const showForm = !signature?.configured || replacing;

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2">
					<PenLine className="h-5 w-5" />
					<Text uuid="signature.title" />
				</CardTitle>
				<CardDescription>
					<Text uuid="signature.description" />
				</CardDescription>
			</CardHeader>
			<CardContent className="space-y-4">
				{signature?.configured && <SignatureStatus signature={signature} />}

				{signature?.configured && !replacing && (
					<div className="flex flex-wrap gap-2">
						<Button variant="outline" onClick={() => setReplacing(true)}>
							<Text uuid="signature.replace" />
						</Button>
						<AlertDialog>
							<AlertDialogTrigger
								render={
									<Button variant="ghost" className="text-destructive">
										<Trash className="mr-2 h-4 w-4" />
										<Text uuid="signature.delete" />
									</Button>
								}
							/>
							<AlertDialogContent>
								<AlertDialogHeader>
									<AlertDialogTitle>
										<Text uuid="signature.delete.title" />
									</AlertDialogTitle>
									<AlertDialogDescription>
										<Text uuid="signature.delete.description" />
									</AlertDialogDescription>
								</AlertDialogHeader>
								<AlertDialogFooter>
									<AlertDialogCancel>
										<Text uuid="form.cancel" />
									</AlertDialogCancel>
									<AlertDialogAction onClick={handleDelete}>
										<Text uuid="form.continue" />
									</AlertDialogAction>
								</AlertDialogFooter>
							</AlertDialogContent>
						</AlertDialog>
					</div>
				)}

				{signature && showForm && (
					<P12UploadForm
						upload={uploadMySignature}
						onSuccess={(data) => {
							if (data) setSignature(data);
							setReplacing(false);
						}}
						fileLabel={<Text uuid="signature.file.label" />}
						fileDescription={<Text uuid="signature.file.description" />}
						passwordLabel={<Text uuid="signature.password.label" />}
						passwordDescription={<Text uuid="signature.password.description" />}
						submitLabel={<Text uuid="signature.save" />}
					/>
				)}
			</CardContent>
		</Card>
	);
}
