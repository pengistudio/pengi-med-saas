import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import {
	type Customer,
	getCustomerById,
	updateCustomer,
} from "@/api/billing-service";
import CustomerForm, {
	type CustomerFormValues,
} from "@/sections/forms/billing/customer-form";

export default function EditCustomerPage() {
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [loading, setLoading] = useState(false);
	const [initialLoading, setInitialLoading] = useState(true);
	const [customer, setCustomer] = useState<Customer | undefined>();

	useEffect(() => {
		const fetchCustomer = async () => {
			if (!id) return;
			setInitialLoading(true);
			const res = await getCustomerById(Number(id));
			if (res.success && res.data) {
				setCustomer(res.data);
			} else {
				navigate("/billing/customers");
			}
			setInitialLoading(false);
		};

		fetchCustomer();
	}, [id, navigate]);

	const handleSubmit = async (values: CustomerFormValues) => {
		if (!id) return;
		setLoading(true);
		const res = await updateCustomer(Number(id), {
			identification: values.identification.trim(),
			name: values.name.trim(),
			address: values.address,
			email: values.email,
			phone: values.phone,
		});
		if (res.success) {
			navigate("/billing/customers");
		}
		setLoading(false);
	};

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			{initialLoading ? (
				<div className="flex h-[50vh] items-center justify-center">
					<Loader2 className="h-8 w-8 animate-spin text-primary" />
				</div>
			) : (
				<CustomerForm
					initialData={customer}
					onSubmit={handleSubmit}
					loading={loading}
				/>
			)}
		</main>
	);
}
