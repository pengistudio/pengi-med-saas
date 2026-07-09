import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Notification } from "@/api/notification-service";

// Base UI's Menu positioning relies on APIs jsdom doesn't implement.
beforeEach(() => {
	if (!Element.prototype.hasPointerCapture) {
		Element.prototype.hasPointerCapture = () => false;
	}
	if (!Element.prototype.setPointerCapture) {
		Element.prototype.setPointerCapture = () => {};
	}
	if (!Element.prototype.releasePointerCapture) {
		Element.prototype.releasePointerCapture = () => {};
	}
	if (!Element.prototype.scrollIntoView) {
		Element.prototype.scrollIntoView = () => {};
	}
});

vi.mock("@/hooks/use-text", () => ({
	useText: () => ({
		textGet: (key: string) => key,
	}),
}));

const mockMarkNotificationAsRead = vi.fn();
const mockMarkAllNotificationsAsRead = vi.fn();
vi.mock("@/api/notification-service", () => ({
	markNotificationAsRead: (...args: unknown[]) =>
		mockMarkNotificationAsRead(...args),
	markAllNotificationsAsRead: (...args: unknown[]) =>
		mockMarkAllNotificationsAsRead(...args),
}));

const mockNavigate = vi.fn();
vi.mock("react-router", () => ({
	useNavigate: () => mockNavigate,
}));

const mockMarkReadLocally = vi.fn();
const mockMarkAllReadLocally = vi.fn();
let storeState: {
	notifications: Notification[];
	unreadCount: number;
};

vi.mock("@/store/notification-store", () => ({
	useNotificationStore: (
		selector: (state: {
			notifications: Notification[];
			unreadCount: number;
			markReadLocally: (id: number) => void;
			markAllReadLocally: () => void;
		}) => unknown,
	) =>
		selector({
			...storeState,
			markReadLocally: mockMarkReadLocally,
			markAllReadLocally: mockMarkAllReadLocally,
		}),
}));

vi.mock("@/store/message-store", () => ({
	useMessageStore: (selector: (state: { lang: string }) => unknown) =>
		selector({ lang: "es" }),
}));

import NotificationBell from "@/components/custom/notification-bell";

function buildNotification(
	overrides: Partial<Notification> = {},
): Notification {
	return {
		ID: 1,
		CreatedAt: new Date().toISOString(),
		UpdatedAt: new Date().toISOString(),
		tenant_id: 1,
		user_id: 1,
		type: "clinical.draft.stale",
		resource_type: "medical_record_draft",
		resource_id: 1,
		message_key: "notification.clinical.draft.stale",
		params: { patient_name: "Juan Perez", minutes_elapsed: "75" },
		action_url: "/clinical/medical-records/1",
		read_at: null,
		...overrides,
	};
}

describe("NotificationBell", () => {
	beforeEach(() => {
		mockMarkNotificationAsRead.mockClear();
		mockMarkAllNotificationsAsRead.mockClear();
		mockMarkReadLocally.mockClear();
		mockMarkAllReadLocally.mockClear();
		mockNavigate.mockClear();
		storeState = { notifications: [], unreadCount: 0 };
	});

	it("does not show a badge when there are no unread notifications", () => {
		render(<NotificationBell />);
		expect(screen.queryByText("0")).toBeNull();
	});

	it("shows the unread count badge", () => {
		storeState = { notifications: [buildNotification()], unreadCount: 3 };
		render(<NotificationBell />);
		expect(screen.getByText("3")).toBeDefined();
	});

	it("caps the badge display at 9+", () => {
		storeState = { notifications: [buildNotification()], unreadCount: 42 };
		render(<NotificationBell />);
		expect(screen.getByText("9+")).toBeDefined();
	});

	it("shows the empty state when there are no notifications", async () => {
		render(<NotificationBell />);
		fireEvent.click(screen.getByRole("button"));
		await waitFor(() => {
			expect(screen.getByText("notification.bell.empty")).toBeDefined();
		});
	});

	it("marks a notification as read and navigates when clicked", async () => {
		storeState = {
			notifications: [buildNotification()],
			unreadCount: 1,
		};
		render(<NotificationBell />);
		fireEvent.click(screen.getByRole("button"));

		const item = await screen.findByText("notification.clinical.draft.stale");
		fireEvent.click(item);

		expect(mockMarkReadLocally).toHaveBeenCalledWith(1);
		expect(mockMarkNotificationAsRead).toHaveBeenCalledWith(1);
		expect(mockNavigate).toHaveBeenCalledWith("/clinical/medical-records/1");
	});

	it("marks all notifications as read", async () => {
		storeState = {
			notifications: [buildNotification()],
			unreadCount: 1,
		};
		render(<NotificationBell />);
		fireEvent.click(screen.getByRole("button"));

		const markAllButton = await screen.findByText(
			"notification.bell.mark_all_read",
		);
		fireEvent.click(markAllButton);

		expect(mockMarkAllReadLocally).toHaveBeenCalled();
		expect(mockMarkAllNotificationsAsRead).toHaveBeenCalled();
	});
});
