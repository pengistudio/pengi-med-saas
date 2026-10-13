import { useText } from "@pengi/shared";
import { Button, Spinner } from "@pengi/ui";
import { ArrowLeft } from "lucide-react";
import React from "react";
import {
	Navigate,
	useNavigate,
	useParams,
	useSearchParams,
} from "react-router";
import { getPatientById, type Patient } from "@/api/clinical-service";
import {
	createExamOrder,
	type ExamCatalogItem,
	type ExamOrder,
	type ExamProfile,
	getExamCatalog,
	getExamOrder,
	getExamProfiles,
	updateExamOrder,
} from "@/api/exam-order-service";
import { PageHeader } from "@/components/custom/page-header";
import { RegisterDoctorNotice } from "@/components/features/doctors/doctor-notices";
import {
	hasAnyResult,
	isOrderVoided,
	patientDisplayName,
	toItemInputs,
} from "@/lib/exam-orders";
import ExamOrderForm, {
	type ExamOrderFormValues,
} from "@/sections/forms/clinical/exam-order-form";

/** Everything the editor needs: the active catalog and profiles, the order (edit) and its patient. */
async function loadEditorData(orderId?: number, patientId?: number) {
	const [catalogRes, profilesRes, orderRes] = await Promise.all([
		getExamCatalog({ active: true }),
		getExamProfiles({ active: true }),
		orderId ? getExamOrder(orderId) : Promise.resolve(null),
	]);
	const order: ExamOrder | null = orderRes?.success
		? (orderRes.data ?? null)
		: null;
	let patient: Patient | null = order?.patient ?? null;
	const pid = order?.patient_id ?? patientId;
	if (!patient && pid) {
		const patientRes = await getPatientById(pid);
		patient = patientRes.success ? (patientRes.data ?? null) : null;
	}
	return {
		catalog: catalogRes.success ? (catalogRes.data ?? []) : [],
		profiles: profilesRes.success ? (profilesRes.data ?? []) : [],
		order,
		patient,
	};
}

/**
 * Create (`/clinical/exam-orders/new?patient_id=&medical_record_id=`) or edit
 * (`/clinical/exam-orders/:id/edit`) an exam order.
 */
export default function ExamOrderEditorPage() {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { id } = useParams<{ id: string }>();
	const [searchParams] = useSearchParams();
	const orderId = id ? Number(id) : undefined;
	const queryPatientId = Number(searchParams.get("patient_id")) || undefined;
	const queryRecordId =
		Number(searchParams.get("medical_record_id")) || undefined;

	const [catalog, setCatalog] = React.useState<ExamCatalogItem[]>([]);
	const [profiles, setProfiles] = React.useState<ExamProfile[]>([]);
	const [order, setOrder] = React.useState<ExamOrder | null>(null);
	const [patient, setPatient] = React.useState<Patient | null>(null);
	const [loading, setLoading] = React.useState(true);
	const [saving, setSaving] = React.useState(false);

	React.useEffect(() => {
		let cancelled = false;
		loadEditorData(orderId, queryPatientId).then((data) => {
			if (cancelled) return;
			setCatalog(data.catalog);
			setProfiles(data.profiles);
			setOrder(data.order);
			setPatient(data.patient);
			setLoading(false);
		});
		return () => {
			cancelled = true;
		};
	}, [orderId, queryPatientId]);

	if (!orderId && !queryPatientId) return <Navigate to="/clinical" replace />;

	if (loading) {
		return (
			<div className="flex h-64 items-center justify-center">
				<Spinner />
			</div>
		);
	}

	if (orderId && (!order || isOrderVoided(order))) {
		return <Navigate to={`/clinical/exam-orders/${orderId}`} replace />;
	}

	/** Creates or updates the order; resolves the saved order's id. */
	async function saveOrder(values: ExamOrderFormValues) {
		const items = toItemInputs(values.items);
		if (order) {
			// With results, the header can't change: send it as it is.
			const header = hasAnyResult(order)
				? {
						diagnoses: order.diagnoses ?? [],
						priority: order.priority,
						destination_lab: order.destination_lab,
						notes: order.notes,
					}
				: {
						diagnoses: values.diagnoses,
						priority: values.priority,
						destination_lab: values.destination_lab?.trim() ?? "",
						notes: values.notes?.trim() ?? "",
						doctor_id: values.doctor_id ?? undefined,
					};
			const res = await updateExamOrder(order.ID, {
				...header,
				medical_record_id: order.medical_record_id,
				items,
			});
			return res.success ? order.ID : undefined;
		}
		if (!queryPatientId) return undefined;
		const res = await createExamOrder({
			patient_id: queryPatientId,
			medical_record_id: queryRecordId ?? null,
			diagnoses: values.diagnoses,
			priority: values.priority,
			destination_lab: values.destination_lab?.trim() ?? "",
			notes: values.notes?.trim() ?? "",
			doctor_id: values.doctor_id ?? undefined,
			items,
		});
		return res.success ? res.data?.ID : undefined;
	}

	async function handleSubmit(values: ExamOrderFormValues) {
		setSaving(true);
		try {
			const savedId = await saveOrder(values);
			if (savedId) navigate(`/clinical/exam-orders/${savedId}`);
		} finally {
			setSaving(false);
		}
	}

	const name = patientDisplayName(patient);

	return (
		<main className="grid grid-cols-1 items-start gap-4">
			<PageHeader
				title={
					order
						? textGet("clinical.exam_orders.edit.title", { code: order.code })
						: textGet("clinical.exam_orders.create.title")
				}
				description={
					name ? textGet("clinical.exam_orders.patient", { name }) : undefined
				}
				actions={
					<Button variant="outline" onClick={() => navigate(-1)}>
						<ArrowLeft className="mr-2 h-4 w-4" />
						{textGet("clinical.exam_orders.back")}
					</Button>
				}
			/>
			<RegisterDoctorNotice />
			<ExamOrderForm
				catalog={catalog}
				profiles={profiles}
				order={order ?? undefined}
				loading={saving}
				patientDoctorId={patient?.doctor_id}
				onSubmit={handleSubmit}
				onCancel={() => navigate(-1)}
			/>
		</main>
	);
}
