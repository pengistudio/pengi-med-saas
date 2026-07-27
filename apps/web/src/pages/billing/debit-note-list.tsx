import { Button, Text } from "@pengi/ui";
import type { Row } from "@tanstack/react-table";
import { Play, Plus } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
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
	debitNoteColumns,
	debitNoteColumnsMobile,
} from "@/sections/columns/billing/debit-note-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useRowStore } from "@/store/row-store";

const PAGE_LIMIT = 20;

const DebitNoteListPage = () => {
	const [loading, setLoading] = useState(true);
	const [processing, setProcessing] = useState(false);
	const [debitNoteList, setDebitNoteList] = useState<DebitNote[]>([]);
	const [page, setPage] = useState(1);
	const [totalPages, setTotalPages] = useState(1);
	const [search, setSearch] = useState("");
	const [searchInput, setSearchInput] = useState("");
	const { rows } = useRowStore();
	const navigate = useNavigate();
	const { isMobile } = useResponsive();
	const { checkPermission } = usePermission();
	const { textGet } = useText();

	const fetchDebitNotes = useCallback(async (p: number, s: string) => {
		setLoading(true);
		const res = await getAllDebitNotes({
			page: p,
			limit: PAGE_LIMIT,
			search: s,
		});
		if (res.success && res.data) {
			setDebitNoteList(res.data.items);
			setTotalPages(res.data.total_pages);
		}
		setLoading(false);
	}, []);

	useEffect(() => {
		fetchDebitNotes(page, search);
	}, [page, search, fetchDebitNotes]);

	useEffect(() => {
		const timer = setTimeout(() => {
			setPage(1);
			setSearch(searchInput);
		}, 400);
		return () => clearTimeout(timer);
	}, [searchInput]);

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
						columns={isMobile ? debitNoteColumnsMobile : debitNoteColumns}
						data={debitNoteList}
						loading={loading}
						pageCount={totalPages}
						page={page}
						onPageChange={setPage}
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
		fetchDebitNotes(page, search);
	}
};

export default DebitNoteListPage;
