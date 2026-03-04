import { createBrowserRouter, Outlet } from "react-router";
import CheckAuth from "@/contexts/check-auth";
import CompanyList from "@/pages/companies/company-list";
import CreateCompany from "@/pages/companies/create-company";
import EditCompany from "@/pages/companies/edit-company";
import CreateFeature from "@/pages/features/create-feature";
import EditFeature from "@/pages/features/edit-feature";
import FeatureList from "@/pages/features/feature-list";
import Home from "@/pages/home/home";
import Login from "@/pages/login/login-page";
import CreatePlan from "@/pages/plans/create-plan";
import EditPlan from "@/pages/plans/edit-plan";
import PlanList from "@/pages/plans/plan-list";
import CreateSubscription from "@/pages/subscriptions/create-subscription";
import EditSubscription from "@/pages/subscriptions/edit-subscription";
import SubscriptionList from "@/pages/subscriptions/subscription-list";
import CreateUser from "@/pages/users/create-user";
import EditUser from "@/pages/users/edit-user";
import UserList from "@/pages/users/user-list";

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
			{ path: "/features", element: <FeatureList /> },
			{ path: "/features/create", element: <CreateFeature /> },
			{ path: "/features/edit/:id", element: <EditFeature /> },
			{ path: "/plans", element: <PlanList /> },
			{ path: "/plans/create", element: <CreatePlan /> },
			{ path: "/plans/edit/:id", element: <EditPlan /> },
			{ path: "/subscriptions", element: <SubscriptionList /> },
			{ path: "/subscriptions/create", element: <CreateSubscription /> },
			{ path: "/subscriptions/edit/:id", element: <EditSubscription /> },
			{ path: "/users", element: <UserList /> },
			{ path: "/users/create", element: <CreateUser /> },
			{ path: "/users/edit/:id", element: <EditUser /> },
		],
	},
]);

export { router };
