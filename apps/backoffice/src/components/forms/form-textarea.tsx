import type * as React from "react";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type z from "zod";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
} from "@/components/ui/field";
import { Textarea } from "@/components/ui/textarea";
import { useText } from "@/hooks/use-text";

type FormTextAreaProps<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
> = Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, "name"> & {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label?: React.ReactNode;
	isOptional?: boolean;
	description?: string;
	className?: string;
};

function FormTextArea<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>({
	name,
	field,
	isOptional,
	label,
	description,
	className,
	...props
}: FormTextAreaProps<T, Output, Input>) {
	const { textGet } = useText();

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

					<Textarea
						{...props}
						{...inputField}
						id={name}
						aria-invalid={fieldState.invalid}
						className={className}
					/>

					{description && <FieldDescription>{description}</FieldDescription>}
					{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
				</Field>
			)}
		/>
	);
}

export { FormTextArea, type FormTextAreaProps };
