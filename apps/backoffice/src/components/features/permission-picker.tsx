import type { ServiceResponse } from "@pengi/shared";
import { useText } from "@pengi/shared";
import { Button, Checkbox, Label } from "@pengi/ui";
import React from "react";
import { getPermissions, type Permission } from "@/api/permission-service";

const UNCATEGORISED = "General";

/**
 * Picks permission IDs from the backoffice permission catalog, grouped by
 * category, with select/clear all (globally and per category).
 */
export function PermissionPicker({
	value,
	onChange,
	load = getPermissions,
}: {
	value: string[];
	onChange: (ids: string[]) => void;
	/** For tests: where the permission catalog comes from. */
	load?: () => Promise<ServiceResponse<Permission[]>>;
}) {
	const { textGet } = useText();
	const [permissions, setPermissions] = React.useState<Permission[]>([]);

	React.useEffect(() => {
		load().then((res) => {
			if (res.success) setPermissions(res.data ?? []);
		});
	}, [load]);

	const groups = React.useMemo(() => {
		const byCategory = new Map<string, Permission[]>();
		for (const p of permissions) {
			const category = p.category || UNCATEGORISED;
			byCategory.set(category, [...(byCategory.get(category) ?? []), p]);
		}
		return [...byCategory].sort(([a], [b]) => a.localeCompare(b));
	}, [permissions]);

	if (permissions.length === 0) return null;

	const selected = new Set(value);
	const allSelected = (perms: Permission[]) =>
		perms.every((p) => selected.has(p.ID));
	const setAll = (perms: Permission[], on: boolean) => {
		const next = new Set(selected);
		for (const p of perms) {
			if (on) next.add(p.ID);
			else next.delete(p.ID);
		}
		onChange([...next]);
	};
	const toggle = (id: string) =>
		setAll([{ ID: id } as Permission], !selected.has(id));

	const selectAllButton = (perms: Permission[], className: string) => (
		<Button
			type="button"
			variant="ghost"
			size="sm"
			className={className}
			onClick={() => setAll(perms, !allSelected(perms))}
		>
			{allSelected(perms)
				? textGet("backoffice.linking.deselect_all")
				: textGet("backoffice.linking.select_all")}
		</Button>
	);

	return (
		<div className="space-y-3">
			<div className="flex items-center justify-between">
				<Label>{textGet("backoffice.features.col.permissions")}</Label>
				{selectAllButton(permissions, "text-xs h-7")}
			</div>
			<div className="max-h-72 overflow-y-auto border rounded-md p-3 space-y-4">
				{groups.map(([category, perms]) => (
					<section key={category}>
						<div className="flex items-center justify-between mb-2 pb-1 border-b">
							<span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
								{category}
							</span>
							{selectAllButton(perms, "text-xs h-6 px-2")}
						</div>
						<div className="grid grid-cols-1 gap-1">
							{perms.map((p) => (
								// biome-ignore lint/a11y/noLabelWithoutControl: the base-ui Checkbox inside is the control; clicking the name toggles it (tested)
								<label
									key={p.ID}
									className="flex items-center gap-2 cursor-pointer hover:bg-muted/50 rounded px-2 py-1.5 transition-colors"
								>
									<Checkbox
										checked={selected.has(p.ID)}
										onCheckedChange={() => toggle(p.ID)}
									/>
									<span className="flex flex-col">
										<span className="text-sm font-medium">{p.name}</span>
										<span className="text-xs text-muted-foreground">
											{p.ID}
										</span>
									</span>
								</label>
							))}
						</div>
					</section>
				))}
			</div>
			<p className="text-xs text-muted-foreground">
				{value.length} / {permissions.length}{" "}
				{textGet("backoffice.linking.selected")}
			</p>
		</div>
	);
}
