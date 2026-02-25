import type { Row } from "@tanstack/react-table";
import { create } from "zustand";

interface SelectedRowState {
	currentId: number;
	open: boolean;
	rows: Row<unknown>[];
	setCurrentId: (currentId: number) => void;
	setOpen: (open: boolean) => void;
	setRows: (selectedRows: Row<unknown>[]) => void;
}

export const useRowStore = create<SelectedRowState>((set) => ({
	currentId: -1,
	open: false,
	rows: [],
	setCurrentId: (currentId: number) => set(() => ({ currentId })),
	setOpen: (open: boolean) => set(() => ({ open })),
	setRows: (selectedRows: Row<unknown>[]) =>
		set(() => ({ rows: selectedRows })),
}));
