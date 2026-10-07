import { Button, Text } from "@pengi/ui";
import React from "react";
import {
	getWhatsAppMessages,
	type WhatsAppMessage,
	type WhatsAppMessageKind,
} from "@/api/whatsapp-service";
import { DataTable } from "@/components/custom/table/data-table";
import { whatsappMessageColumns } from "@/sections/columns/whatsapp/whatsapp-message-columns";

/** The DataTable pages by 20 rows. */
const LOG_PAGE_SIZE = 20;

/** Outbound kinds the "Envíos" tab filters by ("" = all). */
const KIND_FILTERS: ("" | Exclude<WhatsAppMessageKind, "inbound">)[] = [
	"",
	"reminder",
	"template",
	"reply",
	"test",
];

/** The "Envíos" tab: every message the clinic sent (reminders, templates, replies, tests). */
export function SentMessages() {
	const [messages, setMessages] = React.useState<WhatsAppMessage[]>([]);
	const [page, setPage] = React.useState(1);
	const [kind, setKind] = React.useState<(typeof KIND_FILTERS)[number]>("");
	const [totalPages, setTotalPages] = React.useState(1);
	const [loading, setLoading] = React.useState(true);

	function changePage(next: number) {
		setLoading(true);
		setPage(next);
	}

	function changeKind(next: (typeof KIND_FILTERS)[number]) {
		setLoading(true);
		setKind(next);
		setPage(1);
	}

	React.useEffect(() => {
		let cancelled = false;
		getWhatsAppMessages({
			page,
			limit: LOG_PAGE_SIZE,
			kind: kind || undefined,
		}).then((res) => {
			if (cancelled) return;
			setMessages(res.success ? (res.data?.items ?? []) : []);
			setTotalPages(res.success ? res.data?.total_pages || 1 : 1);
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [page, kind]);

	return (
		<div className="grid gap-3">
			<div className="flex flex-wrap gap-2">
				{KIND_FILTERS.map((k) => (
					<Button
						key={k || "all"}
						size="sm"
						variant={kind === k ? "default" : "outline"}
						onClick={() => changeKind(k)}
					>
						<Text uuid={`whatsapp.inbox.sent.filter.${k || "all"}`} />
					</Button>
				))}
			</div>
			<DataTable
				columns={whatsappMessageColumns}
				data={messages}
				loading={loading}
				page={page}
				pageCount={totalPages}
				onPageChange={changePage}
				emptyState={
					<p className="py-6 text-center text-sm text-muted-foreground">
						<Text uuid="settings.whatsapp.log.empty" />
					</p>
				}
			/>
		</div>
	);
}
