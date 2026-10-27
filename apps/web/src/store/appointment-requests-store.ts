import { create } from "zustand";
import { getPendingAppointmentRequestCount } from "@/api/appointment-request-service";

/** Pending portal booking requests: the nav badge and the inbox share it. */
interface AppointmentRequestsStore {
	pendingCount: number;
	refreshPendingCount: () => Promise<void>;
	resetPendingCount: () => void;
}

export const useAppointmentRequestsStore = create<AppointmentRequestsStore>(
	(set) => ({
		pendingCount: 0,
		refreshPendingCount: async () => {
			const pendingCount = await getPendingAppointmentRequestCount();
			set({ pendingCount });
		},
		resetPendingCount: () => set({ pendingCount: 0 }),
	}),
);
