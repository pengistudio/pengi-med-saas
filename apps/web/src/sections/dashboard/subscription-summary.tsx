import { useText } from "@pengi/shared";
import { Card, CardContent } from "@pengi/ui";
import { CreditCard } from "lucide-react";
import { useNavigate } from "react-router";
import type { SubscriptionInfo } from "@/api/clinical-service";
import { cn, dateParser } from "@/lib/utils";
import { SUBSCRIPTION_WARNING_DAYS } from "./attention-strip";

/** One line about the plan; it only takes color when renewal is close. */
export function SubscriptionSummary({
	subscription,
}: {
	subscription: SubscriptionInfo;
}) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const expired = subscription.days_left <= 0;
	const status = expired ? "expired" : subscription.status;
	const urgent = expired || subscription.days_left <= 7;
	const warning =
		!urgent && subscription.days_left <= SUBSCRIPTION_WARNING_DAYS;

	return (
		<Card
			className={cn(
				"py-0",
				urgent && "border-destructive/40",
				warning && "border-amber-500/40",
			)}
		>
			<CardContent className="flex items-center gap-3 px-4 py-3">
				<CreditCard
					className={cn(
						"h-4 w-4 shrink-0",
						urgent
							? "text-destructive"
							: warning
								? "text-amber-600"
								: "text-muted-foreground",
					)}
				/>
				<div className="min-w-0 flex-1">
					<p className="text-sm">
						<span className="font-semibold">{subscription.plan_name}</span>{" "}
						<span className="text-muted-foreground">
							{textGet(`subscription.status.${status}`)}
						</span>
					</p>
					<p className="text-xs text-muted-foreground">
						{textGet("dashboard.subscription.expires")}{" "}
						{dateParser(subscription.expires_at, { dateStyle: "long" })}
					</p>
				</div>
				<button
					type="button"
					onClick={() => navigate("/subscription")}
					className="shrink-0 rounded text-sm font-medium text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
				>
					{textGet("dashboard.subscription.manage")}
				</button>
			</CardContent>
		</Card>
	);
}
