import { Field, FieldContent, FieldDescription, FieldLabel } from "@pengi/ui";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type z from "zod";
import { Switch } from "@/components/ui/switch";

type FormSwitchProps<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
> = {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label: string;
	description?: string;
	disabled?: boolean;
};

/** A boolean setting: label (and help) on the left, the switch on the right. */
function FormSwitch<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>({
	name,
	field,
	label,
	description,
	disabled,
}: FormSwitchProps<T, Output, Input>) {
	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: inputField }) => (
				<Field orientation="horizontal" className="justify-between gap-4">
					<FieldContent>
						<FieldLabel htmlFor={name}>{label}</FieldLabel>
						{description && <FieldDescription>{description}</FieldDescription>}
					</FieldContent>
					<Switch
						id={name}
						checked={Boolean(inputField.value)}
						onCheckedChange={(checked) => inputField.onChange(checked)}
						onBlur={inputField.onBlur}
						disabled={disabled}
					/>
				</Field>
			)}
		/>
	);
}

export { FormSwitch };
