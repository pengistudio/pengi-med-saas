import { useText } from "@pengi/shared";
import type React from "react";
import { DashboardLayout } from "@/sections/template/dashboard-template";

/** The create/edit page frame: dashboard layout, and a loading state until the item arrives. */
export function ResourceEditPage({
	loading = false,
	children,
}: {
	loading?: boolean;
	children: React.ReactNode;
}) {
	const { textGet } = useText();
	return (
		<DashboardLayout>
			{loading ? (
				<div className="flex items-center justify-center h-64">
					<p className="text-muted-foreground animate-pulse">
						{textGet("backoffice.common.loading")}
					</p>
				</div>
			) : (
				children
			)}
		</DashboardLayout>
	);
}
