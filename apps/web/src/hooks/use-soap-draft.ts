import React from "react";
import {
	deleteMedicalRecordDraft,
	getMedicalRecordDraft,
	saveMedicalRecordDraft,
} from "@/api/clinical-service";

type DraftValues = {
	date?: string;
	motive?: string;
	observation?: string;
	next_appointment_date?: string;
	next_appointment_status?: "scheduled" | "pending" | "not_required";
	soap_record?: {
		subjective?: string;
		objective?: string;
		assessment?: string;
		plan?: string;
	};
	prescription?: {
		content?: string;
		indications?: string;
		items?: Array<{
			medication: string;
			dose: string;
			frequency: string;
			duration: string;
			notes?: string;
		}>;
	};
	vital_signs?: {
		weight?: number | null;
		height?: number | null;
		blood_pressure?: string;
		temperature?: number | null;
		heart_rate?: number | null;
		o2_saturation?: number | null;
	};
	diagnoses?: Array<{ code: string; title: string }>;
};

type FormValues = {
	date: Date;
	motive?: string;
	observation?: string;
	next_appointment_date?: Date;
	next_appointment_status?: "scheduled" | "pending" | "not_required";
	soap_record?: {
		subjective?: string;
		objective?: string;
		assessment?: string;
		plan?: string;
	};
	prescription?: {
		content?: string;
		indications?: string;
		items?: Array<{
			medication: string;
			dose: string;
			frequency: string;
			duration: string;
			notes?: string;
		}>;
	};
	vital_signs?: {
		weight?: number | null;
		height?: number | null;
		blood_pressure?: string;
		temperature?: number | null;
		heart_rate?: number | null;
		o2_saturation?: number | null;
	};
	diagnoses?: Array<{ code: string; title: string }>;
};

function toDraftValues(values: FormValues): DraftValues {
	return {
		...values,
		date: values.date?.toISOString(),
		next_appointment_date: values.next_appointment_date?.toISOString(),
	};
}

function fromDraftValues(parsed: DraftValues): FormValues {
	return {
		...parsed,
		date: parsed.date ? new Date(parsed.date) : new Date(),
		next_appointment_date: parsed.next_appointment_date
			? new Date(parsed.next_appointment_date)
			: undefined,
	};
}

export function useSoapDraft<TValues>(
	patientId: string | null,
	form: {
		watch: (callback: (values: TValues) => void) => { unsubscribe: () => void };
		reset: (values: Partial<TValues>) => void;
	},
) {
	const [hasDraft, setHasDraft] = React.useState(false);
	const [lastSaved, setLastSaved] = React.useState<Date | null>(null);
	const debounceRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);

	// Restore draft whenever we switch patients. Deliberately no persistent
	// "already restored" ref here: combined with the cancellation flag below,
	// that pattern breaks under React StrictMode's dev-mode double-invoke
	// (effect runs, cleanup marks the first run's fetch as cancelled, second
	// run sees the ref already set and skips fetching entirely — so the
	// draft is fetched but its result is always thrown away). Re-running the
	// fetch per invocation and only guarding via the per-invocation `cancelled`
	// flag is the standard StrictMode-safe pattern.
	React.useEffect(() => {
		if (!patientId) return;

		let cancelled = false;
		(async () => {
			const result = await getMedicalRecordDraft(Number(patientId));
			if (cancelled) return;
			if (!result.success || !result.data) {
				setHasDraft(false);
				setLastSaved(null);
				return;
			}
			const values = fromDraftValues(result.data.data as DraftValues);
			form.reset(values as unknown as Partial<TValues>);
			setHasDraft(true);
			setLastSaved(new Date(result.data.UpdatedAt ?? result.data.CreatedAt));
		})();

		return () => {
			cancelled = true;
		};
	}, [patientId, form]);

	// Watch all values and debounce-save to the backend
	React.useEffect(() => {
		if (!patientId) return;
		const subscription = form.watch((values) => {
			if (debounceRef.current) clearTimeout(debounceRef.current);
			debounceRef.current = setTimeout(async () => {
				const draft = toDraftValues(values as unknown as FormValues);
				const result = await saveMedicalRecordDraft(Number(patientId), draft);
				if (result.success) {
					setHasDraft(true);
					setLastSaved(new Date());
				}
				// silent on failure — no toast spam while typing
			}, 800);
		});
		return () => {
			subscription.unsubscribe();
			if (debounceRef.current) clearTimeout(debounceRef.current);
		};
	}, [patientId, form]);

	async function clearDraft() {
		if (!patientId) return;
		await deleteMedicalRecordDraft(Number(patientId));
		setHasDraft(false);
		setLastSaved(null);
	}

	return { hasDraft, lastSaved, clearDraft };
}
