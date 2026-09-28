import { type DateInput, type DateStyle, useText } from "@pengi/shared";

// For column definitions: they are built once (memoized) outside the render,
// so they can't hold formatters; these cells read the interface language
// themselves and re-render when it changes.

export function Money({
	amount,
	className,
}: {
	amount: number | null | undefined;
	className?: string;
}) {
	const { formatMoney } = useText();
	return <span className={className}>{formatMoney(amount)}</span>;
}

export function FormattedDate({
	value,
	style,
	withTime = false,
	className,
}: {
	value: DateInput;
	style?: DateStyle;
	/** Date and time ("27 sept 2026, 2:05 p. m."); `style` is ignored. */
	withTime?: boolean;
	className?: string;
}) {
	const { formatDate, formatDateTime } = useText();
	const text = withTime ? formatDateTime(value) : formatDate(value, style);
	return <span className={className}>{text || "—"}</span>;
}
