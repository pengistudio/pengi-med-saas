import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	AlertDialogTrigger,
	Button,
	Text,
	ToggleGroup,
	ToggleGroupItem,
} from "@pengi/ui";
import type { Row } from "@tanstack/react-table";
import { Play, Plus, Trash } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import {
	deleteInvoice,
	getAllInvoices,
	type Invoice,
	processInvoiceSRI,
	processMultipleInvoicesSRI,
} from "@/api/billing-service";
import { DataTable } from "@/components/custom/table/data-table";
import usePermission from "@/hooks/use-permission";
import { useText } from "@/hooks/use-text";
import { useResponsive } from "@/hooks/user-responsive";
import { PERMISSIONS, ZERO } from "@/lib/constants";
import {
	getInvoiceColumns,
	getInvoiceColumnsMobile,
} from "@/sections/columns/billing/invoice-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useRowStore } from "@/store/row-store";

const PAGE_LIMIT = 20;
const TRANSIENT_STATUSES = ["pending", "processing", "signed", "validated"];
const POLL_INTERVAL_MS = 4000;
const POLL_MAX_DURATION_MS = 2 * 60 * 1000;

const STATUS_FILTERS = [
	{ value: "all", labelKey: "billing.filter.all" },
	{
		value: "pending,processing,signed,validated",
		labelKey: "billing.filter.pending",
	},
	{ value: "failed", labelKey: "billing.filter.failed" },
	{
		value: "connection_error",
		labelKey: "billing.filter.connection_error",
	},
	{ value: "authorized", labelKey: "billing.filter.authorized" },
] as const;

const InvoiceListPage = () => {
	const [loading, setLoading] = useState(true);
	const [processing, setProcessing] = useState(false);
	const [invoiceList, setInvoiceList] = useState<Invoice[]>([]);
	const [page, setPage] = useState(1);
	const [totalPages, setTotalPages] = useState(1);
	const [search, setSearch] = useState("");
	const [searchInput, setSearchInput] = useState("");
	const [statusFilter, setStatusFilter] = useState("all");
	const { rows } = useRowStore();
	const navigate = useNavigate();
	const { isMobile } = useResponsive();
	const { checkPermission } = usePermission();
	const { textGet } = useText();

	const fetchInvoices = useCallback(
		async (p: number, s: string, st: string, silent = false) => {
			if (!silent) setLoading(true);
			const res = await getAllInvoices({
				page: p,
				limit: PAGE_LIMIT,
				search: s,
				status: st === "all" ? undefined : st,
			});
			if (res.success && res.data) {
				setInvoiceList(res.data.items);
				setTotalPages(res.data.total_pages);
			}
			if (!silent) setLoading(false);
		},
		[],
	);

	useEffect(() => {
		fetchInvoices(page, search, statusFilter);
	}, [page, search, statusFilter, fetchInvoices]);

	// Invoice processing is async (RabbitMQ worker). Poll while any row is in a
	// transient status so the table reflects progress without a manual reload.
	// pollStartedAtRef tracks the real wall-clock start across re-renders (a plain
	// local var would reset every time invoiceList changes and the effect re-runs).
	const pollStartedAtRef = useRef<number | null>(null);
	useEffect(() => {
		const hasTransient = invoiceList.some((invoice) =>
			TRANSIENT_STATUSES.includes(invoice.status),
		);
		if (!hasTransient) {
			pollStartedAtRef.current = null;
			return;
		}
		if (pollStartedAtRef.current === null) {
			pollStartedAtRef.current = Date.now();
		}
		if (Date.now() - pollStartedAtRef.current >= POLL_MAX_DURATION_MS) return;

		const timeout = setTimeout(() => {
			fetchInvoices(page, search, statusFilter, true);
		}, POLL_INTERVAL_MS);

		return () => clearTimeout(timeout);
	}, [invoiceList, page, search, statusFilter, fetchInvoices]);

	// Debounce search
	useEffect(() => {
		const timer = setTimeout(() => {
			setPage(1);
			setSearch(searchInput);
		}, 400);
		return () => clearTimeout(timer);
	}, [searchInput]);

	const handleRetry = useCallback(
		async (id: number) => {
			const res = await processInvoiceSRI(id);
			if (res.success) {
				fetchInvoices(page, search, statusFilter, true);
			}
		},
		[page, search, statusFilter, fetchInvoices],
	);

	const columns = useMemo(
		() =>
			isMobile
				? getInvoiceColumnsMobile(handleRetry)
				: getInvoiceColumns(handleRetry),
		[isMobile, handleRetry],
	);

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex flex-row items-center gap-2 sm:gap-5 sm:justify-end justify-start flex-wrap">
					{checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]) && (
						<Button onClick={() => navigate("/billing/create")}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.invoice.create.button" />
						</Button>
					)}

					{checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]) && (
						<Button
							variant="secondary"
							disabled={rows.length === ZERO || processing}
							onClick={handleProcessSelected}
							className="md:ml-auto"
						>
							<Play className="mr-2 h-4 w-4" />
							<Text uuid="billing.invoice.process.selected" />
						</Button>
					)}

					<AlertDialog>
						{checkPermission([
							PERMISSIONS.BILLING.PERMISSION_DELETE_BILLING,
						]) && (
							<AlertDialogTrigger
								render={
									<Button variant="outline" disabled={rows.length === ZERO}>
										<Trash className="mr-2 h-4 w-4" />
										<Text uuid="table.button.delete.all.selected" />
									</Button>
								}
							/>
						)}
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>
									<Text uuid="dialog.title.absolutely.sure" />
								</AlertDialogTitle>
								<AlertDialogDescription>
									<Text uuid="billing.invoice.delete.description" />
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>
									<Text uuid="form.cancel" />
								</AlertDialogCancel>
								<AlertDialogAction onClick={handleDelete}>
									<Text uuid="form.continue" />
								</AlertDialogAction>
							</AlertDialogFooter>
						</AlertDialogContent>
					</AlertDialog>
				</div>
				<div className="sm:max-w-[calc(100vw-6.5rem)] max-w-[calc(100vw-2rem)]">
					<DataTable
						searchPlaceholder={textGet("billing.invoice.search.placeholder")}
						searchValue={searchInput}
						onSearchChange={setSearchInput}
						toolbarRight={
							<ToggleGroup
								value={[statusFilter]}
								onValueChange={(value) => {
									setStatusFilter(value[0] ?? "all");
									setPage(1);
								}}
							>
								{STATUS_FILTERS.map((filter) => (
									<ToggleGroupItem key={filter.value} value={filter.value}>
										<Text uuid={filter.labelKey} />
									</ToggleGroupItem>
								))}
							</ToggleGroup>
						}
						columns={columns}
						data={invoiceList}
						loading={loading}
						pageCount={totalPages}
						page={page}
						onPageChange={setPage}
						rowClassName={(row) =>
							row.original.status === "failed"
								? "bg-destructive/5 hover:bg-destructive/10"
								: row.original.status === "connection_error"
									? "bg-amber-500/5 hover:bg-amber-500/10"
									: ""
						}
					/>
				</div>
			</main>
		</DashboardLayout>
	);

	async function handleDelete() {
		const parsedRows = rows as Row<Invoice>[];
		for (const row of parsedRows) {
			const res = await deleteInvoice(row.original.ID);
			if (!res.success) break;
		}
		fetchInvoices(page, search, statusFilter);
	}

	async function handleProcessSelected() {
		const parsedRows = rows as Row<Invoice>[];
		const ids = parsedRows.map((row) => row.original.ID);

		setProcessing(true);
		const res = await processMultipleInvoicesSRI(ids);
		setProcessing(false);

		if (res.success) {
			fetchInvoices(page, search, statusFilter);
		}
	}
};

export default InvoiceListPage;
