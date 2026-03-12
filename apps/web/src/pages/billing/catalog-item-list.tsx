import { Plus, Trash } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type CatalogItem,
	deleteCatalogItem,
	getAllCatalogItems,
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
	getCatalogItemColumns,
	getCatalogItemColumnsMobile,
} from "@/sections/columns/billing/catalog-item-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useRowStore } from "@/store/row-store";

const CatalogItemList = () => {
	const { checkPermission } = usePermission();
	const navigate = useNavigate();
	const [loading, setLoading] = React.useState(true);
	const { rows } = useRowStore();
	const { isMobile } = useResponsive();
	const { textGet } = useText();
	const [itemList, setItemList] = React.useState<CatalogItem[]>([]);

	React.useEffect(() => {
		setLoading(true);
		getAllCatalogItems().then((res) => {
			if (res.success) {
				setItemList(res.data || []);
			}
			setLoading(false);
		});
	}, []);

	const loadItems = async () => {
		setLoading(true);
		const res = await getAllCatalogItems();
		if (res.success) {
			setItemList(res.data || []);
		}
		setLoading(false);
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
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex flex-row items-center gap-2 sm:gap-5 sm:justify-end justify-start flex-wrap">
					{checkPermission([PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]) && (
						<Button
							onClick={() => navigate("/billing/catalog-items/create")}
							className="mr-auto"
						>
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="billing.catalog-item.create.button" />
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
				</div>
				<div className="sm:max-w-[calc(100vw-6.5rem)] max-w-[calc(100vw-2rem)]">
					{loading ? (
						<div className="h-24 w-full animate-pulse bg-muted rounded-md" />
					) : (
						<DataTable
							searchKey="sku"
							searchPlaceholder={textGet(
								"billing.catalog-item.search.placeholder",
							)}
							columns={
								isMobile
									? getCatalogItemColumnsMobile({
											onEdit: handleEdit,
											onDelete: handleDeleteRow,
										})
									: getCatalogItemColumns({
											onEdit: handleEdit,
											onDelete: handleDeleteRow,
										})
							}
							data={itemList || []}
						/>
					)}
				</div>
			</main>
		</DashboardLayout>
	);
};

export default CatalogItemList;
