import { useText } from "@pengi/shared";
import { Button, buttonVariants, cn } from "@pengi/ui";
import {
	CalendarDays,
	CalendarPlus,
	FileText,
	FlaskConical,
	LogOut,
	type LucideIcon,
	Receipt,
} from "lucide-react";
import React from "react";
import { Link } from "react-router";
import type { Me } from "@/api/portal-service";
import { ClinicBrand } from "@/components/clinic-brand";
import { AppointmentsSection } from "@/components/sections/appointments-section";
import { DocumentsSection } from "@/components/sections/documents-section";
import { InvoicesSection } from "@/components/sections/invoices-section";
import { ResultsSection } from "@/components/sections/results-section";

const SECTIONS: {
	key: keyof Me["clinic"]["sections"];
	icon: LucideIcon;
	Component: (props: { clinic: Me["clinic"] }) => React.ReactNode;
}[] = [
	{ key: "appointments", icon: CalendarDays, Component: AppointmentsSection },
	{ key: "results", icon: FlaskConical, Component: ResultsSection },
	{ key: "documents", icon: FileText, Component: DocumentsSection },
	{ key: "invoices", icon: Receipt, Component: InvoicesSection },
];

/**
 * The signed-in home: the clinic, a greeting and one section at a time. On a
 * phone, one column with a floating bottom bar; from `lg`, a sidebar with the
 * greeting, the sections and booking, next to the open section.
 */
export function HomeContent({
	me,
	onLogout,
}: {
	me: Me;
	onLogout: () => void;
}) {
	const { textGet } = useText();
	const enabled = SECTIONS.filter(({ key }) => me.clinic.sections[key]);
	const [active, setActive] = React.useState(enabled[0]?.key);
	const current = enabled.find(({ key }) => key === active) ?? enabled[0];
	const canBook = Boolean(
		me.clinic.booking?.enabled && me.clinic.sections.appointments,
	);

	const tabs = (variant: "top" | "bottom" | "side") =>
		enabled.map(({ key, icon: Icon }) => {
			const selected = key === current?.key;
			return (
				<button
					key={key}
					type="button"
					aria-current={selected ? "page" : undefined}
					onClick={() => {
						setActive(key);
						window.scrollTo({ top: 0 });
					}}
					className={cn(
						"flex items-center font-medium transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
						variant === "top" &&
							"justify-center gap-2 rounded-full px-4 py-2 text-sm",
						variant === "bottom" &&
							"flex-1 flex-col justify-center gap-1 rounded-full py-1.5 text-[11px]",
						variant === "side" &&
							"w-full gap-3 rounded-xl px-3 py-2.5 text-[15px]",
						selected
							? "bg-secondary text-secondary-foreground"
							: "text-muted-foreground hover:text-foreground",
						variant === "side" && !selected && "hover:bg-secondary/50",
					)}
				>
					<Icon
						className={cn(
							variant === "bottom" ? "size-5" : "size-4",
							variant === "side" && selected && "text-primary",
						)}
					/>
					{textGet(`portal.section.${key}`)}
				</button>
			);
		});

	return (
		<div className="mx-auto min-h-screen w-full max-w-xl px-4 pb-32 sm:pb-12 lg:max-w-6xl lg:px-10">
			<header className="flex items-center justify-between gap-3 py-5 lg:border-b lg:py-6">
				<ClinicBrand
					slug={me.clinic.slug}
					name={me.clinic.name}
					hasLogo={me.clinic.has_logo}
				/>
				<Button
					type="button"
					variant="ghost"
					size="sm"
					onClick={onLogout}
					className="shrink-0 text-muted-foreground"
				>
					<LogOut />
					{textGet("portal.logout")}
				</Button>
			</header>

			<div className="lg:grid lg:grid-cols-[15rem_minmax(0,1fr)] lg:gap-16 lg:pt-10">
				<aside className="lg:sticky lg:top-10 lg:self-start">
					<h1 className="pt-2 pb-6 text-2xl font-semibold tracking-tight lg:pt-0 lg:pb-8 lg:text-3xl">
						{textGet("portal.home.greeting", { name: me.patient.first_name })}
					</h1>
					{enabled.length > 1 && (
						<nav
							aria-label={textGet("portal.home.sections")}
							className="hidden flex-col gap-1 lg:flex"
						>
							{tabs("side")}
						</nav>
					)}
					{canBook && (
						<Link
							to={`/${me.clinic.slug}/citas/nueva`}
							className={cn(
								buttonVariants(),
								"mt-8 hidden h-12 w-full gap-2 rounded-full text-base lg:flex",
							)}
						>
							<CalendarPlus className="size-5" />
							{textGet("portal.booking.new")}
						</Link>
					)}
				</aside>

				<main className="min-w-0 lg:max-w-3xl">
					{!current ? (
						<p className="text-sm text-muted-foreground">
							{textGet("portal.home.no_sections")}
						</p>
					) : (
						<>
							{enabled.length > 1 && (
								<nav
									aria-label={textGet("portal.home.sections")}
									className="mb-6 hidden gap-1 sm:flex lg:hidden"
								>
									{tabs("top")}
								</nav>
							)}
							<h2 className="hidden pb-6 text-xl font-semibold tracking-tight lg:block">
								{textGet(`portal.section.${current.key}`)}
							</h2>
							<current.Component clinic={me.clinic} />
							{enabled.length > 1 && (
								<nav
									aria-label={textGet("portal.home.sections")}
									className="fixed inset-x-3 bottom-[max(0.75rem,env(safe-area-inset-bottom))] z-10 flex gap-1 rounded-full border bg-card/90 p-1.5 shadow-lg backdrop-blur-md sm:hidden"
								>
									{tabs("bottom")}
								</nav>
							)}
						</>
					)}
				</main>
			</div>
		</div>
	);
}
