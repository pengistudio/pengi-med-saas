import { useText } from "@pengi/shared";
import { downloadResultFile, getResults } from "@/api/portal-service";
import { DownloadButton } from "@/components/download-button";
import { byYear, Ledger, LedgerRow, LedgerYear } from "@/components/ledger";
import { SectionState } from "@/components/section-state";
import { nameCase } from "@/lib/person-name";
import { useLoad } from "@/lib/use-load";

export function ResultsSection() {
	const { textGet } = useText();
	const state = useLoad(getResults);
	const thisYear = String(new Date().getFullYear());

	return (
		<SectionState
			state={state}
			emptyKey="portal.results.empty"
			isEmpty={(orders) => orders.length === 0}
		>
			{(orders) => (
				<div>
					<p className="pb-2 text-sm text-muted-foreground">
						{textGet("portal.results.hint")}
					</p>
					{byYear(orders).map(([year, items]) => (
						<div key={year}>
							{year !== thisYear && <LedgerYear year={year} />}
							<Ledger>
								{items.map((order) => (
									<LedgerRow key={order.id} date={order.date}>
										<p className="font-medium">
											{textGet("portal.results.order", {
												number: order.number,
											})}
										</p>
										{order.doctor_name && (
											<p className="text-sm text-muted-foreground">
												{nameCase(order.doctor_name)}
											</p>
										)}
										<ul className="mt-3 space-y-3">
											{order.items.map((item) => (
												<li key={item.id} className="space-y-1.5">
													<p className="text-sm">{item.name}</p>
													<div className="flex flex-wrap gap-2">
														{item.files.map((file) => (
															<DownloadButton
																key={file.id}
																name={`${item.name}, ${file.file_name}`}
																label={file.file_name}
																fileName={file.file_name}
																fetchFile={() => downloadResultFile(file.id)}
															/>
														))}
													</div>
												</li>
											))}
										</ul>
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
