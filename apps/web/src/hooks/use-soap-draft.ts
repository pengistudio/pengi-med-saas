import React from "react";

type DraftValues = {
	date?: string;
	motive?: string;
	observation?: string;
	next_appointment_date?: string;
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

function draftKey(patientId: string) {
	return `soap-draft-${patientId}`;
}

function serialize(values: FormValues): string {
	const draft: DraftValues = {
		...values,
		date: values.date?.toISOString(),
		next_appointment_date: values.next_appointment_date?.toISOString(),
	};
	return JSON.stringify(draft);
}

function deserialize(raw: string): FormValues | null {
	try {
		const parsed: DraftValues = JSON.parse(raw);
		return {
			...parsed,
			date: parsed.date ? new Date(parsed.date) : new Date(),
			next_appointment_date: parsed.next_appointment_date
				? new Date(parsed.next_appointment_date)
				: undefined,
		};
	} catch {
		return null;
	}
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
	const restoredRef = React.useRef(false);

	// Restore draft on mount
	React.useEffect(() => {
		if (!patientId || restoredRef.current) return;
		restoredRef.current = true;
		const raw = localStorage.getItem(draftKey(patientId));
		if (!raw) return;
		const values = deserialize(raw);
		if (!values) return;
		form.reset(values as unknown as Partial<TValues>);
		setHasDraft(true);
		setLastSaved(new Date());
	}, [patientId, form]);

	// Watch all values and debounce-save to localStorage
	React.useEffect(() => {
		if (!patientId) return;
		const subscription = form.watch((values) => {
			if (debounceRef.current) clearTimeout(debounceRef.current);
			debounceRef.current = setTimeout(() => {
				try {
					localStorage.setItem(
						draftKey(patientId),
						serialize(values as unknown as FormValues),
					);
					setHasDraft(true);
					setLastSaved(new Date());
				} catch {
					// localStorage full or unavailable — silently ignore
				}
			}, 800);
		});
		return () => {
			subscription.unsubscribe();
			if (debounceRef.current) clearTimeout(debounceRef.current);
		};
	}, [patientId, form]);

	function clearDraft() {
		if (!patientId) return;
		localStorage.removeItem(draftKey(patientId));
		setHasDraft(false);
		setLastSaved(null);
	}

	return { hasDraft, lastSaved, clearDraft };
}
