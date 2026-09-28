import { useText } from "@pengi/shared";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	cn,
	Spinner,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@pengi/ui";
import { Ban, Plus } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type Announcement,
	announcements,
	cancelAnnouncement,
} from "@/api/announcement-service";
import { useResourceList } from "@/lib/resource";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const levelColors: Record<string, string> = {
	info: "bg-sky-500/10 text-sky-600",
	success: "bg-emerald-500/10 text-emerald-600",
	warning: "bg-amber-500/10 text-amber-600",
	critical: "bg-red-500/10 text-red-600",
};

const statusColors: Record<string, string> = {
	scheduled: "bg-sky-500/10 text-sky-600",
	sent: "bg-emerald-500/10 text-emerald-600",
	cancelled: "bg-zinc-500/10 text-zinc-500",
	failed: "bg-red-500/10 text-red-600",
};

function Pill({
	className,
	children,
}: React.PropsWithChildren<{ className?: string }>) {
	return (
		<span
			className={cn(
				"inline-flex items-center rounded-full px-2 py-1 text-xs font-medium",
				className ?? "bg-muted text-muted-foreground",
			)}
		>
			{children}
		</span>
	);
}

const AnnouncementList = () => {
	const { textGet, formatDateTime } = useText();
	const navigate = useNavigate();
	const { items, loading, refetch } = useResourceList(announcements);
	const [cancelling, setCancelling] = React.useState<Announcement | null>(null);
	const [saving, setSaving] = React.useState(false);

	const target = (a: Announcement) => {
		if (a.scope === "global")
			return textGet("backoffice.announcements.scope.global");
		const company = a.company_name || `#${a.company_id}`;
		if (a.scope === "company") return company;
		return `${a.user_name || `#${a.user_id}`} · ${company}`;
	};

	const confirmCancel = async () => {
		if (!cancelling) return;
		setSaving(true);
		const res = await cancelAnnouncement(cancelling.ID);
		setSaving(false);
		setCancelling(null);
		if (res.success) await refetch();
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet("backoffice.announcements.title")}
					</h1>
					<Button onClick={() => navigate("/announcements/create")}>
						<Plus className="h-4 w-4 mr-2" />
						{textGet("backoffice.announcements.create")}
					</Button>
				</div>
				<Card>
					<CardHeader>
						<CardTitle>
							{textGet("backoffice.announcements.list.title")}
						</CardTitle>
						<CardDescription>
							{textGet("backoffice.announcements.list.description")}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.common.loading")}
							</p>
						) : items.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet("backoffice.announcements.empty")}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>
											{textGet("backoffice.announcements.col.title")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.announcements.col.target")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.announcements.col.level")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.announcements.col.status")}
										</TableHead>
										<TableHead>
											{textGet("backoffice.announcements.col.date")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("backoffice.announcements.col.recipients")}
										</TableHead>
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{items.map((a) => (
										<TableRow key={a.ID}>
											<TableCell className="max-w-xs">
												<p className="font-medium truncate">{a.title}</p>
												<p className="text-xs text-muted-foreground truncate">
													{a.body}
												</p>
											</TableCell>
											<TableCell>{target(a)}</TableCell>
											<TableCell>
												<Pill className={levelColors[a.level]}>
													{textGet(`backoffice.announcements.level.${a.level}`)}
												</Pill>
											</TableCell>
											<TableCell>
												<Pill className={statusColors[a.status]}>
													{textGet(
														`backoffice.announcements.status.${a.status}`,
													)}
												</Pill>
											</TableCell>
											<TableCell className="text-muted-foreground">
												{formatDateTime(
													a.sent_at ?? a.scheduled_at ?? a.CreatedAt,
												) || "—"}
											</TableCell>
											<TableCell className="text-right">
												{a.recipient_count}
											</TableCell>
											<TableCell className="text-right">
												{a.status === "scheduled" && (
													<Button
														variant="ghost"
														size="icon"
														aria-label={textGet(
															"backoffice.announcements.cancel",
														)}
														onClick={() => setCancelling(a)}
													>
														<Ban className="h-4 w-4 text-destructive" />
													</Button>
												)}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>

			<AlertDialog
				open={cancelling !== null}
				onOpenChange={(open) => !open && setCancelling(null)}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{textGet("backoffice.announcements.cancel.title")}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{textGet("backoffice.announcements.cancel.description")}{" "}
							<strong>{cancelling?.title}</strong>
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel disabled={saving}>
							{textGet("backoffice.common.cancel")}
						</AlertDialogCancel>
						<AlertDialogAction onClick={confirmCancel} disabled={saving}>
							{saving && <Spinner />}
							{textGet("backoffice.announcements.cancel")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</DashboardLayout>
	);
};

export default AnnouncementList;
