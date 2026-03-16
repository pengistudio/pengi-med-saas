import { useCallback, useRef, useState } from "react";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type z from "zod";
import { type DiagnosisItem, searchICD10, searchICD11 } from "@/api/clinical-service";
import {
	Combobox,
	ComboboxChip,
	ComboboxChips,
	ComboboxChipsInput,
	ComboboxContent,
	ComboboxEmpty,
	ComboboxItem,
	ComboboxList,
	useComboboxAnchor,
} from "@/components/ui/combobox";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
} from "@/components/ui/field";
import { useText } from "@/hooks/use-text";

type FormIcd11SelectProps<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
> = {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label?: string;
	isOptional?: boolean;
	description?: string;
	system?: "cie11" | "cie10";
};

function FormIcd11Select<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>({
	name,
	field,
	label,
	isOptional,
	description,
	system = "cie11",
}: FormIcd11SelectProps<T, Output, Input>) {
	const { textGet } = useText();
	const anchor = useComboboxAnchor();
	const [results, setResults] = useState<DiagnosisItem[]>([]);
	const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	const searchFn = system === "cie10" ? searchICD10 : searchICD11;

	const handleInputChange = useCallback((value: string) => {
		if (debounceRef.current) clearTimeout(debounceRef.current);
		if (!value || value.length < 2) {
			setResults([]);
			return;
		}
		debounceRef.current = setTimeout(async () => {
			const res = await searchFn(value);
			if (res.success && res.data) {
				setResults(res.data);
			}
		}, 350);
	}, [searchFn]);

	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: inputField, fieldState }) => {
				const selected: DiagnosisItem[] = Array.isArray(inputField.value)
					? inputField.value
					: [];

				return (
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

						<Combobox
							multiple
							value={selected.map((d) => d.code)}
							filteredItems={results.map((r) => r.code)}
							onValueChange={(codes: string[]) => {
								const kept = selected.filter((d) => codes.includes(d.code));
								const added = results.filter(
									(r) =>
										codes.includes(r.code) &&
										!kept.find((k) => k.code === r.code),
								);
								inputField.onChange([...kept, ...added]);
							}}
						>
							<ComboboxChips ref={anchor} aria-invalid={fieldState.invalid}>
								{selected.map((d) => (
									<ComboboxChip key={d.code} value={d.code}>
										<span className="font-mono text-xs">{d.code}</span>{" "}
										{d.title}
									</ComboboxChip>
								))}
								<ComboboxChipsInput
									placeholder={
										selected.length === 0
											? textGet(
													"form.create_medical_record.diagnoses.placeholder",
												)
											: undefined
									}
									onChange={(e) => handleInputChange(e.target.value)}
									onKeyDown={(e) => {
										if (e.key === "Enter") e.preventDefault();
									}}
								/>
							</ComboboxChips>

							<ComboboxContent anchor={anchor} className="min-w-96">
								<ComboboxList>
									<ComboboxEmpty>
										{textGet("form.create_medical_record.diagnoses.empty")}
									</ComboboxEmpty>
									{results.map((item) => (
										<ComboboxItem key={item.code} value={item.code}>
											<span
												className="font-mono text-xs text-muted-foreground w-24 shrink-0 truncate mr-2"
												title={item.code}
											>
												{item.code}
											</span>
											<span className="truncate" title={item.title}>
												{item.title}
											</span>
										</ComboboxItem>
									))}
								</ComboboxList>
							</ComboboxContent>
						</Combobox>

						{description && <FieldDescription>{description}</FieldDescription>}
						{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
					</Field>
				);
			}}
		/>
	);
}

export { FormIcd11Select };
