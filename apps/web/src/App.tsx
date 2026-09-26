import { useMessages } from "@pengi/shared";
import { Suspense } from "react";
import { RouterProvider } from "react-router";
import { router } from "./routes/routes";

export function App() {
	useMessages(__APP_VERSION__);

	return (
		<Suspense fallback={null}>
			<RouterProvider router={router} />
		</Suspense>
	);
}

export default App;
