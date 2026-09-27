import { describe, expect, it } from "vitest";
import { createNavItems } from "@/config/nav-config";
import { getPageTitle } from "./page-title";

const textGet = (key: string) => key;
const navItems = createNavItems(textGet);
const title = (pathname: string) => getPageTitle(navItems, pathname, textGet);

describe("getPageTitle", () => {
	it("matches top-level and nested nav items", () => {
		expect(title("/")).toBe("dashboard.title");
		expect(title("/tasks")).toBe("tasks.title");
		expect(title("/clinical/waiting-room")).toBe(
			"dashboard.clinical.waiting_room",
		);
		expect(title("/subscription")).toBe("subscription.nav.title");
		expect(title("/settings")).toBe("settings.title");
		expect(title("/billing")).toBe("dashboard.billing.invoices");
	});

	it("prefers the longest matching prefix", () => {
		expect(title("/billing/settings")).toBe("dashboard.billing.settings");
		expect(title("/billing/create")).toBe("dashboard.billing.invoices");
		expect(title("/clinical/medical-records/12")).toBe(
			"dashboard.clinical.patients",
		);
	});

	it("covers pages outside the sidebar and leaves unknown ones to the caller", () => {
		expect(title("/profile")).toBe("profile.title");
		expect(title("/unknown")).toBeUndefined();
		expect(title("/billingx")).toBeUndefined();
	});
});
