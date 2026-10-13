import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { AlertTriangle, Stethoscope } from "lucide-react";
import { useNavigate } from "react-router";
import { markDoctorsReviewed } from "@/api/doctors-service";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { useDoctorStatus, useDoctorStore } from "@/store/doctors-store";

/**
 * "Register at least one doctor": shown in clinical screens while the tenant
 * has no active doctor (records and signable documents need one).
 */
export function RegisterDoctorNotice({ className }: { className?: string }) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const canManage = checkPermission([
		PERMISSIONS.DOCTORS.PERMISSION_MANAGE_DOCTORS,
	]);
	// Only those who attend or manage doctors; the front desk isn't blocked.
	const relevant =
		canManage ||
		checkPermission([
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
		]);
	const status = useDoctorStatus(relevant);

	if (!relevant || !status || status.active_doctors > 0) return null;

	return (
		<div
			role="status"
			className={cn(
				"flex flex-wrap items-center justify-between gap-x-3 gap-y-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3",
				className,
			)}
		>
			<div className="flex items-center gap-2">
				<Stethoscope className="h-4 w-4 shrink-0 text-amber-600" />
				<p className="text-sm text-amber-800 dark:text-amber-300">
					<span className="font-medium">
						{textGet("doctors.notice.register.title")}
					</span>{" "}
					{textGet(
						canManage
							? "doctors.notice.register.description"
							: "doctors.notice.register.ask_admin",
					)}
				</p>
			</div>
			{canManage && (
				<Button
					size="sm"
					variant="outline"
					onClick={() => navigate("/doctors")}
				>
					{textGet("doctors.notice.register.cta")}
				</Button>
			)}
		</div>
	);
}

/**
 * One-time "Review your doctors' data" banner for admins after the data
 * migration created profiles from existing users.
 */
export function DoctorReviewBanner() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const canManage = checkPermission([
		PERMISSIONS.DOCTORS.PERMISSION_MANAGE_DOCTORS,
	]);
	const status = useDoctorStatus(canManage);
	const refresh = useDoctorStore((s) => s.refresh);

	if (!canManage || !status?.needs_review) return null;

	async function handleDismiss() {
		const res = await markDoctorsReviewed();
		if (res.success) refresh();
	}

	return (
		<div className="flex shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-2 border-b border-sky-500/30 bg-sky-500/10 px-4 py-2.5">
			<div className="flex items-center gap-2">
				<AlertTriangle className="h-4 w-4 shrink-0 text-sky-600" />
				<p className="text-sm font-medium text-sky-800 dark:text-sky-300">
					{textGet("doctors.notice.review.description")}
				</p>
			</div>
			<div className="flex shrink-0 gap-2">
				<Button size="sm" variant="ghost" onClick={handleDismiss}>
					{textGet("doctors.notice.review.dismiss")}
				</Button>
				<Button
					size="sm"
					variant="outline"
					onClick={() => navigate("/doctors")}
				>
					{textGet("doctors.notice.review.cta")}
				</Button>
			</div>
		</div>
	);
}
