import { useText } from "@pengi/shared";
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
	useViewport,
} from "@pengi/ui";
import { Plus, Trash } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type CatalogItem,
	deleteCatalogItem,
	getAllCatalogItems,
} from "@/api/billing-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS, ZERO } from "@/lib/constants";
import {
	getCatalogItemColumns,
	getCatalogItemColumnsMobile,
} from "@/sections/columns/billing/catalog-item-columns";
import { useRowStore } from "@/store/row-store";

const PAGE_LIMIT = 20;

const CatalogItemList = () => {
	const { checkPermission } = usePermission();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(true);
	const { rows } = useRowStore();
	const { isPhone } = useViewport();
	const { textGet } = useText();
	const [itemList, setItemList] = React.useState<CatalogItem[]>([]);
	const [page, setPage] = React.useState(1);
	const [totalPages, setTotalPages] = React.useState(1);
	const [search, setSearch] = React.useState("");
	const [searchInput, setSearchInput] = React.useState("");

	const fetchItems = React.useCallback(async (p: number, s: string) => {
		setLoading(true);
		const res = await getAllCatalogItems({
			page: p,
			limit: PAGE_LIMIT,
			search: s,
		});
		if (res.success && res.data) {
			setItemList(res.data.items);
			setTotalPages(res.data.total_pages);
		}
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetchItems(page, search);
	}, [page, search, fetchItems]);

	// Debounce search
	React.useEffect(() => {
		const timer = setTimeout(() => {
			setPage(1);
			setSearch(searchInput);
		}, 400);
		return () => clearTimeout(timer);
	}, [searchInput]);

	const loadItems = async () => {
		fetchItems(page, search);
	};

	const handleEdit = (id: number) => {
		navigate(`/billing/catalog-items/edit/${id}`);
	};

	const handleDeleteRow = async (id: number) => {
		const res = await deleteCatalogItem(id);
		if (res.success) {
			loadItems();
		}
	};

	const handleDelete = async () => {
		// As this list didn't have bulk delete implemented in API, we delete rows individually
		for (const id of rows) {
			await deleteCatalogItem(Number(id));
		}
		loadItems();
	};

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("dashboard.billing.catalog-items")}
				description={textGet("billing.catalog-item.page.description")}
				actions={
					checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]) && (
						<Button onClick={() => navigate("/billing/catalog-items/create")}>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.catalog-item.create.button" />
						</Button>
					)
				}
			/>
			<div>
				<DataTable
					bulkActions={
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
										<Text uuid="billing.catalog-item.delete.description" />
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
					}
					searchPlaceholder={textGet("billing.catalog-item.search.placeholder")}
					searchValue={searchInput}
					onSearchChange={setSearchInput}
					columns={
						isPhone
							? getCatalogItemColumnsMobile({
									onEdit: handleEdit,
									onDelete: handleDeleteRow,
								})
							: getCatalogItemColumns({
									onEdit: handleEdit,
									onDelete: handleDeleteRow,
								})
					}
					data={itemList}
					loading={loading}
					pageCount={totalPages}
					page={page}
					onPageChange={setPage}
				/>
			</div>
		</main>
	);
};

export default CatalogItemList;
