import { CheckCircle2, Info, Loader2, XCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { getSriStatus, type SriStatus } from "@/api/tenant-service";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Text } from "@/components/ui/text";
import { SriSignatureForm } from "@/sections/forms/billing/sri-signature-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const SriSettingsPage = () => {
	const [status, setStatus] = useState<SriStatus | null>(null);
	const [loading, setLoading] = useState(true);

	const fetchStatus = useCallback(async () => {
		setLoading(true);
		const res = await getSriStatus();
		if (res.success) {
			setStatus(res.data);
		}
		setLoading(false);
	}, []);

	useEffect(() => {
		fetchStatus();
	}, [fetchStatus]);
	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex items-center">
					<h1 className="text-2xl font-semibold tracking-tight">
						<Text uuid="billing.sri.settings.title" />
					</h1>
				</div>
				<div className="grid gap-6 md:grid-cols-2">
					<div className="flex flex-col gap-6">
						{loading ? (
							<Card className="flex items-center justify-center p-8">
								<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
							</Card>
						) : status?.is_configured ? (
							<Card className="bg-primary/5 border-primary/20 gap-0">
								<CardHeader className="pb-3">
									<CardTitle className="flex items-center gap-2 text-primary text-base">
										<CheckCircle2 className="h-5 w-5" />
										<Text uuid="billing.sri.status.configured" />
									</CardTitle>
								</CardHeader>
								<CardContent>
									{status.expiration_date && (
										<p className="text-sm text-foreground/80">
											<Text uuid="billing.sri.status.valid_until" />{" "}
											<span className="font-semibold">
												{new Date(status.expiration_date).toLocaleDateString()}
											</span>
										</p>
									)}
								</CardContent>
							</Card>
						) : (
							<Card className="bg-destructive/5 border-destructive/20 gap-0">
								<CardHeader className="pb-3">
									<CardTitle className="flex items-center gap-2 text-destructive text-base">
										<XCircle className="h-5 w-5" />
										<Text uuid="billing.sri.status.unconfigured" />
									</CardTitle>
								</CardHeader>
								<CardContent>
									<p className="text-sm text-foreground/80">
										<Text uuid="billing.sri.status.unconfigured_desc" />
									</p>
								</CardContent>
							</Card>
						)}

						<Card>
							<CardHeader>
								<CardTitle>
									<Text uuid="billing.sri.signature.title" />
								</CardTitle>
								<CardDescription>
									<Text uuid="billing.sri.signature.description" />
								</CardDescription>
							</CardHeader>
							<CardContent>
								<SriSignatureForm onSuccess={fetchStatus} />
							</CardContent>
						</Card>
					</div>

					{/* Add more SRI related settings here later (e.g environment, sequence start) */}
					<Card className="bg-primary/5 border-primary/20 h-fit">
						<CardHeader>
							<CardTitle className="flex items-center gap-2 text-primary">
								<Info className="h-5 w-5" />
								<Text uuid="billing.sri.info.title" />
							</CardTitle>
							<CardDescription className="text-primary/80">
								<Text uuid="billing.sri.info.description" />
							</CardDescription>
						</CardHeader>
						<CardContent className="text-sm text-foreground/80 space-y-4">
							<div className="flex items-start gap-2">
								<div className="mt-1.5 shrink-0 h-1.5 w-1.5 rounded-full bg-primary/70" />
								<p>
									<Text uuid="billing.sri.info.p1" />
								</p>
							</div>
							<div className="flex items-start gap-2">
								<div className="mt-1.5 shrink-0 h-1.5 w-1.5 rounded-full bg-primary/70" />
								<p>
									<Text uuid="billing.sri.info.p2" />
								</p>
							</div>
						</CardContent>
					</Card>
				</div>
			</main>
		</DashboardLayout>
	);
};

export default SriSettingsPage;
