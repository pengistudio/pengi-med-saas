import { useText } from "@pengi/shared";
import { downloadDocument, getDocuments } from "@/api/portal-service";
import { DownloadButton } from "@/components/download-button";
import { byYear, Ledger, LedgerRow, LedgerYear } from "@/components/ledger";
import { SectionState } from "@/components/section-state";
import { nameCase } from "@/lib/person-name";
import { useLoad } from "@/lib/use-load";

export function DocumentsSection() {
	const { textGet, formatDate } = useText();
	const state = useLoad(getDocuments);
	const thisYear = String(new Date().getFullYear());

	return (
		<SectionState
			state={state}
			emptyKey="portal.documents.empty"
			isEmpty={(documents) => documents.length === 0}
		>
			{(documents) => (
				<div>
					<p className="pb-2 text-sm text-muted-foreground">
						{textGet("portal.documents.hint")}
					</p>
					{byYear(documents).map(([year, items]) => (
						<div key={year}>
							{year !== thisYear && <LedgerYear year={year} />}
							<Ledger>
								{items.map((doc) => {
									const type = textGet(`portal.documents.type.${doc.type}`);
									return (
										<LedgerRow
											key={`${doc.type}-${doc.id}`}
											date={doc.date}
											action={
												<DownloadButton
													name={`${type}, ${formatDate(doc.date)}`}
													fileName={`${doc.type}-${doc.date}.pdf`}
													fetchFile={() => downloadDocument(doc.type, doc.id)}
												/>
											}
										>
											<p className="font-medium">{type}</p>
											{doc.doctor_name && (
												<p className="text-sm text-muted-foreground">
													{nameCase(doc.doctor_name)}
												</p>
											)}
										</LedgerRow>
									);
								})}
							</Ledger>
						</div>
					))}
				</div>
			)}
		</SectionState>
	);
}
