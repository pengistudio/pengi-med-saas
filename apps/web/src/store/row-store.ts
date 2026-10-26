import type { DataTableRow } from "@pengi/ui";
import type { RowData } from "@tanstack/react-table";
import { create } from "zustand";

interface SelectedRowState {
	currentId: number;
	open: boolean;
	rows: DataTableRow<RowData>[];
	setCurrentId: (currentId: number) => void;
	setOpen: (open: boolean) => void;
	setRows: (selectedRows: DataTableRow<RowData>[]) => void;
}

export const useRowStore = create<SelectedRowState>((set) => ({
	currentId: -1,
	open: false,
	rows: [],
	setCurrentId: (currentId: number) => set(() => ({ currentId })),
	setOpen: (open: boolean) => set(() => ({ open })),
	setRows: (selectedRows: DataTableRow<RowData>[]) =>
		set(() => ({ rows: selectedRows })),
}));

/**
 * The store holds the selected rows of whichever table is on screen, so it
 * can't know their type; the page that renders that table does.
 */
export const rowsAs = <TData extends RowData>(rows: DataTableRow<RowData>[]) =>
	rows as unknown as DataTableRow<TData>[];
