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
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";

type FormRadioGroupProps<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
> = {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label?: React.ReactNode;
	isOptional?: boolean;
	description?: string;
	options: { value: string; label: string | React.ReactNode }[];
	isRow?: boolean;
	className?: string;
};

function FormRadioGroup<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>({
	name,
	field,
	isOptional,
	label,
	description,
	options,
	isRow,
	className,
}: FormRadioGroupProps<T, Output, Input>) {
	const { textGet } = useText();

	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: { value, onChange, disabled }, fieldState }) => (
				<Field data-invalid={fieldState.invalid}>
					{label && (
						<FieldLabel>
							{label}{" "}
							{isOptional && (
								<span className="text-xs text-muted-foreground font-normal">
									({textGet("form.optional") || "opcional"})
								</span>
							)}
						</FieldLabel>
					)}
					<RadioGroup
						onValueChange={onChange}
						value={value}
						disabled={disabled}
						className={cn(
							isRow ? "flex flex-row space-x-4 h-8" : "flex flex-col space-y-2",
							className,
						)}
					>
						{options.map((option) => (
							<div key={option.value} className="flex items-center space-x-2">
								<RadioGroupItem
									value={option.value}
									id={`${name}-${option.value}`}
								/>
								<FieldLabel
									htmlFor={`${name}-${option.value}`}
									className="font-normal cursor-pointer"
								>
									{option.label}
								</FieldLabel>
							</div>
						))}
					</RadioGroup>
					{description && <FieldDescription>{description}</FieldDescription>}
					{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
				</Field>
			)}
		/>
	);
}

export { FormRadioGroup, type FormRadioGroupProps };
