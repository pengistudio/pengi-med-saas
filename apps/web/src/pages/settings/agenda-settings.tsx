import { useText } from "@pengi/shared";
import { Button, Tabs, TabsContent, TabsList, TabsTrigger } from "@pengi/ui";
import { ArrowLeft, Plus } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	type AppointmentType,
	type AppointmentTypePayload,
	createAppointmentType,
	deleteAppointmentType,
	updateAppointmentType,
} from "@/api/agenda-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import { ScheduleBlocksList } from "@/components/features/agenda/schedule-blocks-list";
import { ConfirmActionDialog } from "@/components/features/exam-orders/confirm-action-dialog";
import { FormDialog } from "@/components/features/exam-orders/exam-catalog-parts";
import { getAppointmentTypeColumns } from "@/sections/columns/agenda/appointment-type-columns";
import { AppointmentTypeForm } from "@/sections/forms/agenda/appointment-type-form";
import { useAgendaStore, useAppointmentTypes } from "@/store/agenda-store";
import { useDoctors } from "@/store/doctors-store";

type Editing = { mode: "create" } | { mode: "edit"; row: AppointmentType };

/** `/settings/agenda`: appointment types and clinic-wide days/hours off. */
export default function AgendaSettingsPage() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { types, loaded } = useAppointmentTypes();
	const refreshTypes = useAgendaStore((s) => s.refreshTypes);
	const { doctors } = useDoctors();
	const [editing, setEditing] = React.useState<Editing | null>(null);
	const [saving, setSaving] = React.useState(false);
	const [toDelete, setToDelete] = React.useState<AppointmentType | null>(null);

	const columns = React.useMemo(
		() =>
			getAppointmentTypeColumns(
				{
					onEdit: (row) => setEditing({ mode: "edit", row }),
					onDelete: (row) => setToDelete(row),
					onToggleActive: async (row, active) => {
						const res = await updateAppointmentType(row.ID, { active });
						if (res.success) refreshTypes();
					},
				},
				doctors,
			),
		[doctors, refreshTypes],
	);

	async function handleSave(payload: AppointmentTypePayload) {
		if (!editing) return;
		setSaving(true);
		const res =
			editing.mode === "edit"
				? await updateAppointmentType(editing.row.ID, payload)
				: await createAppointmentType(payload);
		setSaving(false);
		if (res.success) {
			setEditing(null);
			refreshTypes();
		}
	}

	async function handleDelete() {
		if (!toDelete) return;
		const res = await deleteAppointmentType(toDelete.ID);
		setToDelete(null);
		if (res.success) refreshTypes();
	}

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("agenda.settings.title")}
				description={textGet("agenda.settings.description")}
				actions={
					<Button variant="outline" onClick={() => navigate("/settings")}>
						<ArrowLeft className="mr-2 h-4 w-4" />
						{textGet("agenda.settings.back")}
					</Button>
				}
			/>

			<Tabs defaultValue="types">
				<TabsList>
					<TabsTrigger value="types">
						{textGet("agenda.settings.tab.types")}
					</TabsTrigger>
					<TabsTrigger value="blocks">
						{textGet("agenda.settings.tab.blocks")}
					</TabsTrigger>
				</TabsList>

				<TabsContent value="types" className="space-y-2">
					<div className="flex flex-wrap items-center justify-between gap-2 pt-2">
						<p className="text-sm text-muted-foreground">
							{textGet("agenda.types.description")}
						</p>
						<Button onClick={() => setEditing({ mode: "create" })}>
							<Plus className="mr-2 h-4 w-4" />
							{textGet("agenda.types.new")}
						</Button>
					</div>
					<DataTable columns={columns} data={types} loading={!loaded} />
				</TabsContent>

				<TabsContent value="blocks" className="space-y-2 pt-2">
					<p className="text-sm text-muted-foreground">
						{textGet("agenda.blocks.clinic_description")}
					</p>
					<ScheduleBlocksList scope="clinic" />
				</TabsContent>
			</Tabs>

			<FormDialog
				open={editing !== null}
				busy={saving}
				onClose={() => setEditing(null)}
				className="max-h-[90vh] overflow-y-auto sm:max-w-md"
				title={
					editing?.mode === "edit"
						? textGet("agenda.types.edit")
						: textGet("agenda.types.new")
				}
			>
				{editing && (
					<AppointmentTypeForm
						type={editing.mode === "edit" ? editing.row : undefined}
						loading={saving}
						onSubmit={handleSave}
					/>
				)}
			</FormDialog>

			<ConfirmActionDialog
				open={toDelete !== null}
				onOpenChange={(open) => {
					if (!open) setToDelete(null);
				}}
				title={textGet("agenda.types.delete.title", {
					name: toDelete?.name ?? "",
				})}
				description={textGet("agenda.types.delete.description")}
				onConfirm={handleDelete}
			/>
		</main>
	);
}
