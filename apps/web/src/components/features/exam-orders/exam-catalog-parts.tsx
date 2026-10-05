import { useText } from "@pengi/shared";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import type React from "react";
import { EXAM_CATEGORIES, type ExamCategory } from "@/api/exam-order-service";
import { EXAM_CATEGORY_KEYS } from "@/lib/exam-orders";

/** Value of the filters meaning "no filter". */
export const ALL_FILTER = "all";

interface CatalogFiltersProps {
	category: ExamCategory | typeof ALL_FILTER;
	onCategoryChange: (category: ExamCategory | typeof ALL_FILTER) => void;
	subgroup: string;
	onSubgroupChange: (subgroup: string) => void;
	/** The subgroups of the chosen category. */
	subgroups: string[];
}

/** Category and subgroup filters of the catalog list. */
export function CatalogFilters({
	category,
	onCategoryChange,
	subgroup,
	onSubgroupChange,
	subgroups,
}: CatalogFiltersProps) {
	const { textGet } = useText();
	const allCategories = textGet("clinical.exam_catalog.filter.all_categories");
	const allSubgroups = textGet("clinical.exam_catalog.filter.all_subgroups");
	return (
		<>
			<Select
				value={category}
				onValueChange={(value) =>
					onCategoryChange((value as ExamCategory | null) ?? ALL_FILTER)
				}
			>
				<SelectTrigger
					className="w-40"
					aria-label={textGet("clinical.exam_orders.field.category")}
				>
					<SelectValue>
						{category === ALL_FILTER
							? allCategories
							: textGet(EXAM_CATEGORY_KEYS[category])}
					</SelectValue>
				</SelectTrigger>
				<SelectContent>
					<SelectItem value={ALL_FILTER}>{allCategories}</SelectItem>
					{EXAM_CATEGORIES.map((value) => (
						<SelectItem key={value} value={value}>
							{textGet(EXAM_CATEGORY_KEYS[value])}
						</SelectItem>
					))}
				</SelectContent>
			</Select>
			<Select
				value={subgroup}
				onValueChange={(value) =>
					onSubgroupChange((value as string) ?? ALL_FILTER)
				}
			>
				<SelectTrigger
					className="w-48"
					aria-label={textGet("clinical.exam_catalog.field.subgroup")}
				>
					<SelectValue>
						{subgroup === ALL_FILTER ? allSubgroups : subgroup}
					</SelectValue>
				</SelectTrigger>
				<SelectContent>
					<SelectItem value={ALL_FILTER}>{allSubgroups}</SelectItem>
					{subgroups.map((value) => (
						<SelectItem key={value} value={value}>
							{value}
						</SelectItem>
					))}
				</SelectContent>
			</Select>
		</>
	);
}

/** A dialog holding a create/edit form; it can't close while saving. */
export function FormDialog({
	open,
	busy,
	title,
	onClose,
	className,
	children,
}: {
	open: boolean;
	busy: boolean;
	title: React.ReactNode;
	onClose: () => void;
	className?: string;
	children: React.ReactNode;
}) {
	return (
		<Dialog
			open={open}
			onOpenChange={(next) => {
				if (!next && !busy) onClose();
			}}
		>
			<DialogContent className={className ?? "sm:max-w-md"}>
				<DialogHeader>
					<DialogTitle>{title}</DialogTitle>
				</DialogHeader>
				{children}
			</DialogContent>
		</Dialog>
	);
}
