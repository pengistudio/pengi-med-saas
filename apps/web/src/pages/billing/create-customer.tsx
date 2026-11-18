import { useState } from "react";
import { useNavigate } from "react-router";
import { createCustomer } from "@/api/billing-service";
import CustomerForm, {
	type CustomerFormValues,
} from "@/sections/forms/billing/customer-form";

export default function CreateCustomerPage() {
	const navigate = useNavigate();
	const [loading, setLoading] = useState(false);

	const handleSubmit = async (values: CustomerFormValues) => {
		setLoading(true);
		const res = await createCustomer({
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
			<CustomerForm onSubmit={handleSubmit} loading={loading} />
		</main>
	);
}
