import { createBrowserRouter, Outlet, type RouteObject } from "react-router";
import CheckPermission from "@/components/custom/check-permission";
import CheckAuth from "@/contexts/check-auth";
import { PERMISSIONS } from "@/lib/constants";
import CatalogItemList from "@/pages/billing/catalog-item-list";
import CreateCatalogItemPage from "@/pages/billing/create-catalog-item";
import CreateInvoicePage from "@/pages/billing/create-invoice";
import EditCatalogItemPage from "@/pages/billing/edit-catalog-item";
// Billing Module
import InvoiceListPage from "@/pages/billing/invoice-list";
import SriSettingsPage from "@/pages/billing/sri-settings";
import AppointmentsPage from "@/pages/clincal/appointments/appointments";
import CreateMedicalRecordPage from "@/pages/clincal/patient/create-medical-record";
import CreatePatientPage from "@/pages/clincal/patient/create-patient";
import EditPatientPage from "@/pages/clincal/patient/edit-patient";
import MedicalRecords from "@/pages/clincal/patient/medical-record-list";
import Clinical from "@/pages/clincal/patient/patient-list";
import UpdateMedicalRecordPage from "@/pages/clincal/patient/update-medical-record";
import ViewMedicalRecordPage from "@/pages/clincal/patient/view-medical-record";
import WaitingRoomPage from "@/pages/clincal/waiting-room/waiting-room";
import WaitingRoomDisplayPage from "@/pages/clincal/waiting-room/waiting-room-display";
import PairDisplayPage from "@/pages/display/pair";
import Home from "@/pages/home/home";
import LoginEnvironments from "@/pages/login/login-environments";
import Login from "@/pages/login/login-page";
import Profile from "@/pages/profile/profile";
import ResetPasswordPage from "@/pages/reset-password/reset-password-page";
import SettingsPage from "@/pages/settings/settings-page";
import Signup from "@/pages/signup/signup-page";
import TeamPage from "@/pages/team/team-page";

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
