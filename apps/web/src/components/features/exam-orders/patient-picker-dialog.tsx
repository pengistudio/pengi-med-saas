import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	Input,
	Spinner,
} from "@pengi/ui";
import { ChevronRight, Search } from "lucide-react";
import React from "react";
import {
	getAllPatientsWithLastFollowUp,
	type Patient,
} from "@/api/clinical-service";
import { patientDisplayName } from "@/lib/exam-orders";

const SEARCH_LIMIT = 10;
const SEARCH_DEBOUNCE_MS = 300;

interface PatientPickerDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onPick: (patient: Patient) => void;
}

/** Searches a patient by name or document and hands back the chosen one. */
export function PatientPickerDialog({
	open,
	onOpenChange,
	onPick,
}: PatientPickerDialogProps) {
	const { textGet } = useText();
	const [search, setSearch] = React.useState("");
	const [query, setQuery] = React.useState("");
	const [patients, setPatients] = React.useState<Patient[]>([]);
	const [loading, setLoading] = React.useState(false);

	// Debounce the typed text into the query sent to the server.
	React.useEffect(() => {
		const timer = setTimeout(() => setQuery(search.trim()), SEARCH_DEBOUNCE_MS);
		return () => clearTimeout(timer);
	}, [search]);

	// Show the spinner as soon as the query changes (adjust state during render).
	const [prevQuery, setPrevQuery] = React.useState(query);
	if (prevQuery !== query) {
		setPrevQuery(query);
		setLoading(query.length > 0);
		if (!query) setPatients([]);
	}

	React.useEffect(() => {
		if (!open || !query) return;
		let cancelled = false;
		getAllPatientsWithLastFollowUp({ search: query, limit: SEARCH_LIMIT }).then(
			(res) => {
				if (cancelled) return;
				setPatients(res.success ? (res.data?.items ?? []) : []);
				setLoading(false);
			},
		);
		return () => {
			cancelled = true;
		};
	}, [open, query]);

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle>
						{textGet("clinical.exam_orders.patient_picker.title")}
					</DialogTitle>
					<DialogDescription>
						{textGet("clinical.exam_orders.patient_picker.description")}
					</DialogDescription>
				</DialogHeader>
				<div className="relative">
					<Search className="pointer-events-none absolute top-1/2 left-2.5 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						type="search"
						autoFocus
						value={search}
						onChange={(e) => setSearch(e.target.value)}
						placeholder={textGet("clinical.exam_orders.patient_picker.search")}
						aria-label={textGet("clinical.exam_orders.patient_picker.search")}
						className="pl-8"
					/>
				</div>
				<div className="min-h-24">
					{loading ? (
						<div className="flex justify-center py-6">
							<Spinner />
						</div>
					) : query && patients.length === 0 ? (
						<p className="py-6 text-center text-sm text-muted-foreground">
							{textGet("clinical.exam_orders.patient_picker.empty")}
						</p>
					) : (
						<ul className="divide-y">
							{patients.map((patient) => (
								<li key={patient.ID}>
									<button
										type="button"
										className="flex w-full items-center gap-3 px-2 py-2.5 text-left text-sm hover:bg-muted/50"
										onClick={() => onPick(patient)}
									>
										<span className="min-w-0 flex-1">
											<span className="block truncate font-medium">
												{patientDisplayName(patient)}
											</span>
											<span className="block text-xs text-muted-foreground">
												{patient.document}
											</span>
										</span>
										<ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
									</button>
								</li>
							))}
						</ul>
					)}
				</div>
			</DialogContent>
		</Dialog>
	);
}
