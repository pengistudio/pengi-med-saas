import { useText } from "@pengi/shared";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Spinner,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@pengi/ui";
import { Pencil, Plus, Trash2 } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { type Resource, resourceRoutes } from "./resource";
import { useResourceList } from "./use-resource";

export interface ResourceColumn<T> {
	/** i18n key of the column header. */
	header: string;
	cell: (item: T) => React.ReactNode;
	className?: string;
}

interface ResourceListProps<T extends { ID: number }> {
	resource: Resource<T, never, never>;
	columns: ResourceColumn<T>[];
	/** How an item is named in the delete confirmation. */
	itemLabel: (item: T) => string;
	/** Extra per-row actions, shown before edit and delete. */
	rowActions?: (item: T) => React.ReactNode;
	/** Extra page actions, shown before the create button. */
	headerActions?: React.ReactNode;
}

/**
 * The list page of a resource: title, create button, loading and empty states,
 * one row per item with edit and delete. Deleting always asks for confirmation.
 */
export function ResourceList<T extends { ID: number }>({
	resource,
	columns,
	itemLabel,
	rowActions,
	headerActions,
}: ResourceListProps<T>) {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { items, loading, remove } = useResourceList(resource);
	const [deleting, setDeleting] = React.useState<T | null>(null);
	const [removing, setRemoving] = React.useState(false);
	const routes = resourceRoutes(resource.name);
	const key = (suffix: string) => `backoffice.${resource.name}.${suffix}`;

	const confirmDelete = async () => {
		if (!deleting) return;
		setRemoving(true);
		await remove(deleting.ID);
		setRemoving(false);
		setDeleting(null);
	};

	return (
		<>
			<div className="space-y-6">
				<div className="flex items-center justify-between">
					<h1 className="text-2xl font-bold tracking-tight">
						{textGet(key("title"))}
					</h1>
					<div className="flex items-center gap-2">
						{headerActions}
						<Button onClick={() => navigate(routes.create)}>
							<Plus className="h-4 w-4 mr-2" />
							{textGet(key("create"))}
						</Button>
					</div>
				</div>
				<Card>
					<CardHeader>
						<CardTitle>{textGet(key("list.title"))}</CardTitle>
						<CardDescription>
							{textGet(key("list.description"))}
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loading ? (
							<p className="text-sm text-muted-foreground py-8 text-center animate-pulse">
								{textGet("backoffice.common.loading")}
							</p>
						) : items.length === 0 ? (
							<p className="text-sm text-muted-foreground py-8 text-center">
								{textGet(key("empty"))}
							</p>
						) : (
							<Table>
								<TableHeader>
									<TableRow>
										{columns.map((col) => (
											<TableHead key={col.header}>
												{textGet(col.header)}
											</TableHead>
										))}
										<TableHead className="text-right">
											{textGet("table.actions")}
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{items.map((item) => (
										<TableRow key={item.ID}>
											{columns.map((col) => (
												<TableCell key={col.header} className={col.className}>
													{col.cell(item)}
												</TableCell>
											))}
											<TableCell className="text-right">
												<div className="flex items-center justify-end gap-2">
													{rowActions?.(item)}
													<Button
														variant="ghost"
														size="icon"
														aria-label={textGet("backoffice.common.edit")}
														onClick={() => navigate(routes.edit(item.ID))}
													>
														<Pencil className="h-4 w-4" />
													</Button>
													<Button
														variant="ghost"
														size="icon"
														aria-label={textGet("backoffice.common.delete")}
														onClick={() => setDeleting(item)}
													>
														<Trash2 className="h-4 w-4 text-destructive" />
													</Button>
												</div>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>

			<AlertDialog
				open={deleting !== null}
				onOpenChange={(open) => {
					if (!open && !removing) setDeleting(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{textGet("backoffice.common.delete.title")}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{textGet("backoffice.common.delete.description")}{" "}
							<strong>{deleting ? itemLabel(deleting) : ""}</strong>
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel disabled={removing}>
							{textGet("backoffice.common.cancel")}
						</AlertDialogCancel>
						<AlertDialogAction
							onClick={confirmDelete}
							disabled={removing}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							{removing && <Spinner />}
							{textGet("backoffice.common.delete.confirm")}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}
