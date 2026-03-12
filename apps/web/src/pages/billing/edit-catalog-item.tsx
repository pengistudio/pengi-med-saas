import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import {
	type CatalogItem,
	getCatalogItemById,
	type UpdateCatalogItemPayload,
	updateCatalogItem,
} from "@/api/billing-service";
import CatalogItemForm, {
	type FormValues,
} from "@/sections/forms/billing/catalog-item-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

export default function EditCatalogItemPage() {
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = useState(false);
	const [initialLoading, setInitialLoading] = useState(true);
	const [itemData, setItemData] = useState<CatalogItem | undefined>();

	useEffect(() => {
		const fetchItem = async () => {
			if (!id) return;
			setInitialLoading(true);
			const res = await getCatalogItemById(Number(id));
			if (res.success && res.data) {
				setItemData(res.data);
			} else {
				navigate("/billing/catalog-items");
			}
			setInitialLoading(false);
		};

		fetchItem();
	}, [id, navigate]);

	const handleSubmit = async (values: FormValues) => {
		if (!id) return;
		setLoading(true);
		const payload: UpdateCatalogItemPayload = {
			name: values.name,
			sku: values.sku,
			description: values.description,
			unit_price: Number(values.unit_price),
			tax: values.tax ? Number(values.tax) : undefined,
			tax_code: values.tax_code,
			tax_percentage_code: values.tax_percentage_code,
			ice_tax: values.ice_tax ? Number(values.ice_tax) : undefined,
			ice_tax_code: values.ice_tax_code,
			ice_tax_percentage_code: values.ice_tax_percentage_code,
		};

		const res = await updateCatalogItem(Number(id), payload);
		if (res.success) {
			navigate("/billing/catalog-items");
		}
		setLoading(false);
	};

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				{initialLoading ? (
					<div className="flex h-[50vh] items-center justify-center">
						<Loader2 className="h-8 w-8 animate-spin text-primary" />
					</div>
				) : (
					<CatalogItemForm
						initialData={itemData}
						onSubmit={handleSubmit}
						loading={loading}
					/>
				)}
			</main>
		</DashboardLayout>
	);
}
