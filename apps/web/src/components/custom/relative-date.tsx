import { useMessageStore } from "@pengi/shared";
import { formatRelativeTime } from "@/lib/notification-text";

interface RelativeDateProps {
	date: string;
	className?: string;
}

export function RelativeDate({
	date,
	className = "text-muted-foreground whitespace-nowrap text-sm",
}: RelativeDateProps) {
	const lang = useMessageStore((s) => s.lang);

	return (
		<span className={className} title={new Date(date).toLocaleString()}>
			{formatRelativeTime(date, lang)}
		</span>
	);
}
