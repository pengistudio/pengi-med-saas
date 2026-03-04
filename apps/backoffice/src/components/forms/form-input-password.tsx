import { EyeIcon, EyeOffIcon } from "lucide-react";
import { forwardRef, useState } from "react";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type z from "zod";
import { Button } from "@/components/ui/button";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
} from "@/components/ui/field";

import {
	InputGroup,
	InputGroupAddon,
	InputGroupInput,
} from "@/components/ui/input-group";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";

const PasswordInput = forwardRef<
	HTMLInputElement,
	React.ComponentProps<"input">
>(({ className, ...props }, ref) => {
	const [showPassword, setShowPassword] = useState(false);
	const isValueEmpty = props.value === "" || props.value === undefined;
	const isButtonDisabled = isValueEmpty || props.disabled;

	return (
		<InputGroup>
			<InputGroupInput
				className={cn("hide-password-toggle", className)}
				ref={ref}
				type={showPassword ? "text" : "password"}
				{...props}
				autoComplete="current-password"
			/>
			<InputGroupAddon align="inline-end">
				<Button
					className={cn(
						"h-full px-2 py-0",
						isButtonDisabled
							? "opacity-50 cursor-not-allowed"
							: "hover:cursor-pointer",
					)}
					disabled={props.disabled}
					onClick={(e) => {
						if (isButtonDisabled) {
							e.preventDefault();
							return;
						}
						setShowPassword((prev) => !prev);
					}}
					size="icon-xs"
					type="button"
					variant="ghost"
					tabIndex={-1}
				>
					{showPassword && !isButtonDisabled ? (
						<EyeIcon aria-hidden="true" className="h-4 w-4" />
					) : (
						<EyeOffIcon aria-hidden="true" className="h-4 w-4" />
					)}
					<span className="sr-only">
						{showPassword ? "Hide password" : "Show password"}
					</span>
				</Button>
			</InputGroupAddon>

			{/* hides browsers password toggles */}
			<style>{`
					.hide-password-toggle::-ms-reveal,
					.hide-password-toggle::-ms-clear {
						visibility: hidden;
						pointer-events: none;
						display: none;
					}
				`}</style>
		</InputGroup>
	);
});

PasswordInput.displayName = "PasswordInput";

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
};

function FormPasswordInput<
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
}: FormInputProps<T, Output, Input>) {
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

					<PasswordInput
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

export { PasswordInput, FormPasswordInput };
