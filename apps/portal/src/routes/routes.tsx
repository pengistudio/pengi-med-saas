import { lazy } from "react";
import { createBrowserRouter } from "react-router";
import { HomePage } from "@/pages/home-page";
import { LandingPage } from "@/pages/message-page";
import { SignInPage } from "@/pages/sign-in-page";

const BookingPage = lazy(() => import("@/pages/booking-page"));
const NewPatientPage = lazy(() => import("@/pages/new-patient-page"));

export const router = createBrowserRouter([
	{ path: "/", element: <LandingPage /> },
	{ path: "/:slug/entrar", element: <SignInPage /> },
	{ path: "/:slug/citas/nueva", element: <BookingPage /> },
	{ path: "/:slug/primera-cita", element: <NewPatientPage /> },
	{ path: "/:slug", element: <HomePage /> },
]);
