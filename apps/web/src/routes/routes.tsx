import { lazy } from "react";
import { createBrowserRouter, Outlet, type RouteObject } from "react-router";
import CheckPermission from "@/components/custom/check-permission";
import CheckAuth from "@/contexts/check-auth";
import { PERMISSIONS } from "@/lib/constants";

const CatalogItemList = lazy(() => import("@/pages/billing/catalog-item-list"));
const CreateCatalogItemPage = lazy(
	() => import("@/pages/billing/create-catalog-item"),
);
const CreateInvoicePage = lazy(() => import("@/pages/billing/create-invoice"));
const EditCatalogItemPage = lazy(
	() => import("@/pages/billing/edit-catalog-item"),
);
const InvoiceListPage = lazy(() => import("@/pages/billing/invoice-list"));
const SriSettingsPage = lazy(() => import("@/pages/billing/sri-settings"));
const AppointmentsPage = lazy(
	() => import("@/pages/clincal/appointments/appointments"),
);
const CreateMedicalRecordPage = lazy(
	() => import("@/pages/clincal/patient/create-medical-record"),
);
const CreatePatientPage = lazy(
	() => import("@/pages/clincal/patient/create-patient"),
);
const EditPatientPage = lazy(
	() => import("@/pages/clincal/patient/edit-patient"),
);
const MedicalRecords = lazy(
	() => import("@/pages/clincal/patient/medical-record-list"),
);
const Clinical = lazy(() => import("@/pages/clincal/patient/patient-list"));
const UpdateMedicalRecordPage = lazy(
	() => import("@/pages/clincal/patient/update-medical-record"),
);
const ViewMedicalRecordPage = lazy(
	() => import("@/pages/clincal/patient/view-medical-record"),
);
const WaitingRoomPage = lazy(
	() => import("@/pages/clincal/waiting-room/waiting-room"),
);
const WaitingRoomDisplayPage = lazy(
	() => import("@/pages/clincal/waiting-room/waiting-room-display"),
);
const PairDisplayPage = lazy(() => import("@/pages/display/pair"));
const Home = lazy(() => import("@/pages/home/home"));
const KanbanPage = lazy(() => import("@/pages/kanban/kanban-page"));
const LoginEnvironments = lazy(
	() => import("@/pages/login/login-environments"),
);
const Login = lazy(() => import("@/pages/login/login-page"));
const Profile = lazy(() => import("@/pages/profile/profile"));
const ResetPasswordPage = lazy(
	() => import("@/pages/reset-password/reset-password-page"),
);
const SettingsPage = lazy(() => import("@/pages/settings/settings-page"));
const Signup = lazy(() => import("@/pages/signup/signup-page"));
const TeamPage = lazy(() => import("@/pages/team/team-page"));

const clinicalRoutes: RouteObject = {
	path: "/clinical",
	element: (
		<CheckPermission
			permissions={[PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_PATIENT]}
		/>
	),
	children: [
		{ index: true, element: <Clinical /> },
		{
			path: "create",
			element: (
				<CheckPermission
					permissions={[PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_PATIENT]}
				>
					<CreatePatientPage />
				</CheckPermission>
			),
		},
		{
			path: "edit/:id",
			element: (
				<CheckPermission
					permissions={[PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_PATIENT]}
				>
					<EditPatientPage />
				</CheckPermission>
			),
		},
		{
			path: "medical-records/create",
			element: (
				<CheckPermission
					permissions={[
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
					]}
				>
					<CreateMedicalRecordPage />
				</CheckPermission>
			),
		},
		{
			path: "medical-records/:patientId",
			element: (
				<CheckPermission
					permissions={[
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_MEDICAL_RECORD,
					]}
				>
					<MedicalRecords />
				</CheckPermission>
			),
		},
		{
			path: "medical-records/view/:id",
			element: (
				<CheckPermission
					permissions={[
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_READ_MEDICAL_RECORD,
					]}
				>
					<ViewMedicalRecordPage />
				</CheckPermission>
			),
		},
		{
			path: "medical-records/update/:id",
			element: (
				<CheckPermission
					permissions={[
						PERMISSIONS.MEDICAL_RECORD.PERMISSION_UPDATE_MEDICAL_RECORD,
					]}
				>
					<UpdateMedicalRecordPage />
				</CheckPermission>
			),
		},
		{
			path: "appointments",
			element: <AppointmentsPage />,
		},
		{
			path: "waiting-room",
			element: <WaitingRoomPage />,
		},
	],
};

const billingRoutes: RouteObject = {
	path: "/billing",
	element: (
		<CheckPermission
			permissions={[PERMISSIONS.BILLING.PERMISSION_READ_BILLING]}
		/>
	),
	children: [
		{ index: true, element: <InvoiceListPage /> },
		{
			path: "create",
			element: (
				<CheckPermission
					permissions={[PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]}
				>
					<CreateInvoicePage />
				</CheckPermission>
			),
		},
		{
			path: "settings",
			element: (
				<CheckPermission
					permissions={[PERMISSIONS.BILLING.PERMISSION_MANAGE_SRI_SETTINGS]}
				>
					<SriSettingsPage />
				</CheckPermission>
			),
		},
		{
			path: "catalog-items",
			element: (
				<CheckPermission
					permissions={[PERMISSIONS.BILLING.PERMISSION_READ_BILLING]}
				>
					<CatalogItemList />
				</CheckPermission>
			),
		},
		{
			path: "catalog-items/create",
			element: (
				<CheckPermission
					permissions={[PERMISSIONS.BILLING.PERMISSION_CREATE_BILLING]}
				>
					<CreateCatalogItemPage />
				</CheckPermission>
			),
		},
		{
			path: "catalog-items/edit/:id",
			element: (
				<CheckPermission
					permissions={[PERMISSIONS.BILLING.PERMISSION_UPDATE_BILLING]}
				>
					<EditCatalogItemPage />
				</CheckPermission>
			),
		},
	],
};

const routes: RouteObject[] = [clinicalRoutes, billingRoutes];

const router = createBrowserRouter([
	{
		path: "/login",
		element: <Login />,
	},
	{
		path: "/login/environments",
		element: <LoginEnvironments />,
	},
	{
		path: "/signup",
		element: <Signup />,
	},
	{
		path: "/reset-password",
		element: <ResetPasswordPage />,
	},

	{
		element: (
			<CheckAuth>
				<Outlet />
			</CheckAuth>
		),
		children: [
			{
				path: "/",
				element: <Home />,
			},
			{
				path: "/profile",
				element: <Profile />,
			},
			{
				path: "/settings",
				element: <SettingsPage />,
			},
			{
				path: "/team",
				element: <TeamPage />,
			},
			{
				path: "/tasks",
				element: (
					<CheckPermission
						permissions={[PERMISSIONS.KANBAN.PERMISSION_READ_KANBAN]}
					>
						<KanbanPage />
					</CheckPermission>
				),
			},
			...routes,
		],
	},
	{
		path: "/display/waiting-room",
		element: <WaitingRoomDisplayPage />,
	},
	{
		path: "/display/pair",
		element: <PairDisplayPage />,
	},
]);

export { router };
