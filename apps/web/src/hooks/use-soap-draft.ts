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

/** A saved draft found when opening the form, waiting for the user to decide. */
export type PendingDraft = { savedAt: Date };

/**
 * Autosaves the consultation form per patient. A draft found on open is not
 * loaded on its own: it stays pending until the user resumes it or starts a
 * new consultation, so an unfinished one never shows up unannounced.
 */
export function useSoapDraft<TValues>(
	patientId: string | null,
	form: {
		watch: (callback: (values: TValues) => void) => { unsubscribe: () => void };
		reset: (values: Partial<TValues>) => void;
	},
) {
	const [hasDraft, setHasDraft] = React.useState(false);
	const [lastSaved, setLastSaved] = React.useState<Date | null>(null);
	const [pendingDraft, setPendingDraft] = React.useState<
		(PendingDraft & { values: FormValues }) | null
	>(null);
	// Autosave stays off until the pending draft is resolved, so nothing
	// overwrites it before the user decides.
	const pendingRef = React.useRef(false);
	const debounceRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);

	// Look for a draft whenever we switch patients. Guarded only by the
	// per-invocation `cancelled` flag, which is StrictMode-safe (a persistent
	// "already fetched" ref would make the second dev-mode run skip the fetch).
	React.useEffect(() => {
		if (!patientId) return;

		let cancelled = false;
		pendingRef.current = true;
		(async () => {
			const result = await getMedicalRecordDraft(Number(patientId));
			if (cancelled) return;
			if (!result.success || !result.data) {
				pendingRef.current = false;
				setPendingDraft(null);
				setHasDraft(false);
				setLastSaved(null);
				return;
			}
			setPendingDraft({
				values: fromDraftValues(result.data.data as DraftValues),
				savedAt: new Date(result.data.UpdatedAt ?? result.data.CreatedAt),
			});
		})();

		return () => {
			cancelled = true;
		};
	}, [patientId]);

	// Watch all values and debounce-save to the backend
	React.useEffect(() => {
		if (!patientId) return;
		const subscription = form.watch((values) => {
			if (pendingRef.current) return;
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

	/** Loads the pending draft into the form and resumes autosaving. */
	function resumeDraft() {
		if (!pendingDraft) return;
		form.reset(pendingDraft.values as unknown as Partial<TValues>);
		setHasDraft(true);
		setLastSaved(pendingDraft.savedAt);
		setPendingDraft(null);
		pendingRef.current = false;
	}

	/** Deletes the pending draft; the form keeps its empty defaults. */
	async function discardPendingDraft() {
		setPendingDraft(null);
		await clearDraft();
		pendingRef.current = false;
	}

	return {
		hasDraft,
		lastSaved,
		clearDraft,
		pendingDraft: pendingDraft ? { savedAt: pendingDraft.savedAt } : null,
		resumeDraft,
		discardPendingDraft,
	};
}
