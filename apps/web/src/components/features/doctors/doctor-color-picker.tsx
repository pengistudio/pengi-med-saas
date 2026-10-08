import { useText } from "@pengi/shared";
import { Field, FieldDescription, FieldLabel, Input } from "@pengi/ui";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import { DOCTOR_COLORS } from "@/api/doctors-service";
import { cn } from "@/lib/utils";

interface DoctorColorPickerProps<T extends FieldValues> {
	field: UseFormReturn<T>;
	name: Path<T>;
	label?: React.ReactNode;
}

/**
 * Agenda color of a doctor: one of the palette swatches or any color from the
 * native picker. Empty means "assign the next palette color" (create only).
 */
export function DoctorColorPicker<T extends FieldValues>({
	field,
	name,
	label,
}: DoctorColorPickerProps<T>) {
	const { textGet } = useText();
	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: { value, onChange } }) => {
				const current = typeof value === "string" ? value.toUpperCase() : "";
				const isCustom =
					current !== "" && !DOCTOR_COLORS.some((color) => color === current);
				return (
					<Field>
						{label && <FieldLabel>{label}</FieldLabel>}
						<div className="flex flex-wrap items-center gap-2">
							{DOCTOR_COLORS.map((color) => (
								<button
									type="button"
									key={color}
									aria-label={color}
									aria-pressed={current === color}
									onClick={() => onChange(color)}
									style={{ backgroundColor: color }}
									className={cn(
										"h-7 w-7 cursor-pointer rounded-full transition-transform hover:scale-110",
										current === color && "ring-2 ring-primary ring-offset-2",
									)}
								/>
							))}
							<Input
								type="color"
								aria-label={textGet("doctors.form.color.custom")}
								title={textGet("doctors.form.color.custom")}
								value={current || DOCTOR_COLORS[0]}
								onChange={(e) => onChange(e.target.value.toUpperCase())}
								className={cn(
									"h-8 w-12 cursor-pointer p-1",
									isCustom && "ring-2 ring-primary ring-offset-2",
								)}
							/>
						</div>
						{!current && (
							<FieldDescription>
								{textGet("doctors.form.color.auto")}
							</FieldDescription>
						)}
					</Field>
				);
			}}
		/>
	);
}
