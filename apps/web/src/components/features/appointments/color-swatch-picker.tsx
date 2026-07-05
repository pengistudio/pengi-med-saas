import { Field, FieldLabel } from "@pengi/ui";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import { cn } from "@/lib/utils";
import { EVENT_COLORS } from "./appointment-utils";

interface ColorSwatchPickerProps<T extends FieldValues> {
	field: UseFormReturn<T>;
	name: Path<T>;
	label?: React.ReactNode;
}

export function ColorSwatchPicker<T extends FieldValues>({
	field,
	name,
	label,
}: ColorSwatchPickerProps<T>) {
	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: { value, onChange } }) => (
				<Field>
					{label && <FieldLabel>{label}</FieldLabel>}
					<div className="flex flex-wrap gap-2">
						<button
							type="button"
							aria-label="default"
							onClick={() => onChange("")}
							className={cn(
								"h-7 w-7 rounded-full border-2 border-dashed border-gray-400 bg-transparent transition-transform hover:scale-110 cursor-pointer",
								!value && "ring-2 ring-offset-2 ring-primary",
							)}
						/>
						{EVENT_COLORS.map((color) => (
							<button
								type="button"
								key={color.id}
								aria-label={color.name}
								onClick={() => onChange(color.id)}
								style={{ backgroundColor: color.hex }}
								className={cn(
									"h-7 w-7 rounded-full transition-transform hover:scale-110 cursor-pointer",
									value === color.id && "ring-2 ring-offset-2 ring-primary",
								)}
							/>
						))}
					</div>
				</Field>
			)}
		/>
	);
}
