import { useText } from "@pengi/shared";
import {
	Button,
	Form,
	FormCheckbox,
	FormInput,
	FormRadioGroup,
	FormRow,
} from "@pengi/ui";
import { ArrowRight, BadgeCheck } from "lucide-react";
import {
	birthDateBounds,
	type NewPatientValues,
	newPatientSchema,
	type VerifiedContact,
} from "@/lib/new-patient-flow";

/**
 * Step "Tus datos": what the clinic needs to open the record. The verified
 * contact is filled in and locked; the other one is optional.
 */
export function DetailsForm({
	verified,
	initialValues,
	onSubmit,
}: {
	verified: VerifiedContact;
	initialValues: NewPatientValues;
	onSubmit: (values: NewPatientValues) => void;
}) {
	const { textGet } = useText();
	const bounds = birthDateBounds();
	const verifiedAddon = (
		<BadgeCheck
			aria-label={textGet("portal.new_patient.verified")}
			className="size-5 text-primary"
		/>
	);

	return (
		<Form
			schema={newPatientSchema}
			defaultValues={initialValues}
			onSubmit={onSubmit}
			className="space-y-6"
		>
			{(field) => {
				const documentType = field.watch("document_type");
				const phone = field.watch("phone");
				return (
					<>
						<FormRow>
							<FormInput
								field={field}
								name="first_name"
								label={textGet("portal.new_patient.first_name")}
								autoComplete="given-name"
								maxLength={100}
								className="h-11 text-base"
							/>
							<FormInput
								field={field}
								name="last_name"
								label={textGet("portal.new_patient.last_name")}
								autoComplete="family-name"
								maxLength={100}
								className="h-11 text-base"
							/>
						</FormRow>

						<FormRadioGroup
							field={field}
							name="document_type"
							label={textGet("portal.new_patient.document_type")}
							isRow
							options={[
								{
									value: "cedula",
									label: textGet("portal.new_patient.document_type.cedula"),
								},
								{
									value: "passport",
									label: textGet("portal.new_patient.document_type.passport"),
								},
							]}
						/>
						<FormRow>
							<FormInput
								field={field}
								name="document"
								label={textGet(
									documentType === "passport"
										? "portal.new_patient.document.passport"
										: "portal.new_patient.document.cedula",
								)}
								inputMode={documentType === "passport" ? "text" : "numeric"}
								autoComplete="off"
								autoCapitalize="characters"
								maxLength={20}
								className="h-11 text-base"
							/>
							<FormInput
								field={field}
								name="birth_date"
								type="date"
								label={textGet("portal.new_patient.birth_date")}
								autoComplete="bday"
								min={bounds.min}
								max={bounds.max}
								className="h-11 text-base"
							/>
						</FormRow>

						<FormRow>
							<FormInput
								field={field}
								name="phone"
								type="tel"
								inputMode="tel"
								autoComplete="tel"
								label={textGet("portal.new_patient.phone")}
								isOptional={verified.channel !== "whatsapp"}
								readOnly={verified.channel === "whatsapp"}
								description={
									verified.channel === "whatsapp"
										? textGet("portal.new_patient.verified")
										: undefined
								}
								endAddon={
									verified.channel === "whatsapp" ? verifiedAddon : undefined
								}
								containerClassName="h-11"
								className="text-base"
							/>
							<FormInput
								field={field}
								name="email"
								type="email"
								inputMode="email"
								autoComplete="email"
								label={textGet("portal.new_patient.email")}
								isOptional={verified.channel !== "email"}
								readOnly={verified.channel === "email"}
								description={
									verified.channel === "email"
										? textGet("portal.new_patient.verified")
										: undefined
								}
								endAddon={
									verified.channel === "email" ? verifiedAddon : undefined
								}
								containerClassName="h-11"
								className="text-base"
							/>
						</FormRow>

						{phone.trim() !== "" && (
							<FormCheckbox
								field={field}
								name="whatsapp_opt_in"
								label={textGet("portal.new_patient.whatsapp_opt_in")}
								description={textGet(
									"portal.new_patient.whatsapp_opt_in.description",
								)}
							/>
						)}

						<Button
							type="submit"
							className="h-12 w-full gap-2 rounded-full text-base"
						>
							{textGet("portal.new_patient.continue")}
							<ArrowRight className="size-5" />
						</Button>
					</>
				);
			}}
		</Form>
	);
}
