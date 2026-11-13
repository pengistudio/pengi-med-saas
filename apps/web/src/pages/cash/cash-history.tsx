import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { ArrowLeft } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { type CashSession, getSessions } from "@/api/cash-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import { CashEmptyState } from "@/components/features/cash/cash-figures";
import { getSessionColumns } from "@/sections/columns/cash/session-columns";

const PAGE_LIMIT = 20;

/** Closed cash sessions, newest first. */
const CashHistoryPage = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [sessions, setSessions] = useState<CashSession[]>([]);
	const [page, setPage] = useState(1);
	const [totalPages, setTotalPages] = useState(1);
	const [loading, setLoading] = useState(true);

	// Show the spinner as soon as the page changes (adjust state during render).
	const [prevPage, setPrevPage] = useState(page);
	if (prevPage !== page) {
		setPrevPage(page);
		setLoading(true);
	}

	useEffect(() => {
		let cancelled = false;
		getSessions({ page, limit: PAGE_LIMIT, status: "closed" }).then((res) => {
			if (cancelled) return;
			if (res.success && res.data) {
				setSessions(res.data.items);
				setTotalPages(res.data.total_pages);
			}
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [page]);

	const columns = useMemo(() => getSessionColumns(), []);

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("cash.history.title")}
				description={textGet("cash.history.description")}
				actions={
					<Button variant="outline" onClick={() => navigate("/cash")}>
						<ArrowLeft className="h-4 w-4" />
						{textGet("cash.page.title")}
					</Button>
				}
			/>
			<DataTable
				columns={columns}
				data={sessions}
				loading={loading}
				pageCount={totalPages}
				page={page}
				onPageChange={setPage}
				emptyState={
					<CashEmptyState
						illustration="empty"
						title={textGet("cash.history.empty")}
						hint={textGet("cash.history.empty.hint")}
					/>
				}
			/>
		</main>
	);
};

export default CashHistoryPage;
