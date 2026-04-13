import { lazy } from "react";
import { createBrowserRouter, Outlet } from "react-router";
import CheckAuth from "@/contexts/check-auth";

const CompanyList = lazy(() => import("@/pages/companies/company-list"));
const CompanyUsers = lazy(() => import("@/pages/companies/company-users"));
const CreateCompany = lazy(() => import("@/pages/companies/create-company"));
const EditCompany = lazy(() => import("@/pages/companies/edit-company"));
const CreateFeature = lazy(() => import("@/pages/features/create-feature"));
const EditFeature = lazy(() => import("@/pages/features/edit-feature"));
const FeatureList = lazy(() => import("@/pages/features/feature-list"));
const Home = lazy(() => import("@/pages/home/home"));
const Login = lazy(() => import("@/pages/login/login-page"));
const CreatePlan = lazy(() => import("@/pages/plans/create-plan"));
const EditPlan = lazy(() => import("@/pages/plans/edit-plan"));
const PlanList = lazy(() => import("@/pages/plans/plan-list"));
const CreateRole = lazy(() => import("@/pages/roles/create-role"));
const EditRole = lazy(() => import("@/pages/roles/edit-role"));
const RoleList = lazy(() => import("@/pages/roles/role-list"));
const CreateSubscription = lazy(
	() => import("@/pages/subscriptions/create-subscription"),
);
const EditSubscription = lazy(
	() => import("@/pages/subscriptions/edit-subscription"),
);
const SubscriptionList = lazy(
	() => import("@/pages/subscriptions/subscription-list"),
);
const CreateUser = lazy(() => import("@/pages/users/create-user"));
const EditUser = lazy(() => import("@/pages/users/edit-user"));
const UserList = lazy(() => import("@/pages/users/user-list"));

const router = createBrowserRouter([
	{
		path: "/login",
		element: <Login />,
	},
	{
		element: (
			<CheckAuth>
				<Outlet />
			</CheckAuth>
		),
		children: [
			{ path: "/", element: <Home /> },
			{ path: "/companies", element: <CompanyList /> },
			{ path: "/companies/create", element: <CreateCompany /> },
			{ path: "/companies/edit/:id", element: <EditCompany /> },
			{ path: "/companies/:id/users", element: <CompanyUsers /> },
			{ path: "/features", element: <FeatureList /> },
			{ path: "/features/create", element: <CreateFeature /> },
			{ path: "/features/edit/:id", element: <EditFeature /> },
			{ path: "/plans", element: <PlanList /> },
			{ path: "/plans/create", element: <CreatePlan /> },
			{ path: "/plans/edit/:id", element: <EditPlan /> },
			{ path: "/subscriptions", element: <SubscriptionList /> },
			{ path: "/subscriptions/create", element: <CreateSubscription /> },
			{ path: "/subscriptions/edit/:id", element: <EditSubscription /> },
			{ path: "/roles", element: <RoleList /> },
			{ path: "/roles/create", element: <CreateRole /> },
			{ path: "/roles/edit/:id", element: <EditRole /> },
			{ path: "/users", element: <UserList /> },
			{ path: "/users/create", element: <CreateUser /> },
			{ path: "/users/edit/:id", element: <EditUser /> },
		],
	},
]);

export { router };
