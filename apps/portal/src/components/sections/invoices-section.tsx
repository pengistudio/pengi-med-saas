import { useText } from "@pengi/shared";
import {
	downloadInvoiceRide,
	downloadInvoiceXml,
	getInvoices,
} from "@/api/portal-service";
import { DownloadButton } from "@/components/download-button";
import { byYear, Ledger, LedgerRow, LedgerYear } from "@/components/ledger";
import { SectionState } from "@/components/section-state";
import { useLoad } from "@/lib/use-load";

export function InvoicesSection() {
	const { textGet, formatMoney } = useText();
	const state = useLoad(getInvoices);
	const thisYear = String(new Date().getFullYear());

	return (
		<SectionState
			state={state}
			emptyKey="portal.invoices.empty"
			isEmpty={(invoices) => invoices.length === 0}
		>
			{(invoices) => (
				<div>
					<p className="pb-2 text-sm text-muted-foreground">
						{textGet("portal.invoices.hint")}
					</p>
					{byYear(invoices).map(([year, items]) => (
						<div key={year}>
							{year !== thisYear && <LedgerYear year={year} />}
							<Ledger>
								{items.map((invoice) => (
									<LedgerRow key={invoice.id} date={invoice.date}>
										<div className="flex items-baseline justify-between gap-3">
											<p className="font-medium tabular-nums">
												{invoice.number}
											</p>
											<p className="font-semibold tabular-nums">
												{formatMoney(invoice.total)}
											</p>
										</div>
										<div className="flex gap-2 pt-2">
											<DownloadButton
												name={invoice.number}
												label={textGet("portal.invoices.ride")}
												fileName={`${invoice.number}.pdf`}
												fetchFile={() => downloadInvoiceRide(invoice.id)}
											/>
											<DownloadButton
												name={invoice.number}
												label={textGet("portal.invoices.xml")}
												fileName={`${invoice.number}.xml`}
												fetchFile={() => downloadInvoiceXml(invoice.id)}
											/>
										</div>
									</LedgerRow>
								))}
							</Ledger>
						</div>
					))}
				</div>
			)}
		</SectionState>
	);
}
