import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { Toaster } from "./components/ui/sonner.tsx";
import { TooltipProvider } from "./components/ui/tooltip.tsx";
import { LanguageProvider } from "./contexts/language-context.tsx";

const root = document.getElementById("root");

if (!root) {
	throw new Error("Root element not found");
}

createRoot(root).render(
	<StrictMode>
		<TooltipProvider>
			<LanguageProvider>
				<Toaster position="top-center" />
				<App />
			</LanguageProvider>
		</TooltipProvider>
	</StrictMode>,
);
