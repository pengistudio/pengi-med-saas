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
import {
	Combobox,
	ComboboxContent,
	ComboboxEmpty,
	ComboboxInput,
	ComboboxItem,
	ComboboxList,
} from "../combobox";
import { Field, FieldDescription, FieldError, FieldLabel } from "../field";

type FormComboboxOption = { value: string; label: string };

type FormComboboxProps<
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
	options: FormComboboxOption[];
	className?: string;
	disabled?: boolean;
	emptyMessage?: React.ReactNode;
};

/** Lowercase without accents, so "oncologia" finds "Oncología". */
function normalize(text: string) {
	return text
		.normalize("NFD")
		.replace(/\p{Diacritic}/gu, "")
		.toLowerCase();
}

function matchesOption(option: FormComboboxOption, query: string) {
	return normalize(option.label).includes(normalize(query.trim()));
}

/**
 * FormSelect with a search box: for long option lists the user types to
 * filter (accent- and case-insensitive) instead of scrolling. The form value
 * is the option's `value`; labels must be plain text.
 */
function FormCombobox<
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
}: FormComboboxProps<T, Output, Input>) {
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
					<Combobox
						items={options}
						value={options.find((o) => o.value === value) ?? null}
						onValueChange={(option: FormComboboxOption | null) =>
							onChange(option?.value ?? "")
						}
						itemToStringLabel={(option: FormComboboxOption) => option.label}
						isItemEqualToValue={(
							a: FormComboboxOption,
							b: FormComboboxOption,
						) => a.value === b.value}
						filter={matchesOption}
						disabled={disabled !== undefined ? disabled : fieldDisabled}
					>
						<ComboboxInput
							id={name}
							placeholder={placeholder}
							aria-invalid={fieldState.invalid}
							className={cn(
								fieldState.invalid &&
									"border-destructive focus-visible:ring-destructive",
							)}
						/>
						<ComboboxContent>
							<ComboboxEmpty>
								{emptyMessage ?? textGet("form.select.no_options")}
							</ComboboxEmpty>
							<ComboboxList>
								{(option: FormComboboxOption) => (
									<ComboboxItem key={option.value} value={option}>
										{option.label}
									</ComboboxItem>
								)}
							</ComboboxList>
						</ComboboxContent>
					</Combobox>
					{description && <FieldDescription>{description}</FieldDescription>}
					{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
				</Field>
			)}
		/>
	);
}

export { FormCombobox, type FormComboboxOption, type FormComboboxProps };
