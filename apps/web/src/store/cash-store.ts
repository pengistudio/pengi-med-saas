import { create } from "zustand";
import { type CashSessionDetail, getCurrentSession } from "@/api/cash-service";

/**
 * The open cash session, shared by the Caja page and the payment dialog
 * (which must know whether the desk is open). `loaded` is false until the
 * first fetch answers; `session` is null while the desk is closed. `error`
 * is true when the last fetch failed, so a failure doesn't read as "closed".
 */
interface CashStore {
	session: CashSessionDetail | null;
	loaded: boolean;
	loading: boolean;
	error: boolean;
	fetchCurrent: () => Promise<CashSessionDetail | null>;
	setSession: (session: CashSessionDetail | null) => void;
}

export const useCashStore = create<CashStore>((set) => ({
	session: null,
	loaded: false,
	loading: false,
	error: false,
	fetchCurrent: async () => {
		set({ loading: true });
		const res = await getCurrentSession();
		const session = res.success ? (res.data ?? null) : null;
		set({ session, loaded: true, loading: false, error: !res.success });
		return session;
	},
	setSession: (session) => set({ session, loaded: true, error: false }),
}));
