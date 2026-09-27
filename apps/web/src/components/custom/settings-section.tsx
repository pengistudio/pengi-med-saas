import type React from "react";

/** A settings area: what it is on the left, its controls on the right. */
export function SettingsSection({
	title,
	description,
	children,
}: {
	title: React.ReactNode;
	description?: React.ReactNode;
	children: React.ReactNode;
}) {
	return (
		<section className="grid gap-4 border-t pt-8 md:grid-cols-[15rem_1fr] md:gap-10">
			<div>
				<h2 className="text-base font-semibold">{title}</h2>
				{description && (
					<div className="mt-1 space-y-2 text-sm text-muted-foreground">
						{description}
					</div>
				)}
			</div>
			<div className="grid min-w-0 gap-6 rounded-xl border bg-card p-5">
				{children}
			</div>
		</section>
	);
}
