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
	FormTextArea,
} from "@pengi/ui";
import { Loader2, X } from "lucide-react";
import React from "react";
import { z } from "zod";
import {
	type AppointmentRequest,
	rejectAppointmentRequest,
	requesterName,
} from "@/api/appointment-request-service";

const rejectSchema = z.object({
	reason: z
		.string()
		.trim()
		.max(500, "appointment_requests.reject.error.too_long")
		.optional(),
});

type RejectValues = z.infer<typeof rejectSchema>;

/** Turns a request down; the optional reason reaches the patient. */
export function RejectRequestDialog({
	request,
	onOpenChange,
	onRejected,
}: {
	/** The request to reject; the dialog is open while set. */
	request: AppointmentRequest | null;
	onOpenChange: (open: boolean) => void;
	onRejected: () => void;
}) {
	const { textGet } = useText();
	const [loading, setLoading] = React.useState(false);

	async function onSubmit(values: RejectValues) {
		if (!request) return;
		setLoading(true);
		const res = await rejectAppointmentRequest(request.id, values.reason ?? "");
		setLoading(false);
		if (res.success) {
			onOpenChange(false);
			onRejected();
		}
	}

	return (
		<Dialog open={Boolean(request)} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[480px]">
				<DialogHeader>
					<DialogTitle>
						{textGet("appointment_requests.reject.title")}
					</DialogTitle>
					<DialogDescription>
						{request &&
							textGet("appointment_requests.reject.description", {
								patient: requesterName(request),
							})}
					</DialogDescription>
				</DialogHeader>
				{/* Keyed by request: each one starts with an empty reason. */}
				<Form
					key={request?.id}
					schema={rejectSchema}
					defaultValues={{ reason: "" }}
					onSubmit={onSubmit}
				>
					{(field) => (
						<div className="space-y-4">
							<FormTextArea
								field={field}
								name="reason"
								label={textGet("appointment_requests.reject.reason")}
								placeholder={textGet(
									"appointment_requests.reject.reason.placeholder",
								)}
								description={textGet(
									"appointment_requests.reject.reason.description",
								)}
								maxLength={500}
								isOptional
							/>
							<DialogFooter>
								<Button
									type="button"
									variant="outline"
									onClick={() => onOpenChange(false)}
								>
									{textGet("common.cancel")}
								</Button>
								<Button type="submit" variant="destructive" disabled={loading}>
									{loading ? (
										<Loader2 className="mr-2 h-4 w-4 animate-spin" />
									) : (
										<X className="mr-2 h-4 w-4" />
									)}
									{textGet("appointment_requests.action.reject")}
								</Button>
							</DialogFooter>
						</div>
					)}
				</Form>
			</DialogContent>
		</Dialog>
	);
}
