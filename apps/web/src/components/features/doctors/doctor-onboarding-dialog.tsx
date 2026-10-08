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
	FormSelect,
} from "@pengi/ui";
import { Loader2, Stethoscope } from "lucide-react";
import React from "react";
import { z } from "zod";
import { createMyDoctor, SPECIALTY_OTHER } from "@/api/doctors-service";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import {
	useDoctorStatus,
	useDoctorStore,
	useSpecialties,
} from "@/store/doctors-store";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

const dismissKey = (environmentId: number) =>
	`doctor-onboarding-dismissed:${environmentId}`;

function isDismissed(environmentId: number) {
	try {
		return localStorage.getItem(dismissKey(environmentId)) === "1";
	} catch {
		return false;
	}
}

function dismiss(environmentId: number) {
	try {
		localStorage.setItem(dismissKey(environmentId), "1");
	} catch {
		// Storage blocked: the question comes back next load.
	}
}

const schema = z
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

/**
 * First-login question "Do you attend patients?": offered to a user who can
 * create their own doctor profile and has none, only while the tenant has no
 * active doctor (an admin who also attends in a clinic that already has
 * doctors creates it from the profile page). "Yes" creates the profile; "No"
 * is remembered in this browser.
 */
export function DoctorOnboardingDialog({
	enabled = true,
}: {
	/** Off when the tenant's plan has no clinical module. */
	enabled?: boolean;
}) {
	const { textGet } = useText();
	const environment = useSessionStore(selectEnvironment);
	const { checkPermission } = usePermission();
	const canCreateOwn =
		enabled &&
		(checkPermission([PERMISSIONS.DOCTORS.PERMISSION_MANAGE_DOCTORS]) ||
			checkPermission([
				PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
			]));
	const status = useDoctorStatus(canCreateOwn);
	const refresh = useDoctorStore((s) => s.refresh);
	const [step, setStep] = React.useState<"ask" | "form">("ask");
	const [closed, setClosed] = React.useState(false);
	const [saving, setSaving] = React.useState(false);

	// Another user or tenant gets their own question.
	const [prevEnvironmentId, setPrevEnvironmentId] = React.useState(
		environment?.id,
	);
	if (environment?.id !== prevEnvironmentId) {
		setPrevEnvironmentId(environment?.id);
		setClosed(false);
		setStep("ask");
	}

	const open =
		!closed &&
		canCreateOwn &&
		!!environment &&
		!!status &&
		!status.has_profile &&
		status.active_doctors === 0 &&
		!isDismissed(environment.id);

	function handleNo() {
		if (environment) dismiss(environment.id);
		setClosed(true);
	}

	async function handleSubmit(values: z.infer<typeof schema>) {
		setSaving(true);
		const res = await createMyDoctor({
			full_name: values.full_name.trim(),
			specialty: values.specialty,
			specialty_other:
				values.specialty === SPECIALTY_OTHER
					? (values.specialty_other ?? "").trim()
					: "",
			professional_registry: values.professional_registry ?? "",
		});
		setSaving(false);
		if (res.success) {
			setClosed(true);
			refresh();
		}
	}

	return (
		<Dialog open={open} onOpenChange={(next) => !next && setClosed(true)}>
			<DialogContent className="sm:max-w-[480px]">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Stethoscope className="h-5 w-5 text-primary" />
						{textGet("doctors.onboarding.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet(
							step === "ask"
								? "doctors.onboarding.description"
								: "doctors.onboarding.form.description",
						)}
					</DialogDescription>
				</DialogHeader>
				{step === "ask" ? (
					<DialogFooter>
						<Button variant="outline" onClick={handleNo}>
							{textGet("doctors.onboarding.no")}
						</Button>
						<Button onClick={() => setStep("form")}>
							{textGet("doctors.onboarding.yes")}
						</Button>
					</DialogFooter>
				) : (
					<OnboardingForm
						defaultName={environment?.name ?? ""}
						saving={saving}
						onSubmit={handleSubmit}
						onBack={() => setStep("ask")}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}

function OnboardingForm({
	defaultName,
	saving,
	onSubmit,
	onBack,
}: {
	defaultName: string;
	saving: boolean;
	onSubmit: (values: z.infer<typeof schema>) => void;
	onBack: () => void;
}) {
	const { textGet } = useText();
	const specialties = useSpecialties();
	return (
		<Form
			schema={schema}
			onSubmit={onSubmit}
			defaultValues={{
				full_name: defaultName,
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
						placeholder={textGet("doctors.form.full_name.placeholder")}
					/>
					<FormSelect
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
							placeholder={textGet("doctors.form.specialty_other.placeholder")}
						/>
					)}
					<FormInput
						field={field}
						name="professional_registry"
						label={textGet("doctors.form.professional_registry")}
						placeholder={textGet(
							"doctors.form.professional_registry.placeholder",
						)}
						isOptional
					/>
					<DialogFooter>
						<Button type="button" variant="outline" onClick={onBack}>
							{textGet("doctors.onboarding.back")}
						</Button>
						<Button type="submit" disabled={saving}>
							{saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
							{textGet("doctors.onboarding.create")}
						</Button>
					</DialogFooter>
				</div>
			)}
		</Form>
	);
}
