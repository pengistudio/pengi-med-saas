import {
	AlertTriangle,
	CheckCircle,
	CreditCard,
	Loader2,
	XCircle,
} from "lucide-react";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import {
	confirmPayment,
	getMySubscription,
	getSubscriptionPayments,
	initiatePayment,
	type SubscriptionDetail,
	type SubscriptionPaymentRecord,
} from "@/api/subscription-service";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";

// ─── Helpers ─────────────────────────────────────────────────────────────────

function daysColor(days: number) {
	if (days <= 7) return "text-red-500";
	if (days <= 15) return "text-amber-500";
	return "text-emerald-500";
}

function StatusBadge({ status }: { status: string }) {
	const { textGet } = useText();
	const map: Record<
		string,
		{
			label: string;
			variant: "default" | "secondary" | "destructive" | "outline";
		}
	> = {
		active: {
			label: textGet("subscription.status.active"),
			variant: "default",
		},
		expired: {
			label: textGet("subscription.status.expired"),
			variant: "destructive",
		},
		inactive: {
			label: textGet("subscription.status.inactive"),
			variant: "secondary",
		},
		cancelled: {
			label: textGet("subscription.status.cancelled"),
			variant: "outline",
		},
	};
	const cfg = map[status] ?? { label: status, variant: "secondary" as const };
	return <Badge variant={cfg.variant}>{cfg.label}</Badge>;
}

function PaymentStatusBadge({ status }: { status: string }) {
	const { textGet } = useText();
	const icons: Record<string, React.ReactNode> = {
		pending: <Loader2 className="h-3 w-3 animate-spin" />,
		paid: <CheckCircle className="h-3 w-3" />,
		rejected: <XCircle className="h-3 w-3" />,
		cancelled: <XCircle className="h-3 w-3" />,
		expired: <AlertTriangle className="h-3 w-3" />,
	};
	const variants: Record<
		string,
		"default" | "secondary" | "destructive" | "outline"
	> = {
		paid: "default",
		pending: "secondary",
		rejected: "destructive",
		cancelled: "outline",
		expired: "outline",
	};
	return (
		<Badge variant={variants[status] ?? "secondary"} className="gap-1">
			{icons[status]}
			{textGet(`subscription.payment.status.${status}`)}
		</Badge>
	);
}

// ─── Page ─────────────────────────────────────────────────────────────────────

const MySubscriptionPage = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const paymentSuccess = searchParams.get("status") === "success";

	const [sub, setSub] = React.useState<SubscriptionDetail | null>(null);
	const [payments, setPayments] = React.useState<SubscriptionPaymentRecord[]>(
		[],
	);
	const [loading, setLoading] = React.useState(true);
	const [paying, setPaying] = React.useState(false);

	const fetchData = React.useCallback(async () => {
		const [subRes, paymentsRes] = await Promise.all([
			getMySubscription(),
			getSubscriptionPayments(),
		]);
		if (subRes.success && subRes.data) setSub(subRes.data);
		if (paymentsRes.success && paymentsRes.data) setPayments(paymentsRes.data);
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetchData();
	}, [fetchData]);

	React.useEffect(() => {
		if (!paymentSuccess) return;
		navigate("/subscription", { replace: true });
		// Confirm payment immediately (success redirect = payment completed)
		// then re-fetch to show updated subscription data
		confirmPayment().then(() => fetchData());
	}, [paymentSuccess, navigate, fetchData]);

	const handlePay = async () => {
		setPaying(true);
		// Open window synchronously (before await) to avoid popup blocker
		const win = window.open("", "_blank");
		const res = await initiatePayment();
		setPaying(false);
		if (res.success && res.data.checkout_url) {
			if (win) win.location.href = res.data.checkout_url;
			else window.location.href = res.data.checkout_url;
		} else {
			win?.close();
		}
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				{paymentSuccess && (
					<div className="flex items-center gap-3 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-3">
						<CheckCircle className="h-5 w-5 text-emerald-500 shrink-0" />
						<p className="text-sm font-medium text-emerald-700 dark:text-emerald-400">
							{textGet("subscription.payment.success_banner")}
						</p>
					</div>
				)}
				<div>
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("subscription.page.title")}
					</h1>
					<p className="text-muted-foreground text-sm mt-1">
						{textGet("subscription.page.description")}
					</p>
				</div>

				{/* Subscription overview — full width single card */}
				<Card className="border-t-2 border-t-primary">
					<CardContent className="pt-6">
						{loading ? (
							<div className="flex gap-8">
								{[1, 2, 3, 4].map((i) => (
									<div
										key={i}
										className="h-12 w-32 bg-muted animate-pulse rounded"
									/>
								))}
							</div>
						) : sub ? (
							<div className="grid grid-cols-2 gap-6 sm:grid-cols-4 sm:divide-x sm:gap-0">
								{/* Plan */}
								<div className="sm:pr-6">
									<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
										{textGet("subscription.card.plan")}
									</p>
									<div className="flex items-center gap-2">
										<p className="text-2xl font-bold">{sub.plan_name}</p>
										<StatusBadge status={sub.status} />
									</div>
								</div>
								{/* Amount */}
								<div className="sm:px-6">
									<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
										{textGet("subscription.card.month")}
									</p>
									<p className="text-2xl font-bold">${sub.amount.toFixed(2)}</p>
									<p className="text-xs text-muted-foreground">USD / mes</p>
								</div>
								{/* Expiry */}
								<div className="sm:px-6">
									<p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1">
										{textGet("subscription.card.expiry")}
									</p>
									<p className="text-2xl font-bold">
										{new Date(sub.expires_at).toLocaleDateString()}
									</p>
									<p
										className={cn(
											"text-xs font-medium",
											daysColor(sub.days_left),
										)}
									>
										{sub.days_left} {textGet("subscription.card.days_left")}
									</p>
								</div>
								{/* Action */}
								<div className="sm:pl-6 flex items-center">
									<Button
										className="w-full sm:w-auto"
										onClick={handlePay}
										disabled={paying}
									>
										<CreditCard className="h-4 w-4 mr-2" />
										{paying
											? textGet("subscription.card.paying")
											: textGet("subscription.card.pay_btn")}
									</Button>
								</div>
							</div>
						) : (
							<p className="text-sm text-muted-foreground">—</p>
						)}
					</CardContent>
				</Card>

				{/* Payment History */}
				<Card>
					<CardHeader>
						<CardTitle>{textGet("subscription.payments.title")}</CardTitle>
						<CardDescription>
							{textGet("subscription.payments.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<div className="space-y-2">
								{[1, 2, 3].map((i) => (
									<div
										key={i}
										className="h-10 w-full bg-muted animate-pulse rounded"
									/>
								))}
							</div>
						) : payments.length === 0 ? (
							<p className="text-sm text-muted-foreground text-center py-8">
								{textGet("subscription.payments.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("subscription.payments.col.date")}
										</TableHead>
										<TableHead>
											{textGet("subscription.payments.col.amount")}
										</TableHead>
										<TableHead>
											{textGet("subscription.payments.col.status")}
										</TableHead>
										<TableHead>
											{textGet("subscription.payments.col.order")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{payments.map((p) => (
										<TableRow key={p.ID}>
											<TableCell className="text-sm">
												{new Date(p.CreatedAt).toLocaleDateString()}
											</TableCell>
											<TableCell className="text-sm font-medium">
												${p.amount.toFixed(2)}
											</TableCell>
											<TableCell>
												<PaymentStatusBadge status={p.status} />
											</TableCell>
											<TableCell className="text-xs text-muted-foreground font-mono">
												{p.order_id.slice(0, 8)}…
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>
		</DashboardLayout>
	);
};

export default MySubscriptionPage;
