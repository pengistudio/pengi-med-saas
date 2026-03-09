import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { Invoice } from "@/api/billing-service";

interface BillingState {
	invoiceList: Invoice[] | null;
	setInvoiceList: (list: Invoice[]) => void;
	cleanBilling: () => void;
}

export const useBillingStore = create<BillingState>()(
	persist(
		(set) => ({
			invoiceList: null,
			setInvoiceList: (list: Invoice[]) => set({ invoiceList: list }),
			cleanBilling: () =>
				set({
					invoiceList: null,
				}),
		}),
		{
			name: "billing-storage",
			storage: createJSONStorage(() => sessionStorage),
		},
	),
);
