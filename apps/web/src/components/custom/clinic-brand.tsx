import { useEffect } from "react";
import GentooPenguin from "@/assets/gentoo.png";
import { selectEnvironment, useSessionStore } from "@/store/session-store";
import { useTenantLogoStore } from "@/store/tenant-logo-store";

/**
 * Top of the sidebar: the clinic's logo when it uploaded one, else the Gentoo
 * penguin on a brand tile. A wide logo (a wordmark) already carries the name,
 * so it takes the whole header alone; a square one sits in a white tile
 * beside the name.
 */
export function ClinicBrand() {
	const environment = useSessionStore(selectEnvironment);
	const slug = environment?.tenant_slug;
	const name = environment?.trade_name;
	const { url, wide, load } = useTenantLogoStore();

	useEffect(() => {
		if (slug) load(slug);
	}, [slug, load]);

	if (url && wide) {
		return (
			<img
				src={url}
				alt={name}
				title={name}
				className="h-10 min-w-0 max-w-full object-contain object-left"
			/>
		);
	}

	return (
		<>
			{url === undefined ? (
				// Keeps the slot while the logo loads, so the penguin doesn't flash first.
				<span className="h-8 w-8 shrink-0 rounded-lg bg-sidebar-accent" />
			) : url ? (
				<span className="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white p-0.5 ring-1 ring-sidebar-border">
					<img src={url} alt="" className="h-full w-full object-contain" />
				</span>
			) : (
				<span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary">
					{/* The teal PNG turned white, like the icon tiles elsewhere. */}
					<img
						src={GentooPenguin}
						alt=""
						className="h-6 w-6 brightness-0 invert"
					/>
				</span>
			)}
			<span
				title={name}
				className="line-clamp-2 min-w-0 text-base font-semibold leading-tight text-sidebar-foreground"
			>
				{name}
			</span>
		</>
	);
}
