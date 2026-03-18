import { zodResolver } from "@hookform/resolvers/zod";
import React from "react";
import type {
	DefaultValues,
	FieldErrors,
	FieldValues,
	UseFormReturn,
} from "react-hook-form";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import type { z } from "zod";

import { useLanguage } from "@/contexts/language-context";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";

function countLeafErrors(errors: FieldErrors): number {
	let count = 0;
	for (const key of Object.keys(errors)) {
		const error = errors[key];
		if (!error) continue;
		if (typeof (error as { message?: unknown }).message === "string") {
			count++;
		} else {
			count += countLeafErrors(error as FieldErrors);
		}
	}
	return count;
}

type FormProps<
	T extends z.ZodType<Output, Input>, // <- aquí T depende de Output e Input
	Output = z.output<T>, // <- Output se infiere a partir de T
	Input extends FieldValues = z.input<T>, // <- Input también se infiere
> = {
	schema: T;
	defaultValues?: DefaultValues<z.input<T>>;
	children?: (
		methods: UseFormReturn<z.input<T>, unknown, Output>,
	) => React.ReactNode;
	onSubmit: (values: Output) => void;
	onUpdateValuesCallback?: (values: Input) => void;
	className?: string;
};

export const Form = <
	T extends z.ZodType<Output, Input>,
	Output = z.output<T>,
	Input extends FieldValues = z.input<T>,
>(
	props: FormProps<T, Output, Input>,
) => {
	const {
		schema,
		defaultValues,
		children,
		onSubmit,
		onUpdateValuesCallback,
		className,
	} = props;

	const { currentLanguage } = useLanguage();
	const { textGet } = useText();

	const form = useForm({
		defaultValues,
		resolver: zodResolver(schema),
	});

	// Opcional: al cambiar idioma, limpia y revalida para refrescar mensajes
	React.useEffect(() => {
		if (onUpdateValuesCallback) {
			const subscription = form.watch((value) => {
				onUpdateValuesCallback(value as Input);
			});
			return () => subscription.unsubscribe();
		}
	}, [form, onUpdateValuesCallback]);

	const { isSubmitted } = form.formState;

	React.useEffect(() => {
		if (currentLanguage && isSubmitted) {
			form.trigger(); // revalida todos los campos si ya se intentó enviar
		}
	}, [currentLanguage, form, isSubmitted]);

	// Función para manejar errores de validación
	const handleInvalidSubmit = React.useCallback(
		(errors: FieldErrors<Input>) => {
			const count = countLeafErrors(errors);
			const description =
				count === 1
					? textGet("form.validation.single_field_error")
					: textGet("form.validation.multiple_fields_error").replace(
							"{count}",
							String(count),
						);

			toast.error(textGet("form.validation.invalid_fields"), {
				description,
				duration: 5000,
			});

			console.error("❌ Form validation errors:", errors);
		},
		[textGet],
	);

	if (!children) return null;

	return (
		<FormProvider {...form}>
			<form
				onSubmit={form.handleSubmit(onSubmit, handleInvalidSubmit)}
				className={cn(className)}
			>
				{children(form)}
			</form>
		</FormProvider>
	);
};
