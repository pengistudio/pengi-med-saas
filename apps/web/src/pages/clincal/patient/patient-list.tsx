import type { Row } from "@tanstack/react-table";
import { Plus, Trash } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	deleteMultiplePatients,
	getAllPatientsWithLastFollowUp,
	type Patient,
} from "@/api/clinical-service";
import { DataTable } from "@/components/custom/table/data-table";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import usePermission from "@/hooks/use-permission";
import { useText } from "@/hooks/use-text";
import { useResponsive } from "@/hooks/user-responsive";
import { PERMISSIONS, ZERO } from "@/lib/constants";
import {
	patientColumnsMobile,
	usePatientColumns,
} from "@/sections/columns/clinical/patient-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useRowStore } from "@/store/row-store";

const PAGE_LIMIT = 20;

const Clinical = () => {
	const [loading, setLoading] = React.useState(true);
	const [patients, setPatients] = React.useState<Patient[]>([]);
	const [page, setPage] = React.useState(1);
	const [totalPages, setTotalPages] = React.useState(1);
	const [_, setTotal] = React.useState(0);
	const [search, setSearch] = React.useState("");
	const [searchInput, setSearchInput] = React.useState("");

	const { rows } = useRowStore();
	const navigate = useNavigate();
	const { isMobile } = useResponsive();
	const { checkPermission } = usePermission();
	const patientColumns = usePatientColumns();
	const { textGet } = useText();

	const fetchPatients = React.useCallback(async (p: number, s: string) => {
		setLoading(true);
		const res = await getAllPatientsWithLastFollowUp({
			page: p,
			limit: PAGE_LIMIT,
			search: s,
		});
		if (res.success && res.data) {
			setPatients(res.data.items);
			setTotal(res.data.total);
			setTotalPages(res.data.total_pages);
		}
		setLoading(false);
	}, []);

	React.useEffect(() => {
		fetchPatients(page, search);
	}, [page, search, fetchPatients]);

	// Debounce search input
	React.useEffect(() => {
		const timer = setTimeout(() => {
			setPage(1);
			setSearch(searchInput);
		}, 400);
		return () => clearTimeout(timer);
	}, [searchInput]);

	return (
		<DashboardLayout>
			<main className="grid items-start gap-4 p-4 sm:px-6 sm:py-0">
				<div className="flex flex-row items-center gap-2 sm:gap-5 sm:justify-end justify-start flex-wrap">
					{checkPermission([
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_PATIENT,
					]) && (
						<Button onClick={handleCreate} className="mr-auto">
							<Plus className="mr-2 h-4 w-4" />
							<Text uuid="clinical.patient.create" />
						</Button>
					)}
					<AlertDialog>
						{checkPermission([
							PERMISSIONS.MEDICAL_RECORD.PERMISSION_DELETE_PATIENT,
						]) && (
							<AlertDialogTrigger
								render={
									<Button
										variant="outline"
										disabled={rows.length === ZERO}
										className="md:ml-auto"
									>
										<Trash className="mr-2 h-4 w-4" />
										<Text uuid="table.button.delete.all.selected" />
									</Button>
								}
							/>
						)}
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>
									<Text uuid="dialog.title.absolutely.sure" />
								</AlertDialogTitle>
								<AlertDialogDescription>
									<Text uuid="dialog.description.user.delete" />
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>
									<Text uuid="form.cancel" />
								</AlertDialogCancel>
								<AlertDialogAction onClick={handleDelete}>
									<Text uuid="form.continue" />
								</AlertDialogAction>
							</AlertDialogFooter>
						</AlertDialogContent>
					</AlertDialog>
				</div>

				<div className="sm:max-w-[calc(100vw-6.5rem)] max-w-[calc(100vw-2rem)]">
					<DataTable
						columns={isMobile ? patientColumnsMobile : patientColumns}
						data={patients}
						loading={loading}
						searchPlaceholder={textGet("clinical.patient.search.placeholder")}
						searchValue={searchInput}
						onSearchChange={setSearchInput}
						pageCount={totalPages}
						page={page}
						onPageChange={setPage}
					/>
				</div>
			</main>
		</DashboardLayout>
	);

	async function handleDelete() {
		const parsedRows = rows as Row<Patient>[];
		const res = await deleteMultiplePatients(
			parsedRows.map((row) => row.original.ID),
		);
		if (res.success && res.data) {
			setPatients(res.data.items ?? []);
			setTotal(res.data.total ?? 0);
			setTotalPages(res.data.total_pages ?? 1);
		}
	}

	function handleCreate() {
		navigate("/clinical/create");
	}
};

export default Clinical;
