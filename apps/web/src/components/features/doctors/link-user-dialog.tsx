import { useText } from "@pengi/shared";
import {
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Field,
	FieldLabel,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import { Loader2 } from "lucide-react";
import React from "react";
import { type Doctor, linkDoctorUser } from "@/api/doctors-service";
import { useLinkableUsers } from "@/components/features/doctors/use-linkable-users";

const NONE = "none";

interface LinkUserDialogProps {
	doctor: Doctor | null;
	onOpenChange: (open: boolean) => void;
	onLinked: () => void;
}

/** Links a doctor to a team member's account, or unlinks it. */
export function LinkUserDialog({
	doctor,
	onOpenChange,
	onLinked,
}: LinkUserDialogProps) {
	const { textGet } = useText();
	const options = useLinkableUsers(doctor);
	const [value, setValue] = React.useState(NONE);
	const [saving, setSaving] = React.useState(false);

	// Start from the doctor's current link each time the dialog opens.
	const [prevDoctor, setPrevDoctor] = React.useState<Doctor | null>(null);
	if (doctor !== prevDoctor) {
		setPrevDoctor(doctor);
		setValue(doctor?.user_id ? String(doctor.user_id) : NONE);
	}

	const allOptions = [
		{ value: NONE, label: textGet("doctors.form.user.none") },
		...options,
	];

	async function handleSave() {
		if (!doctor) return;
		setSaving(true);
		const res = await linkDoctorUser(
			doctor.ID,
			value === NONE ? null : Number(value),
		);
		setSaving(false);
		if (res.success) {
			onOpenChange(false);
			onLinked();
		}
	}

	return (
		<Dialog open={doctor !== null} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[460px]">
				<DialogHeader>
					<DialogTitle>{textGet("doctors.link.title")}</DialogTitle>
					<DialogDescription>
						{textGet("doctors.link.description", {
							name: doctor?.full_name ?? "",
						})}
					</DialogDescription>
				</DialogHeader>
				<Field>
					<FieldLabel>{textGet("doctors.form.user")}</FieldLabel>
					<Select
						value={value}
						onValueChange={(v) => setValue(String(v ?? NONE))}
					>
						<SelectTrigger>
							<SelectValue>
								{allOptions.find((o) => o.value === value)?.label}
							</SelectValue>
						</SelectTrigger>
						<SelectContent>
							{allOptions.map((o) => (
								<SelectItem key={o.value} value={o.value}>
									{o.label}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</Field>
				<DialogFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)}>
						{textGet("form.cancel")}
					</Button>
					<Button onClick={handleSave} disabled={saving}>
						{saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
						{textGet("doctors.form.save")}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
