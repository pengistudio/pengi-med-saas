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
	patientColumns,
	patientColumnsMobile,
} from "@/sections/columns/clinical/patient-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { usePatientStore } from "@/store/patient-store";
import { useRowStore } from "@/store/row-store";

const Clinical = () => {
	const [loading, setLoading] = React.useState(true);
	const { rows } = useRowStore();
	const navigate = useNavigate();
	const { patientList, setPatientList } = usePatientStore();
	const { isMobile } = useResponsive();
	const { checkPermission } = usePermission();
	const { textGet } = useText();

	React.useEffect(() => {
		async function fetchPatients() {
			const res = await getAllPatientsWithLastFollowUp();
			if (!res.success) {
				navigate("/");
			} else if (res.data) {
				setPatientList(res.data);
			}
			setLoading(false);
		}
		fetchPatients();
	}, [navigate, setPatientList]);

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
					{loading ? (
						<div className="h-24 w-full animate-pulse bg-muted rounded-md" />
					) : (
						<DataTable
							searchKey="document"
							searchPlaceholder={textGet("clinical.patient.search.placeholder")}
							columns={isMobile ? patientColumnsMobile : patientColumns}
							data={patientList ? patientList : []}
						/>
					)}
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
			setPatientList(res.data);
		}
	}

	function handleCreate() {
		navigate("/clinical/create");
	}
};

export default Clinical;
