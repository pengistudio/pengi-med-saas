import { createBrowserRouter } from "react-router";
import { HomePage } from "@/pages/home-page";
import { LandingPage } from "@/pages/message-page";
import { SignInPage } from "@/pages/sign-in-page";

export const router = createBrowserRouter([
	{ path: "/", element: <LandingPage /> },
	{ path: "/:slug/entrar", element: <SignInPage /> },
	{ path: "/:slug", element: <HomePage /> },
]);
