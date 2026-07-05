import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type z from "zod";
import { useUiText } from "../../context/text-context";
import { Field, FieldDescription, FieldError, FieldLabel } from "../field";
import { Input } from "../input";
import { InputGroup, InputGroupAddon, InputGroupInput } from "../input-group";

type FormInputProps<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
> = Omit<React.InputHTMLAttributes<HTMLInputElement>, "name"> & {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label?: string;
	isOptional?: boolean;
	description?: string;
	startAddon?: React.ReactNode;
	endAddon?: React.ReactNode;
	containerClassName?: string;
};

function FormInput<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>({
	name,
	field,
	isOptional,
	label,
	description,
	type,
	startAddon,
	endAddon,
	containerClassName,
	className,
	...props
}: FormInputProps<T, Output, Input>) {
	const { textGet } = useUiText();

	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: inputField, fieldState }) => (
				<Field data-invalid={fieldState.invalid}>
					{label && (
						<FieldLabel htmlFor={name}>
							{label}{" "}
							{isOptional && (
								<span className="text-xs text-muted-foreground font-normal">
									({textGet("form.optional") || "opcional"})
								</span>
							)}
						</FieldLabel>
					)}

					{startAddon || endAddon ? (
						<InputGroup className={containerClassName}>
							{startAddon && (
								<InputGroupAddon align="inline-start">
									{startAddon}
								</InputGroupAddon>
							)}
							<InputGroupInput
								{...props}
								{...inputField}
								id={name}
								type={type}
								aria-invalid={fieldState.invalid}
								className={className}
								onChange={(e) => {
									if (type === "number") {
										const value = e.target.valueAsNumber;
										inputField.onChange(Number.isNaN(value) ? "" : value);
									} else {
										inputField.onChange(e);
									}
								}}
							/>
							{endAddon && (
								<InputGroupAddon align="inline-end">{endAddon}</InputGroupAddon>
							)}
						</InputGroup>
					) : (
						<Input
							{...props}
							{...inputField}
							id={name}
							type={type}
							aria-invalid={fieldState.invalid}
							className={className}
							onChange={(e) => {
								if (type === "number") {
									const value = e.target.valueAsNumber;
									inputField.onChange(Number.isNaN(value) ? "" : value);
								} else {
									inputField.onChange(e);
								}
							}}
						/>
					)}

					{description && <FieldDescription>{description}</FieldDescription>}
					{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
				</Field>
			)}
		/>
	);
}

export { FormInput, type FormInputProps };
