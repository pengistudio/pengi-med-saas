import { useText } from "@pengi/shared";
import {
	Button,
	Input,
	Popover,
	PopoverContent,
	PopoverTrigger,
	Text,
} from "@pengi/ui";
import { Loader2, Mail } from "lucide-react";
import React from "react";

interface SendEmailPopoverProps {
	defaultEmail: string;
	onSend: (email: string) => Promise<void>;
	/** Shown next to the icon; without it the trigger is icon-only. */
	label?: React.ReactNode;
	disabled?: boolean;
	className?: string;
}

/** Asks for a recipient address and sends a document to it. */
export function SendEmailPopover({
	defaultEmail,
	onSend,
	label,
	disabled,
	className,
}: SendEmailPopoverProps) {
	const { textGet } = useText();
	const [email, setEmail] = React.useState(defaultEmail);
	const [sending, setSending] = React.useState(false);

	React.useEffect(() => setEmail(defaultEmail), [defaultEmail]);

	async function handleSend() {
		if (!email) return;
		setSending(true);
		await onSend(email);
		setSending(false);
	}

	return (
		<Popover>
			<PopoverTrigger
				render={
					label ? (
						<Button
							type="button"
							variant="outline"
							disabled={disabled}
							className={className}
						>
							<Mail className="mr-2 h-4 w-4" />
							{label}
						</Button>
					) : (
						<Button
							type="button"
							variant="ghost"
							size="icon"
							disabled={disabled}
							className={className}
						>
							<Mail className="h-4 w-4" />
						</Button>
					)
				}
			/>
			<PopoverContent className="w-72 space-y-2">
				<p className="text-sm font-medium">
					<Text uuid="clinical.medical_documents.send_email.title" />
				</p>
				<div className="flex items-center gap-2">
					<Input
						type="email"
						value={email}
						onChange={(e) => setEmail(e.target.value)}
						placeholder={textGet("dialog.medical_report.email_placeholder")}
						onKeyDown={(e) => {
							if (e.key === "Enter") {
								e.preventDefault();
								handleSend();
							}
						}}
					/>
					<Button
						type="button"
						size="icon"
						disabled={sending || !email}
						onClick={handleSend}
						aria-label={textGet("dialog.medical_report.send_email")}
					>
						{sending ? (
							<Loader2 className="h-4 w-4 animate-spin" />
						) : (
							<Mail className="h-4 w-4" />
						)}
					</Button>
				</div>
			</PopoverContent>
		</Popover>
	);
}
