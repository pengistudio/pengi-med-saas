import { useText } from "@pengi/shared";
import {
	Button,
	Checkbox,
	Input,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import { Layers, Plus, Search } from "lucide-react";
import React from "react";
import {
	EXAM_CATEGORIES,
	type ExamCatalogItem,
	type ExamCategory,
	type ExamProfile,
} from "@/api/exam-order-service";
import {
	addCatalogItems,
	type DraftExamItem,
	draftFreeText,
	EXAM_CATEGORY_KEYS,
	expandProfile,
	groupCatalog,
} from "@/lib/exam-orders";

interface ExamPickerProps {
	catalog: ExamCatalogItem[];
	profiles: ExamProfile[];
	value: DraftExamItem[];
	onChange: (drafts: DraftExamItem[]) => void;
}

/**
 * Picks the exams of an order: the catalog grouped by category and subgroup
 * with a search, profiles that add all their exams at once, and an exam
 * written by hand with its category.
 */
export function ExamPicker({
	catalog,
	profiles,
	value,
	onChange,
}: ExamPickerProps) {
	const { textGet } = useText();
	const [query, setQuery] = React.useState("");
	const [freeName, setFreeName] = React.useState("");
	const [freeCategory, setFreeCategory] = React.useState<ExamCategory>("other");

	const groups = React.useMemo(
		() =>
			groupCatalog(
				catalog.filter((item) => item.active),
				query,
			),
		[catalog, query],
	);
	const selected = React.useMemo(() => {
		const map = new Map<number, DraftExamItem>();
		for (const draft of value)
			if (draft.catalog_item_id !== undefined)
				map.set(draft.catalog_item_id, draft);
		return map;
	}, [value]);

	function toggle(item: ExamCatalogItem, checked: boolean) {
		if (checked) {
			onChange(addCatalogItems(value, [item]));
			return;
		}
		const draft = selected.get(item.ID);
		if (!draft || draft.locked) return;
		onChange(value.filter((d) => d.key !== draft.key));
	}

	function addFree() {
		if (!freeName.trim()) return;
		onChange([...value, draftFreeText(freeName, freeCategory)]);
		setFreeName("");
	}

	const activeProfiles = profiles.filter((profile) => profile.active);

	return (
		<div className="space-y-4">
			{activeProfiles.length > 0 && (
				<div className="space-y-2">
					<p className="text-sm font-medium">
						{textGet("clinical.exam_orders.picker.profiles")}
					</p>
					<div className="flex flex-wrap gap-2">
						{activeProfiles.map((profile) => (
							<Button
								key={profile.ID}
								type="button"
								size="sm"
								variant="outline"
								title={(profile.items ?? []).map((i) => i.name).join(", ")}
								onClick={() => onChange(expandProfile(value, profile))}
							>
								<Layers className="mr-1.5 h-3.5 w-3.5" />
								{profile.name}
							</Button>
						))}
					</div>
				</div>
			)}

			<div className="relative">
				<Search className="pointer-events-none absolute top-1/2 left-2.5 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
				<Input
					type="search"
					value={query}
					onChange={(e) => setQuery(e.target.value)}
					placeholder={textGet("clinical.exam_orders.picker.search")}
					aria-label={textGet("clinical.exam_orders.picker.search")}
					className="pl-8"
				/>
			</div>

			<div className="max-h-80 space-y-4 overflow-y-auto rounded-md border p-3">
				{groups.length === 0 && (
					<p className="text-sm text-muted-foreground">
						{textGet("clinical.exam_orders.picker.empty")}
					</p>
				)}
				{groups.map((group) => (
					<section key={group.category} className="space-y-2">
						<h4 className="text-sm font-semibold">
							{textGet(EXAM_CATEGORY_KEYS[group.category])}
						</h4>
						{group.subgroups.map((sub) => (
							<div key={sub.subgroup || "-"} className="space-y-1 pl-2">
								{sub.subgroup && (
									<p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
										{sub.subgroup}
									</p>
								)}
								<ul className="grid gap-1 sm:grid-cols-2">
									{sub.items.map((item) => {
										const draft = selected.get(item.ID);
										const id = `exam-catalog-${item.ID}`;
										return (
											<li key={item.ID} className="flex items-center gap-2">
												<Checkbox
													id={id}
													checked={Boolean(draft)}
													disabled={draft?.locked}
													onCheckedChange={(checked) =>
														toggle(item, checked === true)
													}
												/>
												<label
													htmlFor={id}
													className="cursor-pointer text-sm leading-tight"
												>
													{item.name}
												</label>
											</li>
										);
									})}
								</ul>
							</div>
						))}
					</section>
				))}
			</div>

			<div className="space-y-2">
				<p className="text-sm font-medium">
					{textGet("clinical.exam_orders.picker.free_text")}
				</p>
				<div className="flex flex-col gap-2 sm:flex-row">
					<Input
						value={freeName}
						onChange={(e) => setFreeName(e.target.value)}
						onKeyDown={(e) => {
							if (e.key === "Enter") {
								e.preventDefault();
								addFree();
							}
						}}
						placeholder={textGet(
							"clinical.exam_orders.picker.free_text.placeholder",
						)}
						aria-label={textGet("clinical.exam_orders.picker.free_text")}
					/>
					<Select
						value={freeCategory}
						onValueChange={(v) =>
							setFreeCategory((v as ExamCategory) ?? "other")
						}
					>
						<SelectTrigger
							className="sm:w-44"
							aria-label={textGet("clinical.exam_orders.field.category")}
						>
							<SelectValue>
								{textGet(EXAM_CATEGORY_KEYS[freeCategory])}
							</SelectValue>
						</SelectTrigger>
						<SelectContent>
							{EXAM_CATEGORIES.map((category) => (
								<SelectItem key={category} value={category}>
									{textGet(EXAM_CATEGORY_KEYS[category])}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
					<Button
						type="button"
						variant="outline"
						disabled={!freeName.trim()}
						onClick={addFree}
					>
						<Plus className="mr-1.5 h-4 w-4" />
						{textGet("clinical.exam_orders.picker.free_text.add")}
					</Button>
				</div>
			</div>
		</div>
	);
}
