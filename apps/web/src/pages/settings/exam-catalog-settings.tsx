import { useText } from "@pengi/shared";
import { Button, Tabs, TabsContent, TabsList, TabsTrigger } from "@pengi/ui";
import { ArrowLeft, Plus, RotateCcw } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import {
	createExamCatalogItem,
	createExamProfile,
	deleteExamCatalogItem,
	deleteExamProfile,
	type ExamCatalogItem,
	type ExamCategory,
	type ExamProfile,
	getExamCatalog,
	getExamProfiles,
	restoreExamCatalogDefaults,
	updateExamCatalogItem,
	updateExamProfile,
} from "@/api/exam-order-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import { ConfirmActionDialog } from "@/components/features/exam-orders/confirm-action-dialog";
import {
	ALL_FILTER,
	CatalogFilters,
	FormDialog,
} from "@/components/features/exam-orders/exam-catalog-parts";
import { catalogSubgroups, normalizeSearch } from "@/lib/exam-orders";
import {
	getExamCatalogColumns,
	getExamProfileColumns,
} from "@/sections/columns/clinical/exam-catalog-columns";
import {
	ExamCatalogItemForm,
	type ExamCatalogItemValues,
	ExamProfileForm,
	type ExamProfileValues,
} from "@/sections/forms/clinical/exam-catalog-form";

type Editing<T> = { mode: "create" } | { mode: "edit"; row: T } | null;
const ALL = ALL_FILTER;

/** `/settings/exam-catalog`: the tenant's exam catalog and profiles. */
export default function ExamCatalogSettingsPage() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const [catalog, setCatalog] = React.useState<ExamCatalogItem[]>([]);
	const [profiles, setProfiles] = React.useState<ExamProfile[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [saving, setSaving] = React.useState(false);
	const [search, setSearch] = React.useState("");
	const [category, setCategory] = React.useState<ExamCategory | typeof ALL>(
		ALL,
	);
	const [subgroup, setSubgroup] = React.useState<string>(ALL);
	const [editingItem, setEditingItem] =
		React.useState<Editing<ExamCatalogItem>>(null);
	const [editingProfile, setEditingProfile] =
		React.useState<Editing<ExamProfile>>(null);
	const [confirmDelete, setConfirmDelete] = React.useState<
		| { kind: "item"; row: ExamCatalogItem }
		| { kind: "profile"; row: ExamProfile }
		| null
	>(null);

	const load = React.useCallback(
		() =>
			Promise.all([getExamCatalog(), getExamProfiles()]).then(
				([catalogRes, profilesRes]) => {
					setCatalog(catalogRes.success ? (catalogRes.data ?? []) : []);
					setProfiles(profilesRes.success ? (profilesRes.data ?? []) : []);
					setLoading(false);
				},
			),
		[],
	);

	React.useEffect(() => {
		load();
	}, [load]);

	const reload = () => {
		setLoading(true);
		load();
	};

	const subgroups = React.useMemo(
		() => catalogSubgroups(catalog, category === ALL ? undefined : category),
		[catalog, category],
	);

	const filtered = React.useMemo(() => {
		const q = normalizeSearch(search);
		return catalog.filter(
			(item) =>
				(category === ALL || item.category === category) &&
				(subgroup === ALL || item.subgroup === subgroup) &&
				(!q || normalizeSearch(item.name).includes(q)),
		);
	}, [catalog, category, subgroup, search]);

	const itemColumns = React.useMemo(
		() =>
			getExamCatalogColumns({
				onEdit: (row) => setEditingItem({ mode: "edit", row }),
				onDelete: (row) => setConfirmDelete({ kind: "item", row }),
				onToggleActive: async (row, active) => {
					const res = await updateExamCatalogItem(row.ID, { active });
					if (res.success && res.data) {
						const updated = res.data;
						setCatalog((items) =>
							items.map((item) => (item.ID === updated.ID ? updated : item)),
						);
					}
				},
			}),
		[],
	);

	const profileColumns = React.useMemo(
		() =>
			getExamProfileColumns({
				onEdit: (row) => setEditingProfile({ mode: "edit", row }),
				onDelete: (row) => setConfirmDelete({ kind: "profile", row }),
				onToggleActive: async (row, active) => {
					const res = await updateExamProfile(row.ID, { active });
					if (res.success && res.data) {
						const updated = res.data;
						setProfiles((items) =>
							items.map((item) => (item.ID === updated.ID ? updated : item)),
						);
					}
				},
			}),
		[],
	);

	async function saveItem(values: ExamCatalogItemValues) {
		if (!editingItem) return;
		setSaving(true);
		const payload = {
			...values,
			subgroup: values.subgroup.trim(),
			default_indications: values.default_indications.trim(),
		};
		try {
			const res =
				editingItem.mode === "edit"
					? await updateExamCatalogItem(editingItem.row.ID, payload)
					: await createExamCatalogItem(payload);
			if (res.success) {
				setEditingItem(null);
				reload();
			}
		} finally {
			setSaving(false);
		}
	}

	async function saveProfile(values: ExamProfileValues) {
		if (!editingProfile) return;
		setSaving(true);
		try {
			const res =
				editingProfile.mode === "edit"
					? await updateExamProfile(editingProfile.row.ID, values)
					: await createExamProfile(values);
			if (res.success) {
				setEditingProfile(null);
				reload();
			}
		} finally {
			setSaving(false);
		}
	}

	async function handleDelete() {
		if (!confirmDelete) return;
		const res =
			confirmDelete.kind === "item"
				? await deleteExamCatalogItem(confirmDelete.row.ID)
				: await deleteExamProfile(confirmDelete.row.ID);
		setConfirmDelete(null);
		if (res.success) reload();
	}

	async function handleRestore() {
		const res = await restoreExamCatalogDefaults();
		if (res.success) reload();
	}

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("clinical.exam_catalog.title")}
				description={textGet("clinical.exam_catalog.description")}
				actions={
					<div className="flex flex-wrap gap-2">
						<Button variant="outline" onClick={() => navigate("/settings")}>
							<ArrowLeft className="mr-2 h-4 w-4" />
							{textGet("clinical.exam_orders.back")}
						</Button>
						<ConfirmActionDialog
							trigger={
								<Button variant="outline">
									<RotateCcw className="mr-2 h-4 w-4" />
									{textGet("clinical.exam_catalog.restore")}
								</Button>
							}
							title={textGet("clinical.exam_catalog.restore.title")}
							description={textGet("clinical.exam_catalog.restore.description")}
							onConfirm={handleRestore}
						/>
					</div>
				}
			/>

			<Tabs defaultValue="exams">
				<TabsList>
					<TabsTrigger value="exams">
						{textGet("clinical.exam_catalog.tab.exams")}
					</TabsTrigger>
					<TabsTrigger value="profiles">
						{textGet("clinical.exam_catalog.tab.profiles")}
					</TabsTrigger>
				</TabsList>

				<TabsContent value="exams" className="space-y-2">
					<DataTable
						columns={itemColumns}
						data={filtered}
						loading={loading}
						searchPlaceholder={textGet("clinical.exam_orders.picker.search")}
						searchValue={search}
						onSearchChange={setSearch}
						toolbarRight={
							<div className="flex flex-wrap gap-2">
								<CatalogFilters
									category={category}
									onCategoryChange={(value) => {
										setCategory(value);
										setSubgroup(ALL);
									}}
									subgroup={subgroup}
									onSubgroupChange={setSubgroup}
									subgroups={subgroups}
								/>
								<Button onClick={() => setEditingItem({ mode: "create" })}>
									<Plus className="mr-2 h-4 w-4" />
									{textGet("clinical.exam_catalog.new_exam")}
								</Button>
							</div>
						}
					/>
				</TabsContent>

				<TabsContent value="profiles" className="space-y-2">
					<div className="flex justify-end pt-2">
						<Button onClick={() => setEditingProfile({ mode: "create" })}>
							<Plus className="mr-2 h-4 w-4" />
							{textGet("clinical.exam_catalog.new_profile")}
						</Button>
					</div>
					<DataTable
						columns={profileColumns}
						data={profiles}
						loading={loading}
					/>
				</TabsContent>
			</Tabs>

			<FormDialog
				open={editingItem !== null}
				busy={saving}
				onClose={() => setEditingItem(null)}
				title={
					editingItem?.mode === "edit"
						? textGet("clinical.exam_catalog.edit_exam")
						: textGet("clinical.exam_catalog.new_exam")
				}
			>
				{editingItem && (
					<ExamCatalogItemForm
						item={editingItem.mode === "edit" ? editingItem.row : undefined}
						loading={saving}
						onSubmit={saveItem}
					/>
				)}
			</FormDialog>

			<FormDialog
				open={editingProfile !== null}
				busy={saving}
				onClose={() => setEditingProfile(null)}
				className="max-h-[90vh] overflow-y-auto sm:max-w-lg"
				title={
					editingProfile?.mode === "edit"
						? textGet("clinical.exam_catalog.edit_profile")
						: textGet("clinical.exam_catalog.new_profile")
				}
			>
				{editingProfile && (
					<ExamProfileForm
						profile={
							editingProfile.mode === "edit" ? editingProfile.row : undefined
						}
						catalog={catalog}
						loading={saving}
						onSubmit={saveProfile}
					/>
				)}
			</FormDialog>

			<ConfirmActionDialog
				open={confirmDelete !== null}
				onOpenChange={(open) => {
					if (!open) setConfirmDelete(null);
				}}
				title={textGet("clinical.exam_catalog.delete.title", {
					name: confirmDelete?.row.name ?? "",
				})}
				description={
					confirmDelete?.kind === "profile"
						? textGet("clinical.exam_catalog.delete.profile_description")
						: textGet("clinical.exam_catalog.delete.exam_description")
				}
				onConfirm={handleDelete}
			/>
		</main>
	);
}
