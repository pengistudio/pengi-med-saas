import { useText } from "@pengi/shared";

interface RelativeDateProps {
	date: string;
	className?: string;
}

export function RelativeDate({
	date,
	className = "text-muted-foreground whitespace-nowrap text-sm",
}: RelativeDateProps) {
	const { formatDateTime, formatRelative } = useText();

	return (
		<span className={className} title={formatDateTime(date)}>
			{formatRelative(date)}
		</span>
	);
}
