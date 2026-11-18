import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Form,
	FormInput,
} from "@pengi/ui";
import { Loader2, Save } from "lucide-react";
import { z } from "zod";
import type { Customer } from "@/api/billing-service";

const formSchema = z.object({
	identification: z
		.string()
		.trim()
		.min(1, "billing.customer.form.error.required"),
	name: z.string().trim().min(1, "billing.customer.form.error.required"),
	address: z.string().optional(),
	email: z.union([z.literal(""), z.email("billing.customer.form.error.email")]),
	phone: z.string().optional(),
});

export type CustomerFormValues = z.infer<typeof formSchema>;

export interface CustomerFormProps {
	initialData?: Customer; // Present when editing
	onSubmit: (values: CustomerFormValues) => Promise<void>;
	loading?: boolean;
	/** Fields + submit button only (inside a dialog), without the page card. */
	embedded?: boolean;
}

const CustomerForm = ({
	initialData,
	onSubmit,
	loading = false,
	embedded = false,
}: CustomerFormProps) => {
	const { textGet } = useText();
	const isEditing = !!initialData;

	const defaultValues: CustomerFormValues = {
		identification: initialData?.identification || "",
		name: initialData?.name || "",
		address: initialData?.address || "",
		email: initialData?.email || "",
		phone: initialData?.phone || "",
	};

	const submitButton = (
		<Button type="submit" disabled={loading}>
			{loading ? (
				<Loader2 className="mr-2 h-4 w-4 animate-spin" />
			) : (
				<Save className="mr-2 h-4 w-4" />
			)}
			{textGet(
				isEditing
					? "billing.customer.form.submit.edit"
					: "billing.customer.form.submit.create",
			)}
		</Button>
	);

	return (
		<Form schema={formSchema} onSubmit={onSubmit} defaultValues={defaultValues}>
			{(field) => {
				const fields = (
					<div className="space-y-4">
						<div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="identification"
								placeholder={textGet("billing.customer.form.identification")}
								label={textGet("billing.customer.form.identification")}
							/>
							<FormInput
								field={field}
								name="name"
								placeholder={textGet("billing.customer.form.name")}
								label={textGet("billing.customer.form.name")}
							/>
						</div>
						<FormInput
							field={field}
							name="address"
							placeholder={textGet("billing.customer.form.address")}
							label={textGet("billing.customer.form.address")}
							isOptional
						/>
						<div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
							<FormInput
								field={field}
								name="email"
								type="email"
								placeholder={textGet("billing.customer.form.email")}
								label={textGet("billing.customer.form.email")}
								isOptional
							/>
							<FormInput
								field={field}
								name="phone"
								placeholder={textGet("billing.customer.form.phone")}
								label={textGet("billing.customer.form.phone")}
								isOptional
							/>
						</div>
					</div>
				);

				if (embedded) {
					return (
						<div className="space-y-4">
							{fields}
							<div className="flex justify-end">{submitButton}</div>
						</div>
					);
				}

				return (
					<Card className="max-w-4xl mx-auto">
						<CardHeader>
							<CardTitle>
								{textGet(
									isEditing
										? "billing.customer.form.edit.title"
										: "billing.customer.form.create.title",
								)}
							</CardTitle>
							<CardDescription>
								{textGet(
									isEditing
										? "billing.customer.form.edit.description"
										: "billing.customer.form.create.description",
								)}
							</CardDescription>
						</CardHeader>
						<CardContent>{fields}</CardContent>
						<CardFooter>{submitButton}</CardFooter>
					</Card>
				);
			}}
		</Form>
	);
};

export default CustomerForm;
