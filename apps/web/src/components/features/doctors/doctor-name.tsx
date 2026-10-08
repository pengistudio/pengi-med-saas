import type React from "react";
import { findDoctor, useDoctors } from "@/store/doctors-store";

interface DoctorNameProps {
	doctorId?: number | null;
	/** Shown when there is no doctor (e.g. the patient's legacy free-text `medic`). */
	legacyName?: string;
	/** Shown when there is neither. */
	fallback?: React.ReactNode;
	className?: string;
}

/** A doctor's name with their agenda color, looked up in the shared doctors list. */
export function DoctorName({
	doctorId,
	legacyName,
	fallback = null,
	className,
}: DoctorNameProps) {
	const { doctors } = useDoctors(Boolean(doctorId));
	const doctor = findDoctor(doctors, doctorId);
	if (doctor) {
		return (
			<span className={className}>
				<span
					aria-hidden
					className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle"
					style={{ backgroundColor: doctor.color }}
				/>
				{doctor.full_name}
			</span>
		);
	}
	if (legacyName) return <span className={className}>{legacyName}</span>;
	return <>{fallback}</>;
}
