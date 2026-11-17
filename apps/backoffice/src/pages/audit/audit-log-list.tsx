import { useText } from "@pengi/shared";
import {
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
import React from "react";
import { type BackofficeAuditLog, getAuditLogs } from "@/api/audit-service";

const statusColor = (status: number) =>
	status < 400
		? "bg-emerald-500/10 text-emerald-600"
		: "bg-red-500/10 text-red-600";

const AuditLogList = () => {
	const { textGet, formatDateTime } = useText();
	const [items, setItems] = React.useState<BackofficeAuditLog[]>([]);
	const [hasMore, setHasMore] = React.useState(false);
	const [loading, setLoading] = React.useState(true);

	const load = React.useCallback(async (beforeId?: number) => {
		setLoading(true);
		const res = await getAuditLogs(beforeId);
		setLoading(false);
		if (!res.success || !res.data) return;
		const page = res.data;
		setItems((current) =>
			beforeId ? [...current, ...page.items] : page.items,
		);
		setHasMore(page.has_more);
	}, []);

	React.useEffect(() => {
		load();
	}, [load]);

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-bold tracking-tight">
				{textGet("backoffice.audit.title")}
			</h1>
			<Card>
				<CardHeader>
					<CardTitle>{textGet("backoffice.audit.list.title")}</CardTitle>
					<CardDescription>
						{textGet("backoffice.audit.list.description")}
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					{items.length === 0 ? (
						<p
							className={cn(
								"text-sm text-muted-foreground py-8 text-center",
								loading && "animate-pulse",
							)}
						>
							{textGet(
								loading
									? "backoffice.common.loading"
									: "backoffice.audit.empty",
							)}
						</p>
					) : (
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>{textGet("backoffice.audit.col.date")}</TableHead>
									<TableHead>{textGet("backoffice.audit.col.admin")}</TableHead>
									<TableHead>
										{textGet("backoffice.audit.col.action")}
									</TableHead>
									<TableHead>
										{textGet("backoffice.audit.col.status")}
									</TableHead>
									<TableHead>{textGet("backoffice.audit.col.ip")}</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								{items.map((entry) => (
									<TableRow key={entry.id}>
										<TableCell className="text-muted-foreground whitespace-nowrap">
											{formatDateTime(entry.created_at)}
										</TableCell>
										<TableCell className="font-medium">
											{entry.user_name || `#${entry.backoffice_user_id}`}
										</TableCell>
										<TableCell
											className="font-mono text-xs"
											title={entry.route}
										>
											{entry.method} {entry.path}
										</TableCell>
										<TableCell>
											<span
												className={cn(
													"inline-flex items-center rounded-full px-2 py-1 text-xs font-medium",
													statusColor(entry.status),
												)}
											>
												{entry.status}
											</span>
										</TableCell>
										<TableCell
											className="text-muted-foreground"
											title={entry.user_agent}
										>
											{entry.ip}
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					)}
					{hasMore && (
						<div className="flex justify-center">
							<Button
								variant="outline"
								disabled={loading}
								onClick={() => load(items[items.length - 1]?.id)}
							>
								{loading && <Spinner />}
								{textGet("backoffice.audit.load_more")}
							</Button>
						</div>
					)}
				</CardContent>
			</Card>
		</div>
	);
};

export default AuditLogList;
