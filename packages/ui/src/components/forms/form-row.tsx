import type * as React from "react";
import { cn } from "../../lib/utils";

// Each column needs ~13.5rem for a label, an input and its error message.
const COLUMNS = {
	2: "@md:grid-cols-2",
	3: "@2xl:grid-cols-3",
} as const;

interface FormRowProps extends React.ComponentProps<"div"> {
	/** Fields per row once there is room for them (default 2). */
	cols?: keyof typeof COLUMNS;
}

/**
 * Fields side by side when they fit, stacked when they don't. It measures the
 * space it is given (a container query), not the screen, so the same row
 * stacks on a phone, in a narrow dialog or in a side card.
 */
export function FormRow({ cols = 2, className, ...props }: FormRowProps) {
	return (
		<div data-slot="form-row" className="@container">
			<div className={cn("grid gap-4", COLUMNS[cols], className)} {...props} />
		</div>
	);
}
