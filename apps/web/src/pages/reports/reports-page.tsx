import { useText } from "@pengi/shared";
import {
	Button,
	Input,
	Label,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
} from "@pengi/ui";
import { Download, FileText } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { type Doctor, getDoctors } from "@/api/doctors-service";
import {
	downloadReport,
	REPORT_KINDS,
	type ReportFilters,
	type ReportKind,
} from "@/api/reports-service";
import { PageHeader } from "@/components/custom/page-header";
import { AppointmentsTab } from "./appointments-tab";
import { ClinicalTab } from "./clinical-tab";
import { ReceivablesTab } from "./receivables-tab";
import {
	currentMonthRange,
	reportFilename,
	saveBlob,
	validateRange,
} from "./report-utils";
import { RevenueTab } from "./revenue-tab";

const ALL_DOCTORS = "all";

/** Revenue, receivables, appointments and clinical statistics of the clinic. */
const ReportsPage = () => {
	const { textGet } = useText();
	const [kind, setKind] = useState<ReportKind>("revenue");
	const [range, setRange] = useState(() => currentMonthRange());
	const [doctorId, setDoctorId] = useState<string>(ALL_DOCTORS);
	const [doctors, setDoctors] = useState<Doctor[]>([]);
	const [downloading, setDownloading] = useState<"csv" | "pdf" | null>(null);

	useEffect(() => {
		let cancelled = false;
		getDoctors({ notifyError: false }).then((res) => {
			if (!cancelled && res.success && res.data) setDoctors(res.data);
		});
		return () => {
			cancelled = true;
		};
	}, []);

	const rangeError = validateRange(range.from, range.to);
	const filters: ReportFilters = useMemo(
		() => ({
			from: range.from,
			to: range.to,
			doctor_id: doctorId === ALL_DOCTORS ? undefined : Number(doctorId),
		}),
		[range.from, range.to, doctorId],
	);

	const handleDownload = async (format: "csv" | "pdf") => {
		setDownloading(format);
		const res = await downloadReport(kind, format, filters);
		setDownloading(null);
		if (res.success && res.data) {
			saveBlob(res.data, reportFilename(kind, filters, format));
		}
	};

	const doctorName =
		doctors.find((d) => String(d.ID) === doctorId)?.full_name ?? "";

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={textGet("reports.page.title")}
				description={textGet("reports.page.description")}
				actions={
					<div className="flex gap-2">
						<Button
							variant="outline"
							disabled={!!rangeError || downloading !== null}
							onClick={() => handleDownload("csv")}
						>
							<Download className="h-4 w-4" />
							{textGet("reports.download.csv")}
						</Button>
						<Button
							variant="outline"
							disabled={!!rangeError || downloading !== null}
							onClick={() => handleDownload("pdf")}
						>
							<FileText className="h-4 w-4" />
							{textGet("reports.download.pdf")}
						</Button>
					</div>
				}
			/>

			<div className="flex flex-wrap items-end gap-4">
				<div className="grid gap-1.5">
					<Label htmlFor="reports-from">{textGet("reports.filter.from")}</Label>
					<Input
						id="reports-from"
						type="date"
						className="w-40"
						value={range.from}
						max={range.to || undefined}
						onChange={(e) => setRange((r) => ({ ...r, from: e.target.value }))}
					/>
				</div>
				<div className="grid gap-1.5">
					<Label htmlFor="reports-to">{textGet("reports.filter.to")}</Label>
					<Input
						id="reports-to"
						type="date"
						className="w-40"
						value={range.to}
						min={range.from || undefined}
						onChange={(e) => setRange((r) => ({ ...r, to: e.target.value }))}
					/>
				</div>
				<div className="grid gap-1.5">
					<Label>{textGet("reports.filter.doctor")}</Label>
					<Select
						value={doctorId}
						onValueChange={(v) => setDoctorId(v ?? ALL_DOCTORS)}
					>
						<SelectTrigger
							className="w-56"
							aria-label={textGet("reports.filter.doctor")}
						>
							<SelectValue>
								{doctorId === ALL_DOCTORS
									? textGet("reports.filter.all_doctors")
									: doctorName}
							</SelectValue>
						</SelectTrigger>
						<SelectContent>
							<SelectItem value={ALL_DOCTORS}>
								{textGet("reports.filter.all_doctors")}
							</SelectItem>
							{doctors.map((d) => (
								<SelectItem key={d.ID} value={String(d.ID)}>
									{d.full_name}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>
			</div>
			{rangeError && (
				<p role="alert" className="text-sm text-destructive">
					{textGet(`reports.filter.error.${rangeError}`)}
				</p>
			)}

			<Tabs value={kind} onValueChange={(v) => setKind(v as ReportKind)}>
				<TabsList>
					{REPORT_KINDS.map((k) => (
						<TabsTrigger key={k} value={k}>
							{textGet(`reports.tab.${k}`)}
						</TabsTrigger>
					))}
				</TabsList>
				{!rangeError && (
					<>
						<TabsContent value="revenue" className="mt-4">
							<RevenueTab filters={filters} />
						</TabsContent>
						<TabsContent value="receivables" className="mt-4">
							<ReceivablesTab filters={filters} />
						</TabsContent>
						<TabsContent value="appointments" className="mt-4">
							<AppointmentsTab filters={filters} />
						</TabsContent>
						<TabsContent value="clinical" className="mt-4">
							<ClinicalTab filters={filters} />
						</TabsContent>
					</>
				)}
			</Tabs>
		</main>
	);
};

export default ReportsPage;
