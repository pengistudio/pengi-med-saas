/**
 * Classes that turn a centered, `fixed` popup (Dialog, AlertDialog) into a
 * sheet from the bottom edge on a phone (below `sm`), within thumb reach.
 * They only add `max-sm:` rules, so a caller's `sm:max-w-*` keeps working;
 * the width is `!` because popups set their own `max-w-*` per size.
 */
export const phoneSheet =
	"max-sm:top-auto max-sm:bottom-0 max-sm:left-0 max-sm:max-h-[90dvh] max-sm:max-w-none! max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-b-none max-sm:rounded-t-2xl max-sm:duration-200 max-sm:data-open:slide-in-from-bottom max-sm:data-closed:slide-out-to-bottom max-sm:data-open:zoom-in-100 max-sm:data-closed:zoom-out-100 motion-reduce:animate-none";

/** The footer of a phone sheet runs to the bottom edge, so it loses its rounding. */
export const phoneSheetFooter = "max-sm:rounded-b-none";
