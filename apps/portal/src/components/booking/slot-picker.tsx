import { useText } from "@pengi/shared";
import { Button, cn, Skeleton, Spinner } from "@pengi/ui";
import { CalendarSearch, Plus } from "lucide-react";
import React from "react";
import {
	type BookingTarget,
	getBookingSlots,
	type SlotDay,
	type SlotsLoader,
} from "@/api/portal-service";
import { addDays, slotRange } from "@/lib/booking-flow";

type Page =
	| { status: "loading" }
	| { status: "error" }
	| { status: "ready"; days: SlotDay[]; nextFrom?: string };

/**
 * Days with free times, 14 at a time ("more days" adds the next 14), and the
 * free times of the chosen day. Picking a time hands the slot back.
 */
export function SlotPicker({
	typeId,
	target,
	minDate,
	maxDate,
	initialDate,
	loadSlots = getBookingSlots,
	emptyAction,
	onPick,
}: {
	typeId: number;
	target: BookingTarget;
	minDate: string;
	maxDate: string;
	/** The day to open on when it has free times (e.g. after going back). */
	initialDate?: string;
	/** Where the times come from (default: the patient's session). */
	loadSlots?: SlotsLoader;
	/** Shown when no loaded day has free times (e.g. "Avísame si se libera un turno"). */
	emptyAction?: React.ReactNode;
	onPick: (date: string, time: string) => void;
}) {
	const { textGet, formatDate } = useText();
	const [page, setPage] = React.useState<Page>({ status: "loading" });
	const [loadingMore, setLoadingMore] = React.useState(false);
	const [selected, setSelected] = React.useState(initialDate);
	const targetRef = React.useRef(target);
	const loadRef = React.useRef(loadSlots);

	const fetchPage = React.useCallback(
		async (from: string) => {
			const range = slotRange(from, maxDate);
			if (!range) return { days: [] as SlotDay[], nextFrom: undefined };
			const res = await loadRef.current(
				typeId,
				targetRef.current,
				range.from,
				range.to,
			);
			if (!res.success) return undefined;
			const next = slotRange(addDays(range.to, 1), maxDate);
			return { days: res.data.days ?? [], nextFrom: next?.from };
		},
		[typeId, maxDate],
	);

	React.useEffect(() => {
		let active = true;
		fetchPage(minDate).then((result) => {
			if (!active) return;
			setPage(result ? { status: "ready", ...result } : { status: "error" });
		});
		return () => {
			active = false;
		};
	}, [fetchPage, minDate]);

	const loadMore = async () => {
		if (page.status !== "ready" || !page.nextFrom) return;
		setLoadingMore(true);
		const result = await fetchPage(page.nextFrom);
		setLoadingMore(false);
		if (!result) return;
		setPage({
			status: "ready",
			days: [...page.days, ...result.days],
			nextFrom: result.nextFrom,
		});
	};

	if (page.status === "loading") {
		return (
			<div className="space-y-4">
				<div className="flex gap-2">
					{[0, 1, 2, 3, 4].map((i) => (
						<Skeleton key={i} className="h-20 w-16 rounded-2xl" />
					))}
				</div>
				<Skeleton className="h-28 w-full rounded-2xl" />
			</div>
		);
	}
	if (page.status === "error") {
		return (
			<p className="text-sm text-muted-foreground">
				{textGet("portal.section.error")}
			</p>
		);
	}

	const days = page.days.filter((d) => d.times.length > 0);
	const day = days.find((d) => d.date === selected) ?? days[0];

	const moreButton = page.nextFrom && (
		<Button
			type="button"
			variant="secondary"
			className="rounded-full"
			disabled={loadingMore}
			onClick={loadMore}
		>
			{loadingMore ? <Spinner /> : <Plus />}
			{textGet("portal.booking.more_days")}
		</Button>
	);

	if (!day) {
		return (
			<div className="space-y-4 rounded-3xl border border-dashed px-5 py-6">
				<div className="flex items-start gap-3">
					<CalendarSearch className="mt-0.5 size-5 shrink-0 text-primary" />
					<p className="text-sm text-muted-foreground">
						{textGet(
							page.nextFrom
								? "portal.booking.no_slots_yet"
								: "portal.booking.no_slots",
						)}
					</p>
				</div>
				{moreButton}
				{emptyAction}
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<fieldset className="-mx-4 flex min-w-0 snap-x gap-2 overflow-x-auto border-0 px-4 pt-1 pb-3 lg:mx-0 lg:flex-wrap lg:overflow-visible lg:px-0">
				<legend className="sr-only">{textGet("portal.booking.days")}</legend>
				{days.map((d) => {
					const active = d.date === day.date;
					return (
						<button
							key={d.date}
							type="button"
							aria-pressed={active}
							aria-label={formatDate(d.date, "weekday-day-month")}
							onClick={() => setSelected(d.date)}
							className={cn(
								"flex w-16 shrink-0 snap-start flex-col items-center rounded-2xl border py-2.5 leading-none tabular-nums transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
								active
									? "border-primary bg-primary text-primary-foreground"
									: "bg-card hover:border-primary/40 hover:bg-secondary/60",
							)}
						>
							<span
								className={cn(
									"text-xs first-letter:uppercase",
									!active && "text-muted-foreground",
								)}
							>
								{formatDate(d.date, "weekday")}
							</span>
							<span className="mt-1.5 text-2xl font-semibold tracking-tight">
								{Number(d.date.slice(8, 10))}
							</span>
							<span
								className={cn(
									"mt-1 text-xs",
									!active && "text-muted-foreground",
								)}
							>
								{formatDate(d.date, "month")}
							</span>
						</button>
					);
				})}
				{page.nextFrom && (
					<button
						type="button"
						disabled={loadingMore}
						onClick={loadMore}
						className="flex w-16 shrink-0 snap-start flex-col items-center justify-center gap-1.5 rounded-2xl border border-dashed text-xs text-muted-foreground transition-colors outline-none hover:bg-secondary/60 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-60"
					>
						{loadingMore ? <Spinner /> : <Plus className="size-5" />}
						{textGet("portal.booking.more_days_short")}
					</button>
				)}
			</fieldset>

			<section aria-labelledby="slot-times">
				<h2
					id="slot-times"
					className="text-base font-semibold first-letter:uppercase"
				>
					{formatDate(day.date, "weekday-day-month")}
				</h2>
				<ul className="mt-3 grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-5">
					{day.times.map((time) => (
						<li key={time}>
							<button
								type="button"
								onClick={() => onPick(day.date, time)}
								className="h-11 w-full rounded-full border bg-card font-medium tabular-nums transition-colors outline-none hover:border-primary hover:bg-secondary focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 active:bg-primary active:text-primary-foreground"
							>
								{time}
							</button>
						</li>
					))}
				</ul>
			</section>
		</div>
	);
}
