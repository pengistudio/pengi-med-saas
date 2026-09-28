import { useText } from "@pengi/shared";
import { SUBSCRIPTION_TIME_ZONE } from "./term";

/** An expiry timestamp from the API, displayed as its Ecuador date. */
export function ExpiryDate({ expiresAt }: { expiresAt: string }) {
	const { formatDate } = useText();
	return (
		<>{formatDate(expiresAt, "medium", { timeZone: SUBSCRIPTION_TIME_ZONE })}</>
	);
}
