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
	Form,
	FormCombobox,
	FormInput,
	Label,
	RadioGroup,
	RadioGroupItem,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import { Loader2, Stethoscope } from "lucide-react";
import React from "react";
import { z } from "zod";
import {
	createDoctor,
	linkDoctorUser,
	SPECIALTY_OTHER,
} from "@/api/doctors-service";
import type { TeamMember } from "@/api/team-service";
import { specialtyLabel } from "@/components/features/doctors/doctor-utils";
import {
	useDoctorStore,
	useDoctors,
	useSpecialties,
} from "@/store/doctors-store";

type Mode = "link" | "create" | "none";

const createSchema = z
	.object({
		full_name: z.string().trim().min(1, "doctors.form.error.required"),
		specialty: z.string().min(1, "doctors.form.error.required"),
		specialty_other: z.string().optional(),
		professional_registry: z.string().optional(),
	})
	.refine(
		(data) =>
			data.specialty !== SPECIALTY_OTHER || !!data.specialty_other?.trim(),
		{
			message: "doctors.form.error.specialty_other",
			path: ["specialty_other"],
		},
	);

interface MemberDoctorDialogProps {
	/** The team member (doctor role, no profile yet); null closes the dialog. */
	member: TeamMember | null;
	onOpenChange: (open: boolean) => void;
}

/**
 * Gives a team member with the doctor role their doctor profile, once they
 * joined (invites are links: the account only exists after they accept):
 * link an existing profile without account, create a new one, or none.
 */
export function MemberDoctorDialog({
	member,
	onOpenChange,
}: MemberDoctorDialogProps) {
	const { textGet } = useText();
	const { activeDoctors } = useDoctors(member !== null);
	const specialties = useSpecialties();
	const refresh = useDoctorStore((s) => s.refresh);
	const unlinked = activeDoctors.filter((d) => !d.user_id);
	const [mode, setMode] = React.useState<Mode>("create");
	const [doctorId, setDoctorId] = React.useState("");
	const [saving, setSaving] = React.useState(false);

	// Start over each time it opens for a member.
	const [prevMember, setPrevMember] = React.useState<TeamMember | null>(null);
	if (member !== prevMember) {
		setPrevMember(member);
		setMode(unlinked.length > 0 ? "link" : "create");
		setDoctorId("");
	}

	const memberName = member ? member.environment_name || member.user_name : "";

	async function handleLink() {
		if (!member || !doctorId) return;
		setSaving(true);
		const res = await linkDoctorUser(Number(doctorId), member.user_id);
		setSaving(false);
		if (res.success) {
			await refresh();
			onOpenChange(false);
		}
	}

	async function handleCreate(values: z.infer<typeof createSchema>) {
		if (!member) return;
		setSaving(true);
		const res = await createDoctor({
			full_name: values.full_name.trim(),
			specialty: values.specialty,
			specialty_other:
				values.specialty === SPECIALTY_OTHER
					? (values.specialty_other ?? "").trim()
					: "",
			professional_registry: values.professional_registry ?? "",
			user_id: member.user_id,
		});
		setSaving(false);
		if (res.success) {
			await refresh();
			onOpenChange(false);
		}
	}

	const modes: { value: Mode; label: string; disabled?: boolean }[] = [
		{
			value: "link",
			label: textGet("doctors.member.mode.link"),
			disabled: unlinked.length === 0,
		},
		{ value: "create", label: textGet("doctors.member.mode.create") },
		{ value: "none", label: textGet("doctors.member.mode.none") },
	];

	return (
		<Dialog open={member !== null} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[480px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Stethoscope className="h-5 w-5 text-primary" />
						{textGet("doctors.member.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("doctors.member.description", { name: memberName })}
					</DialogDescription>
				</DialogHeader>

				<RadioGroup
					aria-label={textGet("doctors.member.title")}
					value={mode}
					onValueChange={(v) => setMode(v as Mode)}
				>
					{modes.map((m) => (
						<Label
							key={m.value}
							htmlFor={`member-doctor-${m.value}`}
							className="flex cursor-pointer items-center gap-3 rounded-lg border p-3 font-normal has-data-checked:border-primary has-data-checked:bg-primary/5"
						>
							<RadioGroupItem
								id={`member-doctor-${m.value}`}
								value={m.value}
								disabled={m.disabled}
							/>
							<span className="text-sm">{m.label}</span>
						</Label>
					))}
				</RadioGroup>

				{mode === "link" && (
					<>
						<Field>
							<FieldLabel>{textGet("doctors.member.existing")}</FieldLabel>
							<Select
								value={doctorId}
								onValueChange={(v) => setDoctorId(String(v ?? ""))}
							>
								<SelectTrigger>
									<SelectValue
										placeholder={textGet("doctors.select.placeholder")}
									>
										{(() => {
											const d = unlinked.find((x) => String(x.ID) === doctorId);
											return d
												? `${d.full_name} · ${specialtyLabel(d, textGet)}`
												: textGet("doctors.select.placeholder");
										})()}
									</SelectValue>
								</SelectTrigger>
								<SelectContent>
									{unlinked.map((d) => (
										<SelectItem key={d.ID} value={String(d.ID)}>
											{d.full_name} · {specialtyLabel(d, textGet)}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</Field>
						<DialogFooter>
							<Button onClick={handleLink} disabled={!doctorId || saving}>
								{saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
								{textGet("doctors.member.link")}
							</Button>
						</DialogFooter>
					</>
				)}

				{mode === "create" && member && (
					<Form
						schema={createSchema}
						onSubmit={handleCreate}
						defaultValues={{
							full_name: memberName,
							specialty: "",
							specialty_other: "",
							professional_registry: "",
						}}
					>
						{(field) => (
							<div className="space-y-4">
								<FormInput
									field={field}
									name="full_name"
									label={textGet("doctors.form.full_name")}
								/>
								<FormCombobox
									field={field}
									name="specialty"
									label={textGet("doctors.form.specialty")}
									placeholder={textGet("doctors.form.specialty.placeholder")}
									options={specialties.map((s) => ({
										value: s.code,
										label: textGet(s.label_key),
									}))}
								/>
								{field.watch("specialty") === SPECIALTY_OTHER && (
									<FormInput
										field={field}
										name="specialty_other"
										label={textGet("doctors.form.specialty_other")}
									/>
								)}
								<FormInput
									field={field}
									name="professional_registry"
									label={textGet("doctors.form.professional_registry")}
									isOptional
								/>
								<DialogFooter>
									<Button type="submit" disabled={saving}>
										{saving && (
											<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										)}
										{textGet("doctors.member.create")}
									</Button>
								</DialogFooter>
							</div>
						)}
					</Form>
				)}

				{mode === "none" && (
					<DialogFooter>
						<Button variant="outline" onClick={() => onOpenChange(false)}>
							{textGet("form.cancel")}
						</Button>
					</DialogFooter>
				)}
			</DialogContent>
		</Dialog>
	);
}
