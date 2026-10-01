import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { CalendarPlus, FilePlus2, UserPlus } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { type DashboardStats, getDashboardStats } from "@/api/clinical-service";
import { PageHeader } from "@/components/custom/page-header";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { AttentionStrip } from "@/sections/dashboard/attention-strip";
import { OpenTasksCard } from "@/sections/dashboard/open-tasks-card";
import { RecentInvoicesCard } from "@/sections/dashboard/recent-invoices-card";
import { SubscriptionSummary } from "@/sections/dashboard/subscription-summary";
import { TodayAgenda } from "@/sections/dashboard/today-agenda";
import { WeekOverview } from "@/sections/dashboard/week-overview";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

function greetingKey(hour: number) {
	if (hour < 12) return "dashboard.greeting.morning";
	if (hour < 19) return "dashboard.greeting.afternoon";
	return "dashboard.greeting.evening";
}

const capitalize = (text: string) =>
	text.charAt(0).toUpperCase() + text.slice(1);

function DashboardSkeleton() {
	const block = "animate-pulse rounded-xl bg-muted/60";
	return (
		<div className="space-y-6" aria-busy>
			<div className="space-y-2">
				<div className={`${block} h-7 w-64`} />
				<div className={`${block} h-4 w-48`} />
			</div>
			<div className={`${block} h-14`} />
			<div className="grid gap-4 lg:grid-cols-3">
				<div className={`${block} h-80 lg:col-span-2`} />
				<div className={`${block} h-80`} />
			</div>
		</div>
	);
}

const Home = () => {
	const [stats, setStats] = React.useState<DashboardStats | null>(null);
	const { textGet, formatDate } = useText();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const environment = useSessionStore(selectEnvironment);

	React.useEffect(() => {
		getDashboardStats().then((res) => {
			if (res.success && res.data) setStats(res.data);
		});
	}, []);

	if (!stats) {
		return <DashboardSkeleton />;
	}

	// A feature missing from the plan info counts as enabled, as in the nav.
	const features = stats.subscription?.enabled_features ?? {};
	const clinicalOn = features.clinical !== false;
	const billingOn =
		features.billing !== false &&
		checkPermission([PERMISSIONS.BILLING.PERMISSION_READ_BILLING]);
	const kanbanOn =
		features.kanban !== false &&
		checkPermission([PERMISSIONS.KANBAN.PERMISSION_READ_KANBAN]);

	const canCreatePatient =
		clinicalOn &&
		checkPermission([PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_PATIENT]);
	const canCreateInvoice =
		billingOn &&
		checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]);

	const firstName = environment?.name?.trim().split(/\s+/)[0] ?? "";
	const greeting = textGet(greetingKey(new Date().getHours()), {
		name: firstName,
	}).replace(/,\s*$/, "");

	return (
		<div className="space-y-6">
			<PageHeader
				title={greeting}
				description={capitalize(formatDate(new Date(), "full"))}
				actions={
					<>
						{canCreatePatient && (
							<Button
								variant="outline"
								onClick={() => navigate("/clinical/create")}
							>
								<UserPlus className="mr-2 h-4 w-4" />
								{textGet("dashboard.actions.new_patient")}
							</Button>
						)}
						{canCreateInvoice && (
							<Button
								variant="outline"
								onClick={() => navigate("/billing/create")}
							>
								<FilePlus2 className="mr-2 h-4 w-4" />
								{textGet("dashboard.actions.new_invoice")}
							</Button>
						)}
						{clinicalOn && (
							<Button onClick={() => navigate("/clinical/appointments")}>
								<CalendarPlus className="mr-2 h-4 w-4" />
								{textGet("dashboard.upcoming.schedule_btn")}
							</Button>
						)}
					</>
				}
			/>

			<AttentionStrip
				criticalCount={clinicalOn ? stats.critical_patients : 0}
				criticalPatients={stats.critical_patient_list}
				draftsCount={clinicalOn ? stats.pending_drafts_count : 0}
				drafts={stats.pending_drafts}
				failedInvoices={billingOn ? stats.failed_invoices : undefined}
				subscription={stats.subscription}
			/>

			<div className="grid gap-4 lg:grid-cols-3">
				{clinicalOn && (
					<TodayAgenda
						className="lg:col-span-2"
						appointments={stats.today_agenda}
						canStartConsultation={checkPermission([
							PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
						])}
					/>
				)}
				<div className="space-y-4">
					{clinicalOn && <WeekOverview stats={stats} />}
					{stats.subscription && (
						<SubscriptionSummary subscription={stats.subscription} />
					)}
				</div>
			</div>

			{(kanbanOn || billingOn) && (
				<div className="grid gap-4 lg:grid-cols-2">
					{kanbanOn && (
						<OpenTasksCard
							tasks={stats.open_tasks}
							total={stats.open_tasks_count}
							canComplete={checkPermission([
								PERMISSIONS.KANBAN.PERMISSION_UPDATE_KANBAN,
							])}
							canCreate={checkPermission([
								PERMISSIONS.KANBAN.PERMISSION_CREATE_KANBAN,
							])}
						/>
					)}
					{billingOn && (
						<RecentInvoicesCard
							canRetry={checkPermission([
								PERMISSIONS.BILLING.PERMISSION_MANAGE_SRI_SETTINGS,
							])}
						/>
					)}
				</div>
			)}
		</div>
	);
};

export default Home;
