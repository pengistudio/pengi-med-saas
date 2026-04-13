import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import "./index.css";

// Reload on stale chunk errors (happens when a new deploy invalidates old chunk hashes)
window.addEventListener("unhandledrejection", (event) => {
	const msg: string = event.reason?.message ?? "";
	if (msg.includes("Failed to fetch dynamically imported module")) {
		window.location.reload();
	}
});

import { TooltipProvider } from "@/components/ui/tooltip";
import App from "./App.tsx";
import { Toaster } from "./components/ui/sonner";
import { LanguageProvider } from "./contexts/language-context.tsx";

// biome-ignore lint/style/noNonNullAssertion: root element is always present
createRoot(document.getElementById("root")!).render(
	<StrictMode>
		<TooltipProvider>
			<LanguageProvider>
				<Toaster position="top-center" />
				<App />
			</LanguageProvider>
		</TooltipProvider>
	</StrictMode>,
);
