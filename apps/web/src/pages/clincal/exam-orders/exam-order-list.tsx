import { useText } from "@pengi/shared";
import { Button, Tabs, TabsList, TabsTrigger } from "@pengi/ui";
import { Plus } from "lucide-react";
import React from "react";
import { useNavigate, useSearchParams } from "react-router";
import { type ExamOrder, getExamOrders } from "@/api/exam-order-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import { PatientPickerDialog } from "@/components/features/exam-orders/patient-picker-dialog";
import { Switch } from "@/components/ui/switch";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import {
	EXAM_ORDER_TABS,
	type ExamOrderTab,
	isExamOrderTab,
	tabFilters,
} from "@/lib/exam-orders";
import { getExamOrderColumns } from "@/sections/columns/clinical/exam-order-columns";

const PAGE_LIMIT = 20;

const TAB_KEYS: Record<ExamOrderTab, string> = {
	pending_results: "clinical.exam_orders.tab.pending_results",
	to_review: "clinical.exam_orders.tab.to_review",
	all: "clinical.exam_orders.tab.all",
};

/** `/clinical/exam-orders`: every order of the tenant, by tab. */
export default function ExamOrderListPage() {
	const { textGet } = useText();
	const [searchParams, setSearchParams] = useSearchParams();
	const navigate = useNavigate();
	const { checkPermission } = usePermission();
	const canCreate = checkPermission([
		PERMISSIONS.EXAM_ORDERS.PERMISSION_CREATE_EXAM_ORDER,
	]);
	const [pickerOpen, setPickerOpen] = React.useState(false);
	const rawTab = searchParams.get("tab");
	const tab: ExamOrderTab = isExamOrderTab(rawTab) ? rawTab : "pending_results";
	const allDoctors = searchParams.get("all") === "1";
	const [page, setPage] = React.useState(1);
	const [orders, setOrders] = React.useState<ExamOrder[]>([]);
	const [totalPages, setTotalPages] = React.useState(1);
	const [loading, setLoading] = React.useState(true);
	const columns = React.useMemo(() => getExamOrderColumns(), []);

	// Back to page 1 with the spinner when the filter changes (adjust state during render).
	const queryKey = `${tab}|${allDoctors}`;
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
		getExamOrders({
			...tabFilters(tab, allDoctors),
			page,
			limit: PAGE_LIMIT,
		}).then((res) => {
			if (cancelled) return;
			setOrders(res.success ? (res.data?.items ?? []) : []);
			setTotalPages(res.success ? (res.data?.total_pages ?? 1) : 1);
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [tab, allDoctors, page]);

	function setParam(name: string, value: string | null) {
		setSearchParams(
			(params) => {
				if (value === null) params.delete(name);
				else params.set(name, value);
				return params;
			},
			{ replace: true },
		);
	}

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("clinical.exam_orders.title")}
				description={textGet("clinical.exam_orders.description")}
				actions={
					canCreate && (
						<Button onClick={() => setPickerOpen(true)}>
							<Plus className="mr-2 h-4 w-4" />
							{textGet("clinical.exam_orders.new")}
						</Button>
					)
				}
			/>
			{canCreate && (
				<PatientPickerDialog
					open={pickerOpen}
					onOpenChange={setPickerOpen}
					onPick={(patient) =>
						navigate(`/clinical/exam-orders/new?patient_id=${patient.ID}`)
					}
				/>
			)}
			<div className="flex flex-wrap items-center justify-between gap-3">
				<Tabs
					value={tab}
					onValueChange={(value) =>
						setParam("tab", value === "pending_results" ? null : String(value))
					}
				>
					<TabsList>
						{EXAM_ORDER_TABS.map((value) => (
							<TabsTrigger key={value} value={value}>
								{textGet(TAB_KEYS[value])}
							</TabsTrigger>
						))}
					</TabsList>
				</Tabs>
				{tab === "to_review" && (
					<label
						htmlFor="exam-orders-all-doctors"
						className="flex cursor-pointer items-center gap-2 text-sm"
					>
						<Switch
							id="exam-orders-all-doctors"
							checked={allDoctors}
							onCheckedChange={(checked) =>
								setParam("all", checked ? "1" : null)
							}
						/>
						{textGet("clinical.exam_orders.filter.all_doctors")}
					</label>
				)}
			</div>
			<DataTable
				columns={columns}
				data={orders}
				loading={loading}
				pageCount={totalPages}
				page={page}
				onPageChange={changePage}
				emptyState={
					<p className="py-6 text-center text-sm text-muted-foreground">
						{textGet("clinical.exam_orders.empty")}
					</p>
				}
			/>
		</main>
	);
}
