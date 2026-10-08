import { useText } from "@pengi/shared";
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetHeader,
	SheetTitle,
} from "@pengi/ui";
import type { AuditLog } from "@/api/audit-service";
import { FormattedDate } from "@/components/custom/formatted";
import { auditChanges, formatAuditValue } from "@/lib/audit";
import {
	ActionBadge,
	EntityTypeLabel,
} from "@/sections/columns/audit/audit-log-columns";

/** One audit row: who, when, which record, and the values before and after. */
export function AuditLogDetailSheet({
	log,
	onClose,
}: {
	/** The row to show; null closes the sheet. */
	log: AuditLog | null;
	onClose: () => void;
}) {
	const { textGet } = useText();
	const changes = log ? auditChanges(log) : [];
	const showBefore = log?.action === "UPDATE" || log?.action === "DELETE";
	const showAfter = log?.action === "UPDATE" || log?.action === "CREATE";

	return (
		<Sheet open={!!log} onOpenChange={(open) => !open && onClose()}>
			<SheetContent className="data-[side=right]:w-full data-[side=right]:sm:max-w-xl">
				{log && (
					<>
						<SheetHeader>
							<SheetTitle className="flex items-center gap-2">
								<ActionBadge action={log.action} />
								<span>
									<EntityTypeLabel type={log.entity_type} />{" "}
									<span className="text-muted-foreground">
										#{log.entity_id}
									</span>
								</span>
							</SheetTitle>
							<SheetDescription>
								<FormattedDate value={log.created_at} withTime /> ·{" "}
								{log.user_name || `#${log.user_id}`}
								{log.patient_name ? ` · ${log.patient_name}` : ""}
							</SheetDescription>
						</SheetHeader>

						<div className="px-4 pb-6">
							{log.action === "READ" ? (
								<p className="text-sm text-muted-foreground">
									{textGet("audit.detail.read")}
								</p>
							) : changes.length === 0 ? (
								<p className="text-sm text-muted-foreground">
									{textGet("audit.detail.no_changes")}
								</p>
							) : (
								<table className="w-full table-fixed text-sm">
									<thead>
										<tr className="border-b text-left text-xs text-muted-foreground">
											<th className="w-1/3 py-2 pr-3 font-medium">
												{textGet("audit.detail.field")}
											</th>
											{showBefore && (
												<th className="py-2 pr-3 font-medium">
													{textGet("audit.detail.before")}
												</th>
											)}
											{showAfter && (
												<th className="py-2 font-medium">
													{textGet("audit.detail.after")}
												</th>
											)}
										</tr>
									</thead>
									<tbody>
										{changes.map((c) => (
											<tr
												key={c.field}
												className="border-b align-top last:border-0"
											>
												<td className="py-2 pr-3 font-mono text-xs break-all">
													{c.field}
												</td>
												{showBefore && (
													<td className="py-2 pr-3 break-words text-muted-foreground">
														{formatAuditValue(c.before)}
													</td>
												)}
												{showAfter && (
													<td className="py-2 break-words">
														{formatAuditValue(c.after)}
													</td>
												)}
											</tr>
										))}
									</tbody>
								</table>
							)}
						</div>
					</>
				)}
			</SheetContent>
		</Sheet>
	);
}
