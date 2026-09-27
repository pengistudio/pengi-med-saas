import React from "react";
import { create } from "zustand";
import { getMySignature, type MySignature } from "@/api/signature-service";

type SignatureState = {
	signature: MySignature | null;
	fetchSignature: () => Promise<void>;
	setSignature: (signature: MySignature) => void;
};

// The current user's electronic signature, shared by the profile card and
// every "Sign" button so a page with many documents fetches it once.
let inflight: Promise<void> | null = null;

export const useSignatureStore = create<SignatureState>((set) => ({
	signature: null,
	fetchSignature: () => {
		inflight ??= getMySignature()
			.then((res) => {
				if (res.success && res.data) set({ signature: res.data });
			})
			.finally(() => {
				inflight = null;
			});
		return inflight;
	},
	setSignature: (signature) => set({ signature }),
}));

/** Loads the current user's signature on mount (only when `enabled`). */
export function useMySignature(enabled = true) {
	const signature = useSignatureStore((s) => s.signature);
	const fetchSignature = useSignatureStore((s) => s.fetchSignature);
	React.useEffect(() => {
		if (enabled) fetchSignature();
	}, [enabled, fetchSignature]);
	return signature;
}
