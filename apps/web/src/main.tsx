import "./sync-store";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import "./index.css";
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
