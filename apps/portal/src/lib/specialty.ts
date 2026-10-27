import { useText } from "@pengi/shared";

type WithSpecialty = { specialty: string; specialty_other: string };

/** A specialty for display: the catalog label, or the clinic's own name for "other". */
export function useSpecialtyLabel() {
	const { textGet } = useText();
	return (item: WithSpecialty) =>
		item.specialty === "other"
			? item.specialty_other
			: item.specialty && textGet(`doctors.specialty.${item.specialty}`);
}
