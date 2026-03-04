import React from "react";
import { END_HOUR, HOUR_HEIGHT, START_HOUR } from "./appointment-utils";

export function CurrentTimeLine() {
	const [now, setNow] = React.useState(new Date());

	React.useEffect(() => {
		const timer = setInterval(() => setNow(new Date()), 60_000);
		return () => clearInterval(timer);
	}, []);

	const minutes = now.getHours() * 60 + now.getMinutes() - START_HOUR * 60;
	if (minutes < 0 || minutes > (END_HOUR - START_HOUR) * 60) return null;

	const top = (minutes / 60) * HOUR_HEIGHT;

	return (
		<div
			className="absolute left-0 right-0 z-20 pointer-events-none"
			style={{ top: `${top}px` }}
		>
			<div className="relative">
				<div className="absolute -left-[5px] -top-[5px] w-[10px] h-[10px] rounded-full bg-red-500" />
				<div className="h-[2px] bg-red-500 w-full" />
			</div>
		</div>
	);
}
