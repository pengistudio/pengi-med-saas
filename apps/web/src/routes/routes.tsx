import { createBrowserRouter, Outlet, type RouteObject } from "react-router";
import CheckPermission from "@/components/custom/check-permission";
import CheckAuth from "@/contexts/check-auth";
import { PERMISSIONS } from "@/lib/constants";
import AppointmentsPage from "@/pages/clincal/appointments/appointments";
import CreateMedicalRecordPage from "@/pages/clincal/patient/create-medical-record";
import CreatePatientPage from "@/pages/clincal/patient/create-patient";
import EditPatientPage from "@/pages/clincal/patient/edit-patient";
import MedicalRecords from "@/pages/clincal/patient/medical-record-list";
import Clinical from "@/pages/clincal/patient/patient-list";
import UpdateMedicalRecordPage from "@/pages/clincal/patient/update-medical-record";
import ViewMedicalRecordPage from "@/pages/clincal/patient/view-medical-record";
import Home from "@/pages/home/home";
import LoginEnvironments from "@/pages/login/login-environments";
import Login from "@/pages/login/login-page";
import Profile from "@/pages/profile/profile";
import Signup from "@/pages/signup/signup-page";

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
	],
};

const routes: RouteObject[] = [clinicalRoutes];

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
			...routes,
		],
	},
]);

export { router };
