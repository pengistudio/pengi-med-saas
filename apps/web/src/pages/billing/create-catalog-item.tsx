import { useState } from "react";
import { useNavigate } from "react-router";
import {
	type CreateCatalogItemPayload,
	createCatalogItem,
} from "@/api/billing-service";
import CatalogItemForm, {
	type FormValues,
} from "@/sections/forms/billing/catalog-item-form";
import { DashboardLayout } from "@/sections/template/dashboard-template";

export default function CreateCatalogItemPage() {
	const navigate = useNavigate();
	const [loading, setLoading] = useState(false);

	const handleSubmit = async (values: FormValues) => {
		setLoading(true);
		const payload: CreateCatalogItemPayload = {
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

		const res = await createCatalogItem(payload);
		if (res.success) {
			navigate("/billing/catalog-items");
		}
		setLoading(false);
	};

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<CatalogItemForm onSubmit={handleSubmit} loading={loading} />
			</main>
		</DashboardLayout>
	);
}
