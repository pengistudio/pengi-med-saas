import { useText } from "@pengi/shared";
import {
	Button,
	Input,
	Label,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import { X } from "lucide-react";
import React from "react";
import { useSearchParams } from "react-router";
import {
	AUDIT_ACTIONS,
	AUDIT_ENTITY_TYPES,
	type AuditAction,
	type AuditLog,
	type AuditUser,
	getAuditLogs,
	getAuditUsers,
} from "@/api/audit-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import {
	EntityTypeLabel,
	getAuditLogColumns,
} from "@/sections/columns/audit/audit-log-columns";
import { AuditLogDetailSheet } from "@/sections/views/audit/audit-log-detail-sheet";

const PAGE_LIMIT = 20;
/** Select value meaning "no filter". */
const ALL = "all";
const FILTERS = ["user", "action", "entity", "from", "to"] as const;

/**
 * The clinic's compliance audit trail: who read, created, changed or deleted
 * which record, newest first. Filters live in the URL so a filtered view can
 * be shared or reloaded.
 */
export default function AuditLogPage() {
	const { textGet } = useText();
	const [searchParams, setSearchParams] = useSearchParams();
	const userId = searchParams.get("user") ?? "";
	const action = searchParams.get("action") ?? "";
	const entity = searchParams.get("entity") ?? "";
	const from = searchParams.get("from") ?? "";
	const to = searchParams.get("to") ?? "";

	const [users, setUsers] = React.useState<AuditUser[]>([]);
	const [logs, setLogs] = React.useState<AuditLog[]>([]);
	const [page, setPage] = React.useState(1);
	const [totalPages, setTotalPages] = React.useState(1);
	const [loading, setLoading] = React.useState(true);
	const [selected, setSelected] = React.useState<AuditLog | null>(null);
	const columns = React.useMemo(() => getAuditLogColumns(setSelected), []);

	React.useEffect(() => {
		getAuditUsers().then((res) => {
			if (res.success) setUsers(res.data ?? []);
		});
	}, []);

	// Back to page 1 with the spinner when a filter changes (adjust state during render).
	const queryKey = `${userId}|${action}|${entity}|${from}|${to}`;
	const [prevQueryKey, setPrevQueryKey] = React.useState(queryKey);
	if (prevQueryKey !== queryKey) {
		setPrevQueryKey(queryKey);
		setPage(1);
		setLoading(true);
	}

	function changePage(next: number) {
		setLoading(true);
		setPage(next);
	}

	React.useEffect(() => {
		let cancelled = false;
		getAuditLogs({
			user_id: userId ? Number(userId) : undefined,
			action: (action || undefined) as AuditAction | undefined,
			entity_type: entity || undefined,
			from: from || undefined,
			to: to || undefined,
			page,
			limit: PAGE_LIMIT,
		}).then((res) => {
			if (cancelled) return;
			setLogs(res.success ? (res.data?.items ?? []) : []);
			setTotalPages(res.success ? (res.data?.total_pages ?? 1) : 1);
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [userId, action, entity, from, to, page]);

	function setParam(name: (typeof FILTERS)[number], value: string) {
		setSearchParams(
			(params) => {
				if (!value || value === ALL) params.delete(name);
				else params.set(name, value);
				return params;
			},
			{ replace: true },
		);
	}

	const hasFilters = FILTERS.some((name) => searchParams.has(name));
	const userName = users.find((u) => String(u.id) === userId)?.name;

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("audit.title")}
				description={textGet("audit.description")}
			/>

			<div className="flex flex-wrap items-end gap-3">
				<FilterField label={textGet("audit.filter.user")}>
					<Select
						value={userId || ALL}
						onValueChange={(v) => setParam("user", String(v ?? ""))}
					>
						<SelectTrigger className="w-48">
							<SelectValue>
								{userName ?? textGet("audit.filter.all")}
							</SelectValue>
						</SelectTrigger>
						<SelectContent>
							<SelectItem value={ALL}>{textGet("audit.filter.all")}</SelectItem>
							{users.map((u) => (
								<SelectItem key={u.id} value={String(u.id)}>
									{u.name || `#${u.id}`}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</FilterField>

				<FilterField label={textGet("audit.filter.action")}>
					<Select
						value={action || ALL}
						onValueChange={(v) => setParam("action", String(v ?? ""))}
					>
						<SelectTrigger className="w-40">
							<SelectValue>
								{action
									? textGet(`audit.action.${action.toLowerCase()}`)
									: textGet("audit.filter.all")}
							</SelectValue>
						</SelectTrigger>
						<SelectContent>
							<SelectItem value={ALL}>{textGet("audit.filter.all")}</SelectItem>
							{AUDIT_ACTIONS.map((a) => (
								<SelectItem key={a} value={a}>
									{textGet(`audit.action.${a.toLowerCase()}`)}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</FilterField>

				<FilterField label={textGet("audit.filter.entity")}>
					<Select
						value={entity || ALL}
						onValueChange={(v) => setParam("entity", String(v ?? ""))}
					>
						<SelectTrigger className="w-48">
							<SelectValue>
								{entity ? (
									<EntityTypeLabel type={entity} />
								) : (
									textGet("audit.filter.all")
								)}
							</SelectValue>
						</SelectTrigger>
						<SelectContent>
							<SelectItem value={ALL}>{textGet("audit.filter.all")}</SelectItem>
							{AUDIT_ENTITY_TYPES.map((type) => (
								<SelectItem key={type} value={type}>
									<EntityTypeLabel type={type} />
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</FilterField>

				<FilterField label={textGet("audit.filter.from")}>
					<Input
						type="date"
						className="w-40"
						value={from}
						max={to || undefined}
						onChange={(e) => setParam("from", e.target.value)}
					/>
				</FilterField>
				<FilterField label={textGet("audit.filter.to")}>
					<Input
						type="date"
						className="w-40"
						value={to}
						min={from || undefined}
						onChange={(e) => setParam("to", e.target.value)}
					/>
				</FilterField>

				{hasFilters && (
					<Button
						variant="ghost"
						onClick={() =>
							setSearchParams(new URLSearchParams(), { replace: true })
						}
					>
						<X />
						{textGet("audit.filter.clear")}
					</Button>
				)}
			</div>

			<DataTable
				columns={columns}
				data={logs}
				loading={loading}
				pageCount={totalPages}
				page={page}
				onPageChange={changePage}
				emptyState={
					<p className="py-6 text-center text-sm text-muted-foreground">
						{textGet(hasFilters ? "audit.empty.filtered" : "audit.empty")}
					</p>
				}
			/>

			<AuditLogDetailSheet log={selected} onClose={() => setSelected(null)} />
		</main>
	);
}

function FilterField({
	label,
	children,
}: {
	label: string;
	children: React.ReactNode;
}) {
	return (
		<div className="grid gap-1.5">
			<Label className="text-xs text-muted-foreground">{label}</Label>
			{children}
		</div>
	);
}
