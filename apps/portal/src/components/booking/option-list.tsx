import { ChevronRight } from "lucide-react";
import type React from "react";

/** A column of large tappable choices; picking one moves the flow on. */
export function OptionList({ children }: { children: React.ReactNode }) {
	return <ul className="space-y-2">{children}</ul>;
}

export function OptionButton({
	leading,
	title,
	subtitle,
	onClick,
}: {
	leading: React.ReactNode;
	title: string;
	subtitle?: string;
	onClick: () => void;
}) {
	return (
		<li>
			<button
				type="button"
				onClick={onClick}
				className="flex w-full items-center gap-4 rounded-2xl border bg-card px-4 py-3.5 text-left transition-colors outline-none hover:border-primary/40 hover:bg-secondary/60 focus-visible:ring-2 focus-visible:ring-ring active:bg-secondary"
			>
				<span
					aria-hidden="true"
					className="flex size-11 shrink-0 items-center justify-center rounded-full bg-secondary text-sm font-semibold text-secondary-foreground"
				>
					{leading}
				</span>
				<span className="min-w-0 flex-1">
					<span className="block font-medium">{title}</span>
					{subtitle && (
						<span className="block text-sm text-muted-foreground">
							{subtitle}
						</span>
					)}
				</span>
				<ChevronRight
					aria-hidden="true"
					className="size-5 shrink-0 text-muted-foreground"
				/>
			</button>
		</li>
	);
}
