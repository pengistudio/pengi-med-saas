import { useText } from "@pengi/shared";
import {
	Button,
	Checkbox,
	Field,
	FieldError,
	FieldLabel,
	Form,
	FormInput,
	FormSelect,
	FormTextArea,
	Input,
} from "@pengi/ui";
import { Loader2, Save } from "lucide-react";
import React from "react";
import { Controller } from "react-hook-form";
import { z } from "zod";
import {
	EXAM_CATEGORIES,
	type ExamCatalogItem,
	type ExamProfile,
} from "@/api/exam-order-service";
import { EXAM_CATEGORY_KEYS, groupCatalog } from "@/lib/exam-orders";

const catalogItemSchema = z.object({
	name: z
		.string()
		.trim()
		.min(1, "clinical.exam_catalog.form.error.name_required")
		.max(255, "clinical.exam_orders.form.error.too_long"),
	category: z.enum(EXAM_CATEGORIES, {
		error: "clinical.exam_catalog.form.error.category",
	}),
	subgroup: z.string().max(255, "clinical.exam_orders.form.error.too_long"),
	default_indications: z
		.string()
		.max(1000, "clinical.exam_orders.form.error.too_long"),
});

export type ExamCatalogItemValues = z.infer<typeof catalogItemSchema>;

/** Creates or edits one exam of the catalog. */
export function ExamCatalogItemForm({
	item,
	loading,
	onSubmit,
}: {
	item?: ExamCatalogItem;
	loading?: boolean;
	onSubmit: (values: ExamCatalogItemValues) => void;
}) {
	const { textGet } = useText();
	return (
		<Form
			schema={catalogItemSchema}
			onSubmit={onSubmit}
			defaultValues={{
				name: item?.name ?? "",
				category: item?.category ?? "laboratory",
				subgroup: item?.subgroup ?? "",
				default_indications: item?.default_indications ?? "",
			}}
		>
			{(field) => (
				<div className="space-y-4">
					<FormInput
						field={field}
						name="name"
						label={textGet("clinical.exam_catalog.field.name")}
					/>
					<FormSelect
						field={field}
						name="category"
						label={textGet("clinical.exam_orders.field.category")}
						options={EXAM_CATEGORIES.map((category) => ({
							value: category,
							label: textGet(EXAM_CATEGORY_KEYS[category]),
						}))}
					/>
					<FormInput
						field={field}
						name="subgroup"
						label={textGet("clinical.exam_catalog.field.subgroup")}
						isOptional
					/>
					<FormTextArea
						field={field}
						name="default_indications"
						label={textGet("clinical.exam_catalog.field.default_indications")}
						isOptional
					/>
					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Save className="mr-2 h-4 w-4" />
							)}
							{textGet("clinical.exam_catalog.form.save")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}

const profileSchema = z.object({
	name: z
		.string()
		.trim()
		.min(1, "clinical.exam_catalog.form.error.name_required")
		.max(255, "clinical.exam_orders.form.error.too_long"),
	item_ids: z
		.array(z.number())
		.min(1, "clinical.exam_catalog.profile.error.items_required"),
});

export type ExamProfileValues = z.infer<typeof profileSchema>;

/** A profile: its name and a multi-select of catalog exams. */
export function ExamProfileForm({
	profile,
	catalog,
	loading,
	onSubmit,
}: {
	profile?: ExamProfile;
	catalog: ExamCatalogItem[];
	loading?: boolean;
	onSubmit: (values: ExamProfileValues) => void;
}) {
	const { textGet } = useText();
	const [query, setQuery] = React.useState("");
	const groups = React.useMemo(
		() =>
			groupCatalog(
				catalog.filter((item) => item.active),
				query,
			),
		[catalog, query],
	);

	return (
		<Form
			schema={profileSchema}
			onSubmit={onSubmit}
			defaultValues={{
				name: profile?.name ?? "",
				item_ids: (profile?.items ?? []).map((item) => item.ID),
			}}
		>
			{(field) => (
				<div className="space-y-4">
					<FormInput
						field={field}
						name="name"
						label={textGet("clinical.exam_catalog.field.name")}
					/>
					<Controller
						control={field.control}
						name="item_ids"
						render={({ field: idsField, fieldState }) => {
							const ids: number[] = idsField.value ?? [];
							const idSet = new Set(ids);
							const toggle = (id: number, checked: boolean) =>
								idsField.onChange(
									checked ? [...ids, id] : ids.filter((v) => v !== id),
								);
							return (
								<Field data-invalid={fieldState.invalid}>
									<FieldLabel>
										{textGet("clinical.exam_catalog.profile.exams", {
											count: ids.length,
										})}
									</FieldLabel>
									<Input
										type="search"
										value={query}
										onChange={(e) => setQuery(e.target.value)}
										placeholder={textGet("clinical.exam_orders.picker.search")}
										aria-label={textGet("clinical.exam_orders.picker.search")}
									/>
									<div className="max-h-72 space-y-3 overflow-y-auto rounded-md border p-2">
										{groups.map((group) => (
											<div key={group.category} className="space-y-1">
												<p className="text-sm font-semibold">
													{textGet(EXAM_CATEGORY_KEYS[group.category])}
												</p>
												{group.subgroups.flatMap((sub) =>
													sub.items.map((item) => {
														const id = `profile-item-${item.ID}`;
														return (
															<div
																key={item.ID}
																className="flex items-center gap-2 pl-2"
															>
																<Checkbox
																	id={id}
																	checked={idSet.has(item.ID)}
																	onCheckedChange={(checked) =>
																		toggle(item.ID, checked === true)
																	}
																/>
																<label
																	htmlFor={id}
																	className="cursor-pointer text-sm"
																>
																	{item.name}
																	{sub.subgroup && (
																		<span className="text-muted-foreground">
																			{" "}
																			· {sub.subgroup}
																		</span>
																	)}
																</label>
															</div>
														);
													}),
												)}
											</div>
										))}
									</div>
									{fieldState.invalid && (
										<FieldError errors={[fieldState.error]} />
									)}
								</Field>
							);
						}}
					/>
					<div className="flex justify-end">
						<Button type="submit" disabled={loading}>
							{loading ? (
								<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							) : (
								<Save className="mr-2 h-4 w-4" />
							)}
							{textGet("clinical.exam_catalog.form.save")}
						</Button>
					</div>
				</div>
			)}
		</Form>
	);
}
