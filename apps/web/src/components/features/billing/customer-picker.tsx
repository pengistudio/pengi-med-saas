import { useText } from "@pengi/shared";
import {
	Combobox,
	ComboboxContent,
	ComboboxEmpty,
	ComboboxInput,
	ComboboxItem,
	ComboboxList,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@pengi/ui";
import { useEffect, useState } from "react";
import {
	type Customer,
	createCustomer,
	getCustomers,
} from "@/api/billing-service";
import CustomerForm, {
	type CustomerFormValues,
} from "@/sections/forms/billing/customer-form";

const SEARCH_DEBOUNCE_MS = 300;
const RESULTS_LIMIT = 10;

export function customerLabel(c: Customer) {
	return `${c.name} - ${c.identification}`;
}

/** Customer combobox that searches on the server (name or identification). */
export function CustomerCombobox({
	value,
	onChange,
	disabled,
	placeholder,
}: {
	value: Customer | null;
	onChange: (customer: Customer | null) => void;
	disabled: boolean;
	placeholder: string;
}) {
	const { textGet } = useText();
	const [search, setSearch] = useState("");
	const [query, setQuery] = useState("");
	const [customers, setCustomers] = useState<Customer[]>([]);

	useEffect(() => {
		const id = setTimeout(() => setQuery(search.trim()), SEARCH_DEBOUNCE_MS);
		return () => clearTimeout(id);
	}, [search]);

	useEffect(() => {
		// Picking a customer writes its label into the input: nothing to search.
		if (value && query === customerLabel(value)) return;
		let cancelled = false;
		getCustomers({ search: query, limit: RESULTS_LIMIT }).then((res) => {
			if (cancelled) return;
			setCustomers(res.success ? (res.data?.items ?? []) : []);
		});
		return () => {
			cancelled = true;
		};
	}, [query, value]);

	return (
		<Combobox
			items={customers}
			filter={null}
			value={value}
			onValueChange={(c) => onChange((c as Customer | null) ?? null)}
			onInputValueChange={setSearch}
			itemToStringLabel={(c: Customer) => customerLabel(c)}
			isItemEqualToValue={(a: Customer, b: Customer) => a.ID === b.ID}
			disabled={disabled}
		>
			<ComboboxInput placeholder={placeholder} disabled={disabled} />
			<ComboboxContent>
				<ComboboxEmpty>{textGet("common.no_results")}</ComboboxEmpty>
				<ComboboxList>
					{(c: Customer) => (
						<ComboboxItem key={c.ID} value={c}>
							{customerLabel(c)}
						</ComboboxItem>
					)}
				</ComboboxList>
			</ComboboxContent>
		</Combobox>
	);
}

/**
 * Quick-create dialog. It is rendered inside the invoice form, and React
 * bubbles synthetic events through portals, so the submit is stopped here or
 * it would also submit the invoice.
 */
export function NewCustomerDialog({
	open,
	onOpenChange,
	onCreated,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onCreated: (customer: Customer) => void;
}) {
	const { textGet } = useText();
	const [loading, setLoading] = useState(false);

	const handleSubmit = async (values: CustomerFormValues) => {
		setLoading(true);
		try {
			const res = await createCustomer({
				identification: values.identification.trim(),
				name: values.name.trim(),
				address: values.address,
				email: values.email,
				phone: values.phone,
			});
			if (res.success && res.data) {
				onCreated(res.data);
				onOpenChange(false);
			}
		} finally {
			setLoading(false);
		}
	};

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-[560px]">
				<DialogHeader>
					<DialogTitle>{textGet("billing.invoice.new_customer")}</DialogTitle>
					<DialogDescription>
						{textGet("billing.invoice.new_customer.desc")}
					</DialogDescription>
				</DialogHeader>
				<div onSubmit={(e) => e.stopPropagation()}>
					<CustomerForm embedded onSubmit={handleSubmit} loading={loading} />
				</div>
			</DialogContent>
		</Dialog>
	);
}
