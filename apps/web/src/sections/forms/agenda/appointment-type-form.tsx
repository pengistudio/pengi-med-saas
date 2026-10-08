import { useText } from "@pengi/shared";
import {
	Button,
	Checkbox,
	Field,
	FieldDescription,
	FieldLabel,
	Form,
	FormCheckbox,
	FormInput,
} from "@pengi/ui";
import { Loader2, Save, X } from "lucide-react";
import { Controller } from "react-hook-form";
import { z } from "zod";
import type {
	AppointmentType,
	AppointmentTypePayload,
} from "@/api/agenda-service";
import { DoctorColorPicker } from "@/components/features/doctors/doctor-color-picker";
import { useDoctors } from "@/store/doctors-store";

const typeSchema = z.object({
	name: z
		.string()
		.trim()
		.min(1, "agenda.types.error.name_required")
		.max(100, "agenda.types.error.name_too_long"),
	duration_minutes: z
		.number({ error: "agenda.types.error.duration" })
		.int("agenda.types.error.duration")
		.min(5, "agenda.types.error.duration")
		.max(480, "agenda.types.error.duration")
		.refine((v) => v % 5 === 0, "agenda.types.error.duration"),
	color: z.string(),
	active: z.boolean(),
	doctor_ids: z.array(z.number()),
});

type TypeValues = z.infer<typeof typeSchema>;

/**
 * An appointment type: name, duration (sets the end time in the appointment
 * form), optional color, the doctors who offer it (none ticked = everyone)
 * and whether it is offered at all.
 */
export function AppointmentTypeForm({
	type,
	loading,
	onSubmit,
}: {
	type?: AppointmentType;
	loading?: boolean;
	onSubmit: (payload: AppointmentTypePayload) => void;
}) {
	const { textGet } = useText();
	const { doctors } = useDoctors();
	return (
		<Form
			schema={typeSchema}
			onSubmit={(values: TypeValues) =>
				onSubmit({ ...values, name: values.name.trim() })
			}
			defaultValues={{
				name: type?.name ?? "",
				duration_minutes: type?.duration_minutes ?? 30,
				color: type?.color ?? "",
				active: type?.active ?? true,
				doctor_ids: type?.doctor_ids ?? [],
			}}
		>
			{(field) => {
				const selected = new Set(field.watch("doctor_ids") ?? []);
				// Active doctors, plus inactive ones the type already lists.
				const choices = doctors.filter((d) => d.active || selected.has(d.ID));
				return (
					<div className="space-y-4">
						<FormInput
							field={field}
							name="name"
							label={textGet("agenda.types.field.name")}
							placeholder={textGet("agenda.types.field.name_placeholder")}
						/>
						<FormInput
							field={field}
							name="duration_minutes"
							type="number"
							min={5}
							max={480}
							step={5}
							label={textGet("agenda.types.field.duration")}
							description={textGet("agenda.types.field.duration_hint")}
						/>
						<div className="grid gap-2">
							<DoctorColorPicker
								field={field}
								name="color"
								label={textGet("agenda.types.field.color")}
								emptyDescription={textGet("agenda.types.field.color_none")}
							/>
							{field.watch("color") && (
								<Button
									type="button"
									variant="ghost"
									size="sm"
									className="w-fit"
									onClick={() =>
										field.setValue("color", "", { shouldDirty: true })
									}
								>
									<X className="mr-1 h-4 w-4" />
									{textGet("agenda.types.field.color_clear")}
								</Button>
							)}
						</div>
						{choices.length > 0 && (
							<Controller
								control={field.control}
								name="doctor_ids"
								render={({ field: { value, onChange } }) => {
									const ids = (value as number[] | undefined) ?? [];
									return (
										<Field>
											<FieldLabel>
												{textGet("agenda.types.field.doctors")}
											</FieldLabel>
											<div className="grid max-h-48 gap-1.5 overflow-y-auto rounded-lg border p-2">
												{choices.map((d) => (
													<div
														key={d.ID}
														className="flex items-center gap-2 text-sm"
													>
														<Checkbox
															id={`type-doctor-${d.ID}`}
															checked={ids.includes(d.ID)}
															onCheckedChange={(checked) =>
																onChange(
																	checked
																		? [...ids, d.ID]
																		: ids.filter((id) => id !== d.ID),
																)
															}
														/>
														<label
															htmlFor={`type-doctor-${d.ID}`}
															className="cursor-pointer"
														>
															{d.full_name}
															{!d.active &&
																` (${textGet("doctors.status.inactive")})`}
														</label>
													</div>
												))}
											</div>
											<FieldDescription>
												{textGet("agenda.types.field.doctors_hint")}
											</FieldDescription>
										</Field>
									);
								}}
							/>
						)}
						<FormCheckbox
							field={field}
							name="active"
							label={textGet("agenda.types.field.active")}
						/>
						<div className="flex justify-end">
							<Button type="submit" disabled={loading}>
								{loading ? (
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
								) : (
									<Save className="mr-2 h-4 w-4" />
								)}
								{textGet("agenda.types.save")}
							</Button>
						</div>
					</div>
				);
			}}
		</Form>
	);
}
