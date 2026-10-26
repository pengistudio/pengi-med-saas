import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import { AppTextBridge, initShared, LanguageProvider } from "@pengi/shared";
import { Toaster, TooltipProvider } from "@pengi/ui";
import App from "./App.tsx";
import { rootApi } from "./api/http-clients";

initShared({ client: rootApi });

const root = document.getElementById("root");

if (!root) {
	throw new Error("Root element not found");
}

createRoot(root).render(
	<StrictMode>
		<TooltipProvider>
			<LanguageProvider>
				<AppTextBridge>
					<Toaster position="top-center" />
					<App />
				</AppTextBridge>
			</LanguageProvider>
		</TooltipProvider>
	</StrictMode>,
);
