import { useText } from "@pengi/shared";
import { Button, cn } from "@pengi/ui";
import {
	CalendarDays,
	FileText,
	FlaskConical,
	LogOut,
	type LucideIcon,
	Receipt,
} from "lucide-react";
import React from "react";
import type { Me } from "@/api/portal-service";
import { ClinicBrand } from "@/components/clinic-brand";
import { AppointmentsSection } from "@/components/sections/appointments-section";
import { DocumentsSection } from "@/components/sections/documents-section";
import { InvoicesSection } from "@/components/sections/invoices-section";
import { ResultsSection } from "@/components/sections/results-section";

const SECTIONS: {
	key: keyof Me["clinic"]["sections"];
	icon: LucideIcon;
	Component: () => React.ReactNode;
}[] = [
	{ key: "appointments", icon: CalendarDays, Component: AppointmentsSection },
	{ key: "results", icon: FlaskConical, Component: ResultsSection },
	{ key: "documents", icon: FileText, Component: DocumentsSection },
	{ key: "invoices", icon: Receipt, Component: InvoicesSection },
];

/** The signed-in home: the clinic, a greeting and one section at a time. */
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

	const tabs = (variant: "top" | "bottom") =>
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
						"flex items-center justify-center rounded-full font-medium transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
						variant === "top"
							? "gap-2 px-4 py-2 text-sm"
							: "flex-1 flex-col gap-1 py-1.5 text-[11px]",
						selected
							? "bg-secondary text-secondary-foreground"
							: "text-muted-foreground hover:text-foreground",
					)}
				>
					<Icon className={variant === "top" ? "size-4" : "size-5"} />
					{textGet(`portal.section.${key}`)}
				</button>
			);
		});

	return (
		<div className="mx-auto min-h-screen w-full max-w-xl px-4 pb-32 sm:pb-12">
			<header className="flex items-center justify-between gap-3 py-5">
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

			<h1 className="pt-2 pb-6 text-2xl font-semibold tracking-tight">
				{textGet("portal.home.greeting", { name: me.patient.first_name })}
			</h1>

			{!current ? (
				<p className="text-sm text-muted-foreground">
					{textGet("portal.home.no_sections")}
				</p>
			) : (
				<>
					{enabled.length > 1 && (
						<nav
							aria-label={textGet("portal.home.sections")}
							className="mb-6 hidden gap-1 sm:flex"
						>
							{tabs("top")}
						</nav>
					)}
					<current.Component />
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
		</div>
	);
}
