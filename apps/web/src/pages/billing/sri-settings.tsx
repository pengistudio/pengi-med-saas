import { useText } from "@pengi/shared";
import { Text } from "@pengi/ui";
import { differenceInCalendarDays } from "date-fns";
import { AlertTriangle, CheckCircle2, Loader2, XCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { getSriStatus, type SriStatus } from "@/api/tenant-service";
import { PageHeader } from "@/components/custom/page-header";
import { SettingsSection } from "@/components/custom/settings-section";
import { cn } from "@/lib/utils";
import { LogoUploadForm } from "@/sections/forms/billing/logo-upload-form";
import { SriInfoForm } from "@/sections/forms/billing/sri-info-form";
import { SriSignatureForm } from "@/sections/forms/billing/sri-signature-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

/** Warn about the signature this many days before it expires. */
const EXPIRY_WARNING_DAYS = 30;

/**
 * The state of the tenant's SRI signature. The server reports an expired P12 as
 * not configured but keeps its date, so the date is read first.
 */
function SignatureStatus({ status }: { status: SriStatus | null }) {
	const { textGet, formatDate } = useText();

	let tone: "ok" | "warn" | "error" = "error";
	let title = textGet("billing.sri.status.unconfigured");
	let detail = textGet("billing.sri.status.unconfigured_desc");

	if (status?.expiration_date) {
		const expires = new Date(status.expiration_date);
		const days = differenceInCalendarDays(expires, new Date());
		const date = formatDate(expires, "long");
		if (days < 0) {
			title = textGet("billing.sri.status.expired");
			detail = textGet("billing.sri.status.expired_desc", { date });
		} else {
			tone = days <= EXPIRY_WARNING_DAYS ? "warn" : "ok";
			title = textGet(
				tone === "warn"
					? "billing.sri.status.expiring"
					: "billing.sri.status.configured",
			);
			detail = textGet("billing.sri.status.expires_on", { date, days });
		}
	} else if (status?.is_configured) {
		tone = "ok";
		title = textGet("billing.sri.status.configured");
		detail = "";
	}

	const Icon =
		tone === "ok" ? CheckCircle2 : tone === "warn" ? AlertTriangle : XCircle;

	return (
		<div
			className={cn(
				"flex items-start gap-3 rounded-xl border px-4 py-3",
				tone === "ok" && "border-primary/20 bg-primary/5",
				tone === "warn" && "border-amber-500/30 bg-amber-500/10",
				tone === "error" && "border-destructive/30 bg-destructive/5",
			)}
		>
			<Icon
				className={cn(
					"mt-0.5 size-5 shrink-0",
					tone === "ok" && "text-primary",
					tone === "warn" && "text-amber-600 dark:text-amber-400",
					tone === "error" && "text-destructive",
				)}
			/>
			<div>
				<p className="text-sm font-medium">{title}</p>
				{detail && <p className="text-sm text-muted-foreground">{detail}</p>}
			</div>
		</div>
	);
}

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
			<div className="grid max-w-5xl gap-8">
				<div className="grid gap-4">
					<PageHeader title={<Text uuid="billing.sri.settings.title" />} />
					{loading && !status ? (
						<div className="flex h-12 items-center">
							<Loader2 className="size-5 animate-spin text-muted-foreground" />
						</div>
					) : (
						<SignatureStatus status={status} />
					)}
				</div>

				<SettingsSection
					title={<Text uuid="billing.sri.signature.title" />}
					description={
						<>
							<p>
								<Text uuid="billing.sri.signature.description" />
							</p>
							<p>
								<Text uuid="billing.sri.info.p1" />
							</p>
						</>
					}
				>
					<SriSignatureForm onSuccess={fetchStatus} />
				</SettingsSection>

				{status && (
					<SettingsSection
						title={<Text uuid="billing.sri.company_info.title" />}
						description={<Text uuid="billing.sri.company_info.description" />}
					>
						<SriInfoForm initialData={status} onSuccess={fetchStatus} />
					</SettingsSection>
				)}

				{status && (
					<SettingsSection
						title={<Text uuid="billing.sri.logo.title" />}
						description={<Text uuid="billing.sri.logo.description" />}
					>
						<LogoUploadForm hasLogo={status.has_logo} onSuccess={fetchStatus} />
					</SettingsSection>
				)}
			</div>
		</DashboardLayout>
	);
};

export default SriSettingsPage;
