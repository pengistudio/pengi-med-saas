import { useText } from "@pengi/shared";
import {
	Field,
	FieldLabel,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import React from "react";
import { type UseFormReturn, useWatch } from "react-hook-form";
import type { AppointmentFormValues } from "@/components/features/appointments/appointment-utils";
import { useAppointmentTypes } from "@/store/agenda-store";
import { addMinutes, typesForDoctor } from "./agenda-utils";

const NONE = "none";

/**
 * Appointment type of the appointment form, among the types the chosen doctor
 * offers. Picking one sets the end time to start + duration; while the user
 * hasn't typed an end time of their own, moving the start moves the end too.
 * A doctor with exactly one type gets it preselected on a new appointment.
 * Renders nothing when the tenant has no types for that doctor.
 */
export function AppointmentTypeField({
	field,
	isEdit,
	currentTypeId,
}: {
	field: UseFormReturn<AppointmentFormValues>;
	isEdit: boolean;
	/** The appointment's type: kept as an option even if inactive. */
	currentTypeId?: number | null;
}) {
	const { textGet } = useText();
	const { types, loaded } = useAppointmentTypes();
	const { control, getValues, setValue } = field;
	const doctorId = useWatch({ control, name: "doctor_id" });
	const typeId = useWatch({ control, name: "appointment_type_id" });
	const startTime = useWatch({ control, name: "start_time" });

	const options = React.useMemo(
		() => typesForDoctor(types, doctorId, currentTypeId),
		[types, doctorId, currentTypeId],
	);
	const duration = (id: number | null | undefined) =>
		types.find((t) => t.ID === id)?.duration_minutes;

	// The end time we last computed: while the field still holds it, the user
	// hasn't edited it and a new start recomputes it.
	const autoEnd = React.useRef<string | null>(null);
	const applyType = React.useCallback(
		(id: number | null) => {
			setValue("appointment_type_id", id, { shouldDirty: true });
			const minutes = types.find((t) => t.ID === id)?.duration_minutes;
			const start = getValues("start_time");
			if (minutes && start) {
				const end = addMinutes(start, minutes);
				setValue("end_time", end, { shouldDirty: true });
				autoEnd.current = end;
			}
		},
		[types, getValues, setValue],
	);

	// Once the types are in: an edited appointment whose end matches its
	// type's duration keeps following the start.
	const initialized = React.useRef(false);
	React.useEffect(() => {
		if (!loaded || initialized.current) return;
		initialized.current = true;
		const minutes = duration(getValues("appointment_type_id"));
		const start = getValues("start_time");
		const end = getValues("end_time");
		if (minutes && start && end === addMinutes(start, minutes)) {
			autoEnd.current = end;
		}
	});

	// Doctor changed (or types arrived): drop a type the doctor doesn't offer,
	// preselect the only one on a new appointment.
	const lastDoctor = React.useRef<string | null>(null);
	React.useEffect(() => {
		if (!loaded) return;
		const key = String(doctorId ?? "");
		if (lastDoctor.current === key) return;
		lastDoctor.current = key;
		const current = getValues("appointment_type_id");
		const offered = current && options.some((t) => t.ID === current);
		if (current && !offered) setValue("appointment_type_id", null);
		if ((!current || !offered) && !isEdit && options.length === 1) {
			applyType(options[0].ID);
		}
	}, [loaded, doctorId, options, isEdit, getValues, setValue, applyType]);

	// Start moved: the end follows while it is still the computed one.
	const lastStart = React.useRef(startTime);
	React.useEffect(() => {
		if (lastStart.current === startTime) return;
		lastStart.current = startTime;
		const minutes = duration(getValues("appointment_type_id"));
		if (!minutes || !startTime) return;
		if (autoEnd.current === null || autoEnd.current !== getValues("end_time"))
			return;
		const end = addMinutes(startTime, minutes);
		setValue("end_time", end, { shouldDirty: true });
		autoEnd.current = end;
	});

	if (!loaded || options.length === 0) return null;

	const selected = typeId ? String(typeId) : NONE;
	const label = (id: number) => {
		const type = options.find((t) => t.ID === id);
		if (!type) return "";
		return `${type.name} · ${textGet("agenda.types.duration_value", {
			minutes: type.duration_minutes,
		})}${type.active ? "" : ` (${textGet("agenda.types.inactive")})`}`;
	};

	return (
		<Field className="flex flex-col">
			<FieldLabel>
				{textGet("agenda.types.field.type")}
				<span className="text-xs font-normal text-muted-foreground">
					({textGet("form.optional")})
				</span>
			</FieldLabel>
			<Select
				value={selected}
				onValueChange={(v) =>
					!v || v === NONE
						? setValue("appointment_type_id", null, { shouldDirty: true })
						: applyType(Number(v))
				}
			>
				<SelectTrigger>
					<SelectValue>
						{selected === NONE
							? textGet("agenda.types.none")
							: label(Number(selected))}
					</SelectValue>
				</SelectTrigger>
				<SelectContent>
					<SelectItem value={NONE}>{textGet("agenda.types.none")}</SelectItem>
					{options.map((t) => (
						<SelectItem key={t.ID} value={String(t.ID)}>
							{label(t.ID)}
						</SelectItem>
					))}
				</SelectContent>
			</Select>
		</Field>
	);
}
