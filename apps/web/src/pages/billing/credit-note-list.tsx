import { Button, Text, ToggleGroup, ToggleGroupItem } from "@pengi/ui";
import type { Row } from "@tanstack/react-table";
import { Play, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import {
	type CreditNote,
	getAllCreditNotes,
	processCreditNoteSRI,
} from "@/api/billing-service";
import { DataTable } from "@/components/custom/table/data-table";
import usePermission from "@/hooks/use-permission";
import { useText } from "@/hooks/use-text";
import { useResponsive } from "@/hooks/user-responsive";
import { PERMISSIONS, ZERO } from "@/lib/constants";
import {
	getCreditNoteColumns,
	getCreditNoteColumnsMobile,
} from "@/sections/columns/billing/credit-note-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useRowStore } from "@/store/row-store";

const PAGE_LIMIT = 20;

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

const CreditNoteListPage = () => {
	const [loading, setLoading] = useState(true);
	const [processing, setProcessing] = useState(false);
	const [creditNoteList, setCreditNoteList] = useState<CreditNote[]>([]);
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

	const fetchCreditNotes = useCallback(
		async (p: number, s: string, st: string) => {
			setLoading(true);
			const res = await getAllCreditNotes({
				page: p,
				limit: PAGE_LIMIT,
				search: s,
				status: st === "all" ? undefined : st,
			});
			if (res.success && res.data) {
				setCreditNoteList(res.data.items);
				setTotalPages(res.data.total_pages);
			}
			setLoading(false);
		},
		[],
	);

	useEffect(() => {
		fetchCreditNotes(page, search, statusFilter);
	}, [page, search, statusFilter, fetchCreditNotes]);

	useEffect(() => {
		const timer = setTimeout(() => {
			setPage(1);
			setSearch(searchInput);
		}, 400);
		return () => clearTimeout(timer);
	}, [searchInput]);

	const handleRetry = useCallback(
		async (id: number) => {
			const res = await processCreditNoteSRI(id);
			if (res.success) {
				fetchCreditNotes(page, search, statusFilter);
			}
		},
		[page, search, statusFilter, fetchCreditNotes],
	);

	const columns = useMemo(
		() =>
			isMobile
				? getCreditNoteColumnsMobile(handleRetry)
				: getCreditNoteColumns(handleRetry),
		[isMobile, handleRetry],
	);

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex flex-row items-center gap-2 sm:gap-5 sm:justify-end justify-start flex-wrap">
					{checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]) && (
						<Button onClick={() => navigate("/billing/credit-notes/create")}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.credit_note.create.button" />
						</Button>
					)}

					{checkPermission([
						PERMISSIONS.BILLING.PERMISSION_MANAGE_SRI_SETTINGS,
					]) && (
						<Button
							variant="secondary"
							disabled={rows.length === ZERO || processing}
							onClick={handleProcessSelected}
							className="md:ml-auto"
						>
							<Play className="mr-2 h-4 w-4" />
							<Text uuid="billing.credit_note.process.selected" />
						</Button>
					)}
				</div>
				<div className="sm:max-w-[calc(100vw-6.5rem)] max-w-[calc(100vw-2rem)]">
					<DataTable
						searchPlaceholder={textGet(
							"billing.credit_note.search.placeholder",
						)}
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
						data={creditNoteList}
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

	async function handleProcessSelected() {
		const parsedRows = rows as Row<CreditNote>[];

		setProcessing(true);
		for (const row of parsedRows) {
			await processCreditNoteSRI(row.original.ID);
		}
		setProcessing(false);
		fetchCreditNotes(page, search, statusFilter);
	}
};

export default CreditNoteListPage;
