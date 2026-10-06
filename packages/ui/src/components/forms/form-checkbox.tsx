import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type z from "zod";
import { Checkbox } from "../checkbox";
import {
	Field,
	FieldContent,
	FieldDescription,
	FieldError,
	FieldLabel,
} from "../field";

type FormCheckboxProps<
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

function FormCheckbox<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>({
	name,
	field,
	label,
	description,
	disabled,
}: FormCheckboxProps<T, Output, Input>) {
	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: inputField, fieldState }) => (
				<Field orientation="horizontal" data-invalid={fieldState.invalid}>
					<Checkbox
						id={name}
						checked={Boolean(inputField.value)}
						onCheckedChange={(checked) => inputField.onChange(checked)}
						onBlur={inputField.onBlur}
						disabled={disabled}
						aria-invalid={fieldState.invalid}
					/>
					<FieldContent>
						<FieldLabel htmlFor={name} className="font-normal">
							{label}
						</FieldLabel>
						{description && <FieldDescription>{description}</FieldDescription>}
						{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
					</FieldContent>
				</Field>
			)}
		/>
	);
}

export { FormCheckbox, type FormCheckboxProps };
