import { Button, Text, ToggleGroup, ToggleGroupItem } from "@pengi/ui";
import type { Row } from "@tanstack/react-table";
import { Play, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import {
	type DebitNote,
	getAllDebitNotes,
	processDebitNoteSRI,
} from "@/api/billing-service";
import { DataTable } from "@/components/custom/table/data-table";
import usePermission from "@/hooks/use-permission";
import { useText } from "@/hooks/use-text";
import { useResponsive } from "@/hooks/user-responsive";
import { PERMISSIONS, ZERO } from "@/lib/constants";
import {
	getDebitNoteColumns,
	getDebitNoteColumnsMobile,
} from "@/sections/columns/billing/debit-note-columns";
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
	{ value: "authorized", labelKey: "billing.filter.authorized" },
] as const;

const DebitNoteListPage = () => {
	const [loading, setLoading] = useState(true);
	const [processing, setProcessing] = useState(false);
	const [debitNoteList, setDebitNoteList] = useState<DebitNote[]>([]);
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

	const fetchDebitNotes = useCallback(
		async (p: number, s: string, st: string) => {
			setLoading(true);
			const res = await getAllDebitNotes({
				page: p,
				limit: PAGE_LIMIT,
				search: s,
				status: st === "all" ? undefined : st,
			});
			if (res.success && res.data) {
				setDebitNoteList(res.data.items);
				setTotalPages(res.data.total_pages);
			}
			setLoading(false);
		},
		[],
	);

	useEffect(() => {
		fetchDebitNotes(page, search, statusFilter);
	}, [page, search, statusFilter, fetchDebitNotes]);

	useEffect(() => {
		const timer = setTimeout(() => {
			setPage(1);
			setSearch(searchInput);
		}, 400);
		return () => clearTimeout(timer);
	}, [searchInput]);

	const handleRetry = useCallback(
		async (id: number) => {
			const res = await processDebitNoteSRI(id);
			if (res.success) {
				fetchDebitNotes(page, search, statusFilter);
			}
		},
		[page, search, statusFilter, fetchDebitNotes],
	);

	const columns = useMemo(
		() =>
			isMobile
				? getDebitNoteColumnsMobile(handleRetry)
				: getDebitNoteColumns(handleRetry),
		[isMobile, handleRetry],
	);

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex flex-row items-center gap-2 sm:gap-5 sm:justify-end justify-start flex-wrap">
					{checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]) && (
						<Button onClick={() => navigate("/billing/debit-notes/create")}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.debit_note.create.button" />
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
							<Text uuid="billing.debit_note.process.selected" />
						</Button>
					)}
				</div>
				<div className="sm:max-w-[calc(100vw-6.5rem)] max-w-[calc(100vw-2rem)]">
					<DataTable
						searchPlaceholder={textGet("billing.debit_note.search.placeholder")}
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
						data={debitNoteList}
						loading={loading}
						pageCount={totalPages}
						page={page}
						onPageChange={setPage}
						rowClassName={(row) =>
							row.original.status === "failed"
								? "bg-destructive/5 hover:bg-destructive/10"
								: ""
						}
					/>
				</div>
			</main>
		</DashboardLayout>
	);

	async function handleProcessSelected() {
		const parsedRows = rows as Row<DebitNote>[];

		setProcessing(true);
		for (const row of parsedRows) {
			await processDebitNoteSRI(row.original.ID);
		}
		setProcessing(false);
		fetchDebitNotes(page, search, statusFilter);
	}
};

export default DebitNoteListPage;
