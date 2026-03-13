import { useState } from "react";
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
import { Input } from "@/components/ui/input";
import { useText } from "@/hooks/use-text";

type FormTagInputProps<
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
> = Omit<
	React.InputHTMLAttributes<HTMLInputElement>,
	"name" | "value" | "onChange"
> & {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label?: string;
	isOptional?: boolean;
	description?: string;
};

function FormTagInput<
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
	...props
}: FormTagInputProps<T, Output, Input>) {
	const { textGet } = useText();
	const [inputValue, setInputValue] = useState("");

	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: inputField, fieldState }) => {
				const tags: string[] = inputField.value
					? (inputField.value as string)
							.split(",")
							.map((t: string) => t.trim())
							.filter(Boolean)
					: [];

				const addTag = (tag: string) => {
					const trimmed = tag.trim();
					if (!trimmed || tags.includes(trimmed)) return;
					const newTags = [...tags, trimmed];
					inputField.onChange(newTags.join(", "));
					setInputValue("");
				};

				const removeTag = (index: number) => {
					const newTags = tags.filter((_, i) => i !== index);
					inputField.onChange(newTags.join(", "));
				};

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

						<div className="flex flex-col gap-2">
							{tags.length > 0 && (
								<div className="flex flex-wrap gap-1.5">
									{tags.map((tag) => (
										<span
											key={tag}
											className="inline-flex items-center gap-1 rounded-full bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300 px-2.5 py-0.5 text-xs font-medium"
										>
											{tag}
											<button
												type="button"
												onClick={() => removeTag(index)}
												className="rounded-full hover:bg-amber-200 dark:hover:bg-amber-800 p-0.5 transition-colors"
												aria-label={`Eliminar ${tag}`}
											>
												<svg
													xmlns="http://www.w3.org/2000/svg"
													width="10"
													height="10"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													strokeWidth="2.5"
													strokeLinecap="round"
													strokeLinejoin="round"
													aria-hidden="true"
												>
													<path d="M18 6 6 18" />
													<path d="m6 6 12 12" />
												</svg>
											</button>
										</span>
									))}
								</div>
							)}
							<Input
								{...props}
								id={name}
								value={inputValue}
								placeholder={placeholder}
								aria-invalid={fieldState.invalid}
								onChange={(e) => setInputValue(e.target.value)}
								onKeyDown={(e) => {
									if (e.key === "Enter" || e.key === ",") {
										e.preventDefault();
										addTag(inputValue);
									} else if (
										e.key === "Backspace" &&
										inputValue === "" &&
										tags.length > 0
									) {
										removeTag(tags.length - 1);
									}
								}}
								onBlur={() => {
									if (inputValue.trim()) {
										addTag(inputValue);
									}
									inputField.onBlur();
								}}
							/>
						</div>

						{description && <FieldDescription>{description}</FieldDescription>}
						{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
					</Field>
				);
			}}
		/>
	);
}

export { FormTagInput, type FormTagInputProps };
