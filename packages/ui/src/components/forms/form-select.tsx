import type * as React from "react";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type z from "zod";
import { useUiText } from "../../context/text-context";
import { cn } from "../../lib/utils";
import { Field, FieldDescription, FieldError, FieldLabel } from "../field";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "../select";

type FormSelectProps<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
> = {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label?: React.ReactNode;
	isOptional?: boolean;
	description?: string;
	placeholder?: string;
	options: { value: string; label: string | React.ReactNode }[];
	className?: string;
	disabled?: boolean;
	emptyMessage?: React.ReactNode;
};

function FormSelect<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>({
	name,
	field,
	isOptional,
	label,
	description,
	placeholder,
	options,
	className,
	disabled,
	emptyMessage,
}: FormSelectProps<T, Output, Input>) {
	const { textGet } = useUiText();

	return (
		<Controller
			control={field.control}
			name={name}
			render={({
				field: { value, onChange, disabled: fieldDisabled },
				fieldState,
			}) => (
				<Field
					data-invalid={fieldState.invalid}
					className={cn("flex flex-col", className)}
				>
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
					<Select
						value={value}
						onValueChange={onChange}
						disabled={disabled !== undefined ? disabled : fieldDisabled}
					>
						<SelectTrigger
							id={name}
							aria-invalid={fieldState.invalid}
							className={cn(
								fieldState.invalid &&
									"border-destructive focus-visible:ring-destructive",
							)}
						>
							<SelectValue placeholder={placeholder}>
								{(() => {
									const selected = options.find((o) => o.value === value);
									return selected ? selected.label : placeholder;
								})()}
							</SelectValue>
						</SelectTrigger>
						<SelectContent>
							{options.length === 0 ? (
								<div className="text-muted-foreground px-2 py-4 text-center text-sm">
									{emptyMessage ?? textGet("form.select.no_options")}
								</div>
							) : (
								options.map((option) => (
									<SelectItem key={option.value} value={option.value}>
										{option.label}
									</SelectItem>
								))
							)}
						</SelectContent>
					</Select>
					{description && <FieldDescription>{description}</FieldDescription>}
					{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
				</Field>
			)}
		/>
	);
}

export { FormSelect, type FormSelectProps };
