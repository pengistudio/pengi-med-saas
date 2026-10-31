import { useMessages } from "@pengi/shared";
import { Suspense } from "react";
import { RouterProvider } from "react-router";
import { PwaInstallBanner } from "./components/custom/pwa-install-banner";
import { router } from "./routes/routes";

export function App() {
	useMessages();

	return (
		<Suspense fallback={null}>
			<RouterProvider router={router} />
			<PwaInstallBanner />
		</Suspense>
	);
}

export default App;
