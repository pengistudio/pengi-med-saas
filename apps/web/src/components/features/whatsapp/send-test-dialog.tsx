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
	Text,
} from "@pengi/ui";
import { Loader2, Send } from "lucide-react";
import React from "react";
import { z } from "zod";
import { sendWhatsAppTest } from "@/api/whatsapp-service";

const testSchema = z.object({
	phone: z.string().trim().min(7, "settings.whatsapp.form.error.phone"),
});

type TestValues = z.infer<typeof testSchema>;

interface SendTestDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	/** Called after the test message was queued (to refresh the log). */
	onSent?: () => void;
}

/** Sends the reminder template with sample data to a phone. */
export function SendTestDialog({
	open,
	onOpenChange,
	onSent,
}: SendTestDialogProps) {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(false);

	async function onSubmit(values: TestValues) {
		setLoading(true);
		const res = await sendWhatsAppTest(values.phone);
		setLoading(false);
		if (res.success) {
			onOpenChange(false);
			onSent?.();
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[425px]">
				<DialogHeader>
					<DialogTitle>
						<Text uuid="settings.whatsapp.test.title" />
					</DialogTitle>
					<DialogDescription>
						<Text uuid="settings.whatsapp.test.description" />
					</DialogDescription>
				</DialogHeader>
				<Form
					schema={testSchema}
					defaultValues={{ phone: "" }}
					onSubmit={onSubmit}
				>
					{(field) => (
						<div className="space-y-4">
							<FormInput
								field={field}
								name="phone"
								type="tel"
								label={textGet("settings.whatsapp.test.phone")}
								placeholder={textGet(
									"settings.whatsapp.test.phone.placeholder",
								)}
							/>
							<DialogFooter>
								<Button type="submit" disabled={loading}>
									{loading ? (
										<Loader2 className="mr-2 h-4 w-4 animate-spin" />
									) : (
										<Send className="mr-2 h-4 w-4" />
									)}
									<Text uuid="settings.whatsapp.test.submit" />
								</Button>
							</DialogFooter>
						</div>
					)}
				</Form>
			</DialogContent>
		</Dialog>
	);
}
