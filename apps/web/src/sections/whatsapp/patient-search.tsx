import { useText } from "@pengi/shared";
import { Input, Text } from "@pengi/ui";
import { BellOff, Loader2, Search } from "lucide-react";
import React from "react";
import {
	getAllPatientsWithLastFollowUp,
	type Patient,
} from "@/api/clinical-service";

const SEARCH_DEBOUNCE_MS = 300;
const RESULTS_LIMIT = 8;

export function patientName(
	p: Pick<Patient, "full_name" | "first_name" | "last_name">,
) {
	return p.full_name || `${p.first_name} ${p.last_name}`.trim();
}

interface PatientSearchProps {
	/** Placeholder and label of the search box (an i18n key). */
	placeholderKey: string;
	onPick: (patient: Patient) => void;
	/** The patient being picked shows a spinner; every row is disabled meanwhile. */
	pickingId?: number | null;
	/** Keeps only the patients that pass (e.g. those with a phone). */
	filter?: (patient: Patient) => boolean;
	/** Marks patients without WhatsApp consent. */
	showOptIn?: boolean;
}

/**
 * Debounced patient search with a clickable result list. Mount it inside the
 * dialog's content so closing the dialog resets it.
 */
export function PatientSearch({
	placeholderKey,
	onPick,
	pickingId = null,
	filter,
	showOptIn = false,
}: PatientSearchProps) {
	const { textGet } = useText();
	const [search, setSearch] = React.useState("");
	const [query, setQuery] = React.useState("");
	// Results with the query they answer: searching while they don't match.
	const [results, setResults] = React.useState<{
		query: string;
		patients: Patient[];
	}>({ query: "", patients: [] });
	const searching = query !== "" && results.query !== query;
	const patients = filter ? results.patients.filter(filter) : results.patients;

	// Debounce the typed search into the query that is fetched.
	React.useEffect(() => {
		const id = setTimeout(() => setQuery(search.trim()), SEARCH_DEBOUNCE_MS);
		return () => clearTimeout(id);
	}, [search]);

	React.useEffect(() => {
		if (!query) return;
		let cancelled = false;
		getAllPatientsWithLastFollowUp({
			search: query,
			limit: RESULTS_LIMIT,
		}).then((res) => {
			if (cancelled) return;
			setResults({
				query,
				patients: res.success ? (res.data?.items ?? []) : [],
			});
		});
		return () => {
			cancelled = true;
		};
	}, [query]);

	return (
		<div className="grid gap-2">
			<div className="relative">
				<Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
				<Input
					value={search}
					onChange={(e) => setSearch(e.target.value)}
					placeholder={textGet(placeholderKey)}
					aria-label={textGet(placeholderKey)}
					className="pl-9"
					autoFocus
				/>
			</div>
			<div className="grid max-h-72 gap-1 overflow-y-auto">
				{searching && (
					<Loader2 className="mx-auto my-4 h-5 w-5 animate-spin text-muted-foreground" />
				)}
				{!searching && query && patients.length === 0 && (
					<p className="py-4 text-center text-sm text-muted-foreground">
						<Text uuid="appointments.patient.not_found" />
					</p>
				)}
				{!searching &&
					query &&
					patients.map((p) => (
						<button
							type="button"
							key={p.ID}
							disabled={pickingId !== null}
							onClick={() => onPick(p)}
							className="flex items-center justify-between gap-2 rounded-md px-3 py-2 text-left text-sm hover:bg-accent disabled:opacity-50"
						>
							<span className="grid">
								<span className="font-medium">{patientName(p)}</span>
								<span className="text-xs text-muted-foreground tabular-nums">
									{p.document}
									{p.phone ? ` · ${p.phone}` : ""}
								</span>
							</span>
							{pickingId === p.ID ? (
								<Loader2 className="h-4 w-4 shrink-0 animate-spin" />
							) : (
								showOptIn &&
								!p.whatsapp_opt_in && (
									<span
										className="text-amber-700 dark:text-amber-400"
										title={textGet("whatsapp.inbox.no_opt_in")}
									>
										<BellOff className="h-4 w-4 shrink-0" />
									</span>
								)
							)}
						</button>
					))}
			</div>
		</div>
	);
}
