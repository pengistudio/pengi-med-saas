import { OTPField as OTPFieldPrimitive } from "@base-ui/react/otp-field";
import type * as React from "react";

import { cn } from "../lib/utils";

/**
 * A row of one-character slots for verification codes. Base UI's OTPField
 * moves focus between slots, splits a pasted code, accepts digits only by
 * default and sets autocomplete="one-time-code" on the first slot (SMS
 * autofill). `id` goes to the first slot, so a `<Label htmlFor>` points at it.
 */
function OtpInput({
	className,
	length = 6,
	"aria-invalid": ariaInvalid,
	...props
}: Omit<React.ComponentProps<typeof OTPFieldPrimitive.Root>, "length"> & {
	/** Number of slots. @default 6 */
	length?: number;
	"aria-invalid"?: boolean;
}) {
	return (
		<OTPFieldPrimitive.Root
			data-slot="otp-input"
			length={length}
			className={cn("flex items-center gap-2", className)}
			{...props}
		>
			{Array.from({ length }, (_, index) => (
				<OTPFieldPrimitive.Input
					// biome-ignore lint/suspicious/noArrayIndexKey: the slots are fixed by `length` and never reorder
					key={index}
					aria-invalid={ariaInvalid}
					className="dark:bg-input/30 border-input focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:aria-invalid:border-destructive/50 h-11 w-10 min-w-0 rounded-lg border bg-transparent text-center text-lg font-medium outline-none transition-colors focus-visible:ring-3 aria-invalid:ring-3 data-disabled:pointer-events-none data-disabled:opacity-50"
				/>
			))}
		</OTPFieldPrimitive.Root>
	);
}

export { OtpInput };
