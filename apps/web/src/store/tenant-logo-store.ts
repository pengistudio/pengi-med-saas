import { create } from "zustand";
import { getLogo } from "@/api/tenant-service";

/** A logo this much wider than tall is a wordmark: it gets the whole header. */
const WIDE_LOGO_RATIO = 1.8;

type TenantLogoState = {
	/** Tenant the logo below belongs to. */
	slug?: string;
	/** Object URL of the clinic's logo: undefined while loading, null when it has none. */
	url?: string | null;
	wide: boolean;
	/** Loads the logo once per tenant; `force` re-reads it (after an upload). */
	load: (slug: string, force?: boolean) => Promise<void>;
};

const aspectRatio = (url: string) =>
	new Promise<number>((resolve) => {
		const img = new Image();
		img.onload = () => resolve(img.naturalWidth / (img.naturalHeight || 1));
		img.onerror = () => resolve(1);
		img.src = url;
	});

export const useTenantLogoStore = create<TenantLogoState>((set, get) => ({
	slug: undefined,
	url: undefined,
	wide: false,
	load: async (slug, force = false) => {
		const current = get();
		if (!force && current.slug === slug) return;
		if (current.slug !== slug) set({ slug, url: undefined, wide: false });

		const res = await getLogo();
		const url =
			res.success && res.data ? window.URL.createObjectURL(res.data) : null;
		const wide = url ? (await aspectRatio(url)) >= WIDE_LOGO_RATIO : false;

		// The tenant changed while this request was in flight.
		if (get().slug !== slug) {
			if (url) window.URL.revokeObjectURL(url);
			return;
		}
		const previous = get().url;
		if (previous) window.URL.revokeObjectURL(previous);
		set({ url, wide });
	},
}));
