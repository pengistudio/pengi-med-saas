import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import { AppTextBridge, initShared, LanguageProvider } from "@pengi/shared";
import { Toaster, TooltipProvider } from "@pengi/ui";
import App from "./App.tsx";
import { noAuthApi } from "./api";
import { session } from "./lib/session";

// Restore the session from the refresh cookie while the app renders.
session.restore();

initShared({ client: noAuthApi });

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
