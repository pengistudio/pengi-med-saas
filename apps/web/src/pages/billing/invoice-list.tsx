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
import useToast from "@/hooks/use-toast";
import { useResponsive } from "@/hooks/user-responsive";
import { PERMISSIONS, ZERO } from "@/lib/constants";
import {
	invoiceColumns,
	invoiceColumnsMobile,
} from "@/sections/columns/billing/invoice-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useBillingStore } from "@/store/billing-store";
import { useRowStore } from "@/store/row-store";

const InvoiceListPage = () => {
	const [loading, setLoading] = useState(true);
	const [processing, setProcessing] = useState(false);
	const { rows } = useRowStore();
	const { errorToast, infoToast, successToast } = useToast();
	const navigate = useNavigate();
	const { invoiceList, setInvoiceList } = useBillingStore();
	const { isMobile } = useResponsive();
	const { checkPermission } = usePermission();
	const { textGet } = useText();

	const fetchInvoices = useCallback(() => {
		setLoading(true);
		getAllInvoices()
			.then((res) => {
				if (!res.success) {
					errorToast(res.data);
					return;
				}
				if (res.data) {
					setInvoiceList(res.data);
				}
			})
			.catch(() => {
				infoToast(textGet("error.unexpected.title"), {
					description: textGet("error.unexpected.description"),
				});
			})
			.finally(() => {
				setLoading(false);
			});
	}, [setInvoiceList, errorToast, infoToast, textGet]);

	useEffect(() => {
		fetchInvoices();
	}, [fetchInvoices]);

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex flex-row items-center gap-2 sm:gap-5 sm:justify-end justify-start flex-wrap">
					{checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]) && (
						<Button onClick={() => navigate("/clinical/billing/create")}>
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
							<AlertDialogTrigger>
								<Button variant="outline" disabled={rows.length === ZERO}>
									<Trash className="mr-2 h-4 w-4" />
									<Text uuid="table.button.delete.all.selected" />
								</Button>
							</AlertDialogTrigger>
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
					{loading ? (
						<div className="h-24 w-full animate-pulse bg-muted rounded-md" />
					) : (
						<DataTable
							searchKey="sequential"
							searchPlaceholder={textGet("billing.invoice.search.placeholder")}
							columns={isMobile ? invoiceColumnsMobile : invoiceColumns}
							data={invoiceList || []}
						/>
					)}
				</div>
			</main>
		</DashboardLayout>
	);

	async function handleDelete() {
		const parsedRows = rows as Row<Invoice>[];
		let hasError = false;

		for (const row of parsedRows) {
			const res = await deleteInvoice(row.original.ID);
			if (!res.success) {
				errorToast(null, res.message);
				hasError = true;
				break;
			}
		}

		if (!hasError) {
			successToast(textGet("billing.invoice.delete.success"));
			fetchInvoices();
		}
	}

	async function handleProcessSelected() {
		const parsedRows = rows as Row<Invoice>[];
		const ids = parsedRows.map((row) => row.original.ID);

		setProcessing(true);
		const res = await processMultipleInvoicesSRI(ids);
		setProcessing(false);

		if (res.success) {
			successToast(res.message);
			fetchInvoices();
		} else {
			errorToast(res.data, res.message);
		}
	}
};

export default InvoiceListPage;
