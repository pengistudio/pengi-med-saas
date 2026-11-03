import { useText } from "@pengi/shared";
import { Sparkles } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * The one visual signature of AI in the app: the same tile + left accent the
 * section cards use (see "Próxima Cita"), in a single color so AI content is
 * recognizable wherever it shows up.
 */
export const AI_CARD_CLASS =
	"rounded-lg border border-l-4 border-l-violet-500 bg-card text-card-foreground shadow-xs";

export function AIMark({ className }: { className?: string }) {
	return (
		<div
			aria-hidden="true"
			className={cn(
				"flex size-8 shrink-0 items-center justify-center rounded-lg bg-violet-500 text-white",
				className,
			)}
		>
			<Sparkles className="size-4" />
		</div>
	);
}

/** Small tag for text the model wrote and the doctor has not edited. */
export function AIDraftTag({ className }: { className?: string }) {
	const { textGet } = useText();
	return (
		<span
			className={cn(
				"inline-flex items-center gap-1 rounded-md bg-violet-100 px-1.5 py-0.5 text-xs font-medium text-violet-700 dark:bg-violet-950 dark:text-violet-300",
				className,
			)}
		>
			<Sparkles className="size-3" aria-hidden="true" />
			{textGet("ai.summary.draft_badge")}
		</span>
	);
}
