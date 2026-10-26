import { env } from "@/lib/env";

const initials = (name: string) =>
	name
		.split(/\s+/)
		.filter(Boolean)
		.slice(0, 2)
		.map((word) => word[0]?.toUpperCase())
		.join("");

/** The clinic's logo, or its initials in the brand color when it has none. */
export function ClinicBrand({
	slug,
	name,
	hasLogo,
}: {
	slug: string;
	name: string;
	hasLogo: boolean;
}) {
	return (
		<div className="flex min-w-0 items-center gap-3">
			{hasLogo ? (
				<img
					src={`${env.VITE_API_URL}/portal/clinics/${encodeURIComponent(slug)}/logo`}
					alt=""
					className="size-10 shrink-0 rounded-xl object-contain"
				/>
			) : (
				<span
					aria-hidden="true"
					className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-primary text-sm font-semibold text-primary-foreground"
				>
					{initials(name)}
				</span>
			)}
			<span className="truncate font-semibold leading-tight">{name}</span>
		</div>
	);
}
