import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Form,
	FormInput,
	FormPasswordInput,
	Text,
} from "@pengi/ui";
import { Loader2 } from "lucide-react";
import React from "react";
import { z } from "zod";
import {
	connectWhatsAppManual,
	type WhatsAppAccount,
} from "@/api/whatsapp-service";

const manualSchema = z.object({
	waba_id: z.string().trim().min(1, "settings.whatsapp.form.error.required"),
	phone_number_id: z
		.string()
		.trim()
		.min(1, "settings.whatsapp.form.error.required"),
	access_token: z
		.string()
		.trim()
		.min(1, "settings.whatsapp.form.error.required"),
});

type ManualValues = z.infer<typeof manualSchema>;

interface ConnectManualDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onConnected: (account: WhatsAppAccount) => void;
}

/** Connects WhatsApp with the ids and token copied from Meta's dashboard. */
export function ConnectManualDialog({
	open,
	onOpenChange,
	onConnected,
}: ConnectManualDialogProps) {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(false);

	async function onSubmit(values: ManualValues) {
		setLoading(true);
		const res = await connectWhatsAppManual(values);
		setLoading(false);
		if (res.success && res.data) {
			onOpenChange(false);
			onConnected(res.data);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[525px]">
				<DialogHeader>
					<DialogTitle>
						<Text uuid="settings.whatsapp.manual.title" />
					</DialogTitle>
					<DialogDescription>
						<Text uuid="settings.whatsapp.manual.description" />
					</DialogDescription>
				</DialogHeader>
				<Form
					schema={manualSchema}
					defaultValues={{ waba_id: "", phone_number_id: "", access_token: "" }}
					onSubmit={onSubmit}
				>
					{(field) => (
						<div className="space-y-4">
							<FormInput
								field={field}
								name="waba_id"
								label={textGet("settings.whatsapp.form.waba_id")}
								autoComplete="off"
							/>
							<FormInput
								field={field}
								name="phone_number_id"
								label={textGet("settings.whatsapp.form.phone_number_id")}
								autoComplete="off"
							/>
							<FormPasswordInput
								field={field}
								name="access_token"
								label={textGet("settings.whatsapp.form.access_token")}
								description={textGet(
									"settings.whatsapp.form.access_token.description",
								)}
								autoComplete="off"
							/>
							<DialogFooter>
								<Button type="submit" disabled={loading}>
									{loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
									<Text uuid="settings.whatsapp.manual.submit" />
								</Button>
							</DialogFooter>
						</div>
					)}
				</Form>
			</DialogContent>
		</Dialog>
	);
}
