import type { Row } from "@tanstack/react-table";
import { Play, Plus, Trash } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import {
	deleteInvoice,
	getAllInvoices,
	type Invoice,
	processMultipleInvoicesSRI,
} from "@/api/billing-service";
import { DataTable } from "@/components/custom/table/data-table";
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
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import usePermission from "@/hooks/use-permission";
import { useText } from "@/hooks/use-text";
import { useResponsive } from "@/hooks/user-responsive";
import { PERMISSIONS, ZERO } from "@/lib/constants";
import {
	invoiceColumns,
	invoiceColumnsMobile,
} from "@/sections/columns/billing/invoice-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useRowStore } from "@/store/row-store";

const PAGE_LIMIT = 20;

const InvoiceListPage = () => {
	const [loading, setLoading] = useState(true);
	const [processing, setProcessing] = useState(false);
	const [invoiceList, setInvoiceList] = useState<Invoice[]>([]);
	const [page, setPage] = useState(1);
	const [totalPages, setTotalPages] = useState(1);
	const [search, setSearch] = useState("");
	const [searchInput, setSearchInput] = useState("");
	const { rows } = useRowStore();
	const navigate = useNavigate();
	const { isMobile } = useResponsive();
	const { checkPermission } = usePermission();
	const { textGet } = useText();

	const fetchInvoices = useCallback(async (p: number, s: string) => {
		setLoading(true);
		const res = await getAllInvoices({ page: p, limit: PAGE_LIMIT, search: s });
		if (res.success && res.data) {
			setInvoiceList(res.data.items);
			setTotalPages(res.data.total_pages);
		}
		setLoading(false);
	}, []);

	useEffect(() => {
		fetchInvoices(page, search);
	}, [page, search, fetchInvoices]);

	// Debounce search
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
						columns={isMobile ? invoiceColumnsMobile : invoiceColumns}
						data={invoiceList}
						loading={loading}
						pageCount={totalPages}
						page={page}
						onPageChange={setPage}
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
		fetchInvoices(page, search);
	}

	async function handleProcessSelected() {
		const parsedRows = rows as Row<Invoice>[];
		const ids = parsedRows.map((row) => row.original.ID);

		setProcessing(true);
		const res = await processMultipleInvoicesSRI(ids);
		setProcessing(false);

		if (res.success) {
			fetchInvoices(page, search);
		}
	}
};

export default InvoiceListPage;
