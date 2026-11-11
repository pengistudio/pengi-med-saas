import { useText } from "@pengi/shared";
import {
	Banknote,
	CircleCheck,
	CircleEllipsis,
	CreditCard,
	Landmark,
	type LucideIcon,
	ScrollText,
	TriangleAlert,
	WalletCards,
} from "lucide-react";
import type { PaymentMethod } from "@/api/cash-service";
import { cn } from "@/lib/utils";

// Shared pieces of the cash screens: the period of a session, the sign of
// outflows, the difference of a close and the icon of each payment method.

const METHOD_ICONS: Record<PaymentMethod, LucideIcon> = {
	cash: Banknote,
	transfer: Landmark,
	credit_card: CreditCard,
	debit_card: WalletCards,
	check: ScrollText,
	other: CircleEllipsis,
};

export function MethodIcon({
	method,
	className,
}: {
	method: PaymentMethod;
	className?: string;
}) {
	const Icon = METHOD_ICONS[method] ?? CircleEllipsis;
	return <Icon aria-hidden className={cn("h-4 w-4 shrink-0", className)} />;
}

/**
 * A session's period, compact: "6 nov 2026, de 9:39 a. m. a 6:10 p. m." when
 * it opened and closed the same day, both date-times otherwise. Without a
 * close, just when it opened.
 */
export function useSessionRange() {
	const { textGet, formatDate, formatDateTime, formatTime } = useText();
	return (openedAt: string, closedAt: string | null | undefined) => {
		if (!closedAt) return formatDateTime(openedAt);
		if (formatDate(openedAt) === formatDate(closedAt)) {
			return textGet("cash.range.same_day", {
				date: formatDate(openedAt),
				start: formatTime(openedAt, { seconds: false }),
				end: formatTime(closedAt, { seconds: false }),
			});
		}
		return textGet("cash.range.days", {
			start: formatDateTime(openedAt),
			end: formatDateTime(closedAt),
		});
	};
}

/**
 * The sign convention of the cash figures: what leaves the drawer (refunds,
 * outflows) reads with a minus and a muted destructive tone; zero has no sign
 * and no tone (formatMoney never writes "-$0,00").
 */
export function Outflow({
	amount,
	className,
}: {
	/** Positive: the amount that left the drawer. */
	amount: number;
	className?: string;
}) {
	const { formatMoney } = useText();
	const cents = Math.round(amount * 100);
	return (
		<span
			className={cn(
				"tabular-nums",
				cents !== 0 && "text-destructive/90",
				className,
			)}
		>
			{formatMoney(cents === 0 ? 0 : -amount)}
		</span>
	);
}

type DifferenceTone = "balanced" | "over" | "short";

function differenceTone(value: number): DifferenceTone {
	const cents = Math.round(value * 100);
	if (cents === 0) return "balanced";
	return cents > 0 ? "over" : "short";
}

const TONE_CLASSES: Record<DifferenceTone, string> = {
	balanced:
		"border-emerald-600/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
	over: "border-amber-600/25 bg-amber-500/10 text-amber-700 dark:text-amber-400",
	short: "border-destructive/30 bg-destructive/10 text-destructive",
};

/**
 * The difference of a close as text plus colour (never colour alone):
 * "Cuadra", "Sobrante de $2,00" or "Faltante de $1,50".
 */
export function DifferenceBadge({
	value,
	className,
}: {
	value: number | null | undefined;
	className?: string;
}) {
	const { textGet, formatMoney } = useText();
	if (value === null || value === undefined) return null;
	const tone = differenceTone(value);
	const amount = formatMoney(Math.abs(value));
	const label =
		tone === "balanced"
			? textGet("cash.difference.balanced")
			: tone === "over"
				? textGet("cash.difference.over", { amount })
				: textGet("cash.difference.short", { amount });
	const Icon = tone === "balanced" ? CircleCheck : TriangleAlert;
	return (
		<span
			className={cn(
				"inline-flex items-center gap-1.5 whitespace-nowrap rounded-full border px-2.5 py-0.5 text-xs font-medium tabular-nums",
				TONE_CLASSES[tone],
				className,
			)}
		>
			<Icon aria-hidden className="h-3.5 w-3.5" />
			{label}
		</span>
	);
}

/** An empty list with a direction: what's missing and how it gets filled. */
export function CashEmptyState({
	icon: Icon,
	title,
	hint,
}: {
	icon: LucideIcon;
	title: string;
	hint?: string;
}) {
	return (
		<div className="mx-auto flex max-w-sm flex-col items-center gap-2 py-6 text-center">
			<span className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
				<Icon aria-hidden className="h-5 w-5" />
			</span>
			<p className="text-sm font-medium">{title}</p>
			{hint && (
				<p className="text-pretty text-sm text-muted-foreground">{hint}</p>
			)}
		</div>
	);
}
