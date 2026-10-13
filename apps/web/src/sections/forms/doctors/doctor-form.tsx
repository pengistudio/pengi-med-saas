import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Form,
	FormCombobox,
	FormInput,
	FormRow,
	FormSelect,
} from "@pengi/ui";
import { Loader2, Save } from "lucide-react";
import { z } from "zod";
import { type Doctor, SPECIALTY_OTHER } from "@/api/doctors-service";
import { DoctorColorPicker } from "@/components/features/doctors/doctor-color-picker";
import { cn } from "@/lib/utils";
import { useSpecialties } from "@/store/doctors-store";

/** "none" is the "no linked user" option of the user select. */
export const NO_USER = "none";

const formSchema = z
	.object({
		full_name: z.string().trim().min(1, "doctors.form.error.required"),
		specialty: z.string().min(1, "doctors.form.error.required"),
		specialty_other: z.string().optional(),
		id_number: z.string().optional(),
		professional_registry: z.string().optional(),
		phone: z.string().optional(),
		email: z.union([z.literal(""), z.email("doctors.form.error.email")]),
		color: z.string().optional(),
		user_id: z.string().optional(),
	})
	.refine(
		(data) =>
			data.specialty !== SPECIALTY_OTHER || !!data.specialty_other?.trim(),
		{
			message: "doctors.form.error.specialty_other",
			path: ["specialty_other"],
		},
	);

export type DoctorFormValues = z.infer<typeof formSchema>;

interface DoctorFormProps {
	initialData?: Doctor | null;
	loading?: boolean;
	onSubmit: (values: DoctorFormValues) => void | Promise<void>;
	/** Admin create: lets them link the profile to a team member. */
	userOptions?: { value: string; label: string }[];
	/** Card texts; defaults to the admin create/edit ones. */
	title?: string;
	description?: string;
	/** Card width; centered at max-w-4xl by default. */
	className?: string;
}

/**
 * The doctor profile form: admin create/edit and the linked doctor's own
 * profile. The page builds the payload and calls the service.
 */
export default function DoctorForm({
	initialData,
	loading = false,
	onSubmit,
	userOptions,
	title,
	description,
	className,
}: DoctorFormProps) {
	const { textGet } = useText();
	const specialties = useSpecialties();
	const isEditing = Boolean(initialData);

	const defaultValues: DoctorFormValues = {
		full_name: initialData?.full_name ?? "",
		specialty: initialData?.specialty ?? "",
		specialty_other: initialData?.specialty_other ?? "",
		id_number: initialData?.id_number ?? "",
		professional_registry: initialData?.professional_registry ?? "",
		phone: initialData?.phone ?? "",
		email: initialData?.email ?? "",
		color: initialData?.color ?? "",
		user_id: NO_USER,
	};

	const specialtyOptions = specialties.map((s) => ({
		value: s.code,
		label: textGet(s.label_key),
	}));

	return (
		<Form schema={formSchema} onSubmit={onSubmit} defaultValues={defaultValues}>
			{(field) => (
				<Card className={cn("mx-auto w-full max-w-4xl", className)}>
					<CardHeader>
						<CardTitle>
							{title ??
								textGet(
									isEditing
										? "doctors.form.edit.title"
										: "doctors.form.create.title",
								)}
						</CardTitle>
						<CardDescription>
							{description ?? textGet("doctors.form.description")}
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<FormRow>
							<FormInput
								field={field}
								name="full_name"
								label={textGet("doctors.form.full_name")}
								placeholder={textGet("doctors.form.full_name.placeholder")}
							/>
							<FormCombobox
								field={field}
								name="specialty"
								label={textGet("doctors.form.specialty")}
								placeholder={textGet("doctors.form.specialty.placeholder")}
								options={specialtyOptions}
							/>
						</FormRow>
						{field.watch("specialty") === SPECIALTY_OTHER && (
							<FormInput
								field={field}
								name="specialty_other"
								label={textGet("doctors.form.specialty_other")}
								placeholder={textGet(
									"doctors.form.specialty_other.placeholder",
								)}
							/>
						)}
						<FormRow>
							<FormInput
								field={field}
								name="professional_registry"
								label={textGet("doctors.form.professional_registry")}
								placeholder={textGet(
									"doctors.form.professional_registry.placeholder",
								)}
								description={textGet(
									"doctors.form.professional_registry.description",
								)}
								isOptional
							/>
							<FormInput
								field={field}
								name="id_number"
								label={textGet("doctors.form.id_number")}
								isOptional
							/>
						</FormRow>
						<FormRow>
							<FormInput
								field={field}
								name="phone"
								label={textGet("doctors.form.phone")}
								isOptional
							/>
							<FormInput
								field={field}
								name="email"
								type="email"
								label={textGet("doctors.form.email")}
								isOptional
							/>
						</FormRow>
						{userOptions && (
							<FormSelect
								field={field}
								name="user_id"
								label={textGet("doctors.form.user")}
								description={textGet("doctors.form.user.description")}
								options={[
									{ value: NO_USER, label: textGet("doctors.form.user.none") },
									...userOptions,
								]}
								isOptional
							/>
						)}
						<DoctorColorPicker
							field={field}
							name="color"
							label={textGet("doctors.form.color")}
						/>
					</CardContent>
					<CardFooter className="flex justify-end">
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Save className="mr-2 h-4 w-4" />
							)}
							{textGet("doctors.form.save")}
						</Button>
					</CardFooter>
				</Card>
			)}
		</Form>
	);
}
