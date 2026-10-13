import { useText } from "@pengi/shared";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import React from "react";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import type { Doctor } from "@/api/doctors-service";
import { cn } from "@/lib/utils";
import { findDoctor, useDoctorStatus, useDoctors } from "@/store/doctors-store";
import { specialtyLabel } from "./doctor-utils";

const NONE = "none";

export interface DoctorSelectProps {
	value: number | null | undefined;
	onChange: (doctorId: number | null) => void;
	/**
	 * Defaults tried after the current user's own profile, in order (e.g. the
	 * appointment's doctor, then the patient's médico de cabecera).
	 */
	fallbacks?: (number | null | undefined)[];
	/**
	 * Fill a default when empty (create forms). Off on edit forms, so a row
	 * without a doctor isn't silently assigned one. "single": only when there
	 * is exactly one active doctor (a new patient's médico de cabecera).
	 */
	autoDefault?: boolean | "single";
	/** Optional choice with a "none" entry (médico de cabecera). */
	allowNone?: boolean;
	label?: string;
	description?: string;
	error?: string;
	disabled?: boolean;
	className?: string;
}

/**
 * Picks the doctor of a record, appointment or document among the active
 * doctors. With exactly one active doctor it renders nothing and fills it in
 * (a consultorio never sees it); with none it renders nothing. A doctor that is
 * assigned but inactive still shows, so an edit doesn't lose it.
 */
export function DoctorSelect({
	value,
	onChange,
	fallbacks = [],
	autoDefault = true,
	allowNone = false,
	label,
	description,
	error,
	disabled,
	className,
}: DoctorSelectProps) {
	const { textGet } = useText();
	const { doctors, activeDoctors, loaded } = useDoctors();
	const status = useDoctorStatus();
	const single = activeDoctors.length === 1 ? activeDoctors[0] : null;

	// The default is applied once: after that an empty value is the user's choice.
	const defaulted = React.useRef(false);
	const fallbackKey = fallbacks.join(",");
	React.useEffect(() => {
		if (defaulted.current || !autoDefault || !loaded) return;
		if (value) {
			defaulted.current = true;
			return;
		}
		// Wait for the status (own profile) unless there is only one choice.
		if (autoDefault === "single") {
			defaulted.current = true;
			if (single) onChange(single.ID);
			return;
		}
		if (!single && !status) return;
		defaulted.current = true;
		const active = new Set(activeDoctors.map((d) => d.ID));
		const candidates = [
			single?.ID,
			status?.doctor?.ID,
			...fallbackKey.split(",").map(Number),
		];
		const pick = candidates.find((id) => id && active.has(id));
		if (pick) onChange(pick);
	}, [
		autoDefault,
		loaded,
		value,
		single,
		status,
		activeDoctors,
		fallbackKey,
		onChange,
	]);

	if (!loaded) return null;
	const current = findDoctor(doctors, value);
	if (activeDoctors.length === 0 && !current) return null;
	// One active doctor: it is the answer (an optional empty field still shows,
	// so an existing row without a doctor can get one).
	if (single && (value === single.ID || (!value && !allowNone))) return null;

	const optionLabel = (d: Doctor) =>
		`${d.full_name} · ${specialtyLabel(d, textGet)}${
			d.active ? "" : ` (${textGet("doctors.status.inactive")})`
		}`;
	const options = [
		...(allowNone
			? [{ value: NONE, label: textGet("doctors.select.none") }]
			: []),
		...activeDoctors.map((d) => ({
			value: String(d.ID),
			label: optionLabel(d),
		})),
		...(current && !current.active
			? [{ value: String(current.ID), label: optionLabel(current) }]
			: []),
	];
	const selected = value ? String(value) : allowNone ? NONE : "";

	return (
		<Field data-invalid={!!error} className={cn("flex flex-col", className)}>
			<FieldLabel>
				{label ?? textGet("doctors.select.label")}
				{allowNone && (
					<span className="text-xs font-normal text-muted-foreground">
						({textGet("form.optional")})
					</span>
				)}
			</FieldLabel>
			<Select
				value={selected}
				disabled={disabled}
				onValueChange={(v) => onChange(!v || v === NONE ? null : Number(v))}
			>
				<SelectTrigger aria-invalid={!!error}>
					<SelectValue placeholder={textGet("doctors.select.placeholder")}>
						{options.find((o) => o.value === selected)?.label ??
							textGet("doctors.select.placeholder")}
					</SelectValue>
				</SelectTrigger>
				<SelectContent>
					{options.map((o) => (
						<SelectItem key={o.value} value={o.value}>
							{o.label}
						</SelectItem>
					))}
				</SelectContent>
			</Select>
			{description && <FieldDescription>{description}</FieldDescription>}
			{error && <FieldError errors={[{ message: textGet(error) }]} />}
		</Field>
	);
}

interface FormDoctorSelectProps<T extends FieldValues>
	extends Omit<DoctorSelectProps, "value" | "onChange" | "error"> {
	field: UseFormReturn<T>;
	name: Path<T>;
}

/** `DoctorSelect` bound to a react-hook-form field holding `number | null`. */
export function FormDoctorSelect<T extends FieldValues>({
	field,
	name,
	...props
}: FormDoctorSelectProps<T>) {
	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: { value, onChange }, fieldState }) => (
				<DoctorSelect
					{...props}
					value={value as number | null | undefined}
					onChange={onChange}
					error={fieldState.error?.message}
				/>
			)}
		/>
	);
}
