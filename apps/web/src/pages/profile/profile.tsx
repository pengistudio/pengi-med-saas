import { Building2, Mail, Save, Shield, User } from "lucide-react";
import React from "react";
import { z } from "zod";
import {
	getProfile,
	type ProfileData,
	updateProfile,
} from "@/api/user-service";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useSessionStore } from "@/store/session-store";

const profileSchema = z.object({
	email: z.email(),
	environment_name: z.string().min(1),
});

const Profile = () => {
	const [profile, setProfile] = React.useState<ProfileData | null>(null);
	const [loading, setLoading] = React.useState(false);
	const { textGet } = useText();
	const { environment } = useSessionStore();

	React.useEffect(() => {
		if (!environment?.id) return;
		getProfile(environment.id).then((res) => {
			if (res.success && res.data) {
				setProfile(res.data as ProfileData);
			}
		});
	}, [environment?.id]);

	async function onSubmit(values: z.infer<typeof profileSchema>) {
		if (!environment?.id) return;
		setLoading(true);
		const res = await updateProfile(environment.id, {
			email: values.email,
			environment_name: values.environment_name,
		});
		if (res.success && res.data) {
			setProfile(res.data as ProfileData);
		}
		setLoading(false);
	}

	if (!profile) {
		return (
			<DashboardLayout>
				<div className="flex items-center justify-center h-64">
					<p className="text-muted-foreground animate-pulse">
						{textGet("dashboard.loading")}
					</p>
				</div>
			</DashboardLayout>
		);
	}

	return (
		<DashboardLayout>
			<div className="max-w-2xl mx-auto space-y-6">
				<h1 className="text-2xl font-bold tracking-tight">
					<Text uuid="profile.title" />
				</h1>

				{/* User Info — read only */}
				<Card>
					<CardHeader>
						<CardTitle className="flex items-center gap-2">
							<User className="h-5 w-5" />
							<Text uuid="profile.user_info" />
						</CardTitle>
						<CardDescription>
							<Text uuid="profile.user_info.description" />
						</CardDescription>
					</CardHeader>
					<CardContent className="space-y-4">
						<div className="grid grid-cols-2 gap-4">
							<div>
								<p className="text-sm font-medium text-muted-foreground">
									{textGet("profile.username")}
								</p>
								<p className="text-sm font-semibold mt-1">
									{profile.user_name}
								</p>
							</div>
							<div>
								<p className="text-sm font-medium text-muted-foreground">
									{textGet("profile.role")}
								</p>
								<div className="flex items-center gap-1.5 mt-1">
									<Shield className="h-4 w-4 text-primary" />
									<p className="text-sm font-semibold capitalize">
										{profile.role}
									</p>
								</div>
							</div>
						</div>
					</CardContent>
				</Card>

				{/* Editable Form */}
				<Card>
					<CardHeader>
						<CardTitle className="flex items-center gap-2">
							<Building2 className="h-5 w-5" />
							<Text uuid="profile.environment_info" />
						</CardTitle>
						<CardDescription>
							<Text uuid="profile.environment_info.description" />
						</CardDescription>
					</CardHeader>
					<CardContent>
						<Form
							schema={profileSchema}
							defaultValues={{
								email: profile.email,
								environment_name: profile.environment_name,
							}}
							onSubmit={onSubmit}
						>
							{(field) => (
								<div className="space-y-4">
									<FormInput
										field={field}
										name="email"
										label={textGet("profile.email")}
										type="email"
										startAddon={<Mail className="h-4 w-4" />}
									/>
									<FormInput
										field={field}
										name="environment_name"
										label={textGet("profile.environment_name")}
									/>

									{/* Read-only company info */}
									<div className="grid grid-cols-2 gap-4 pt-2 border-t">
										<div>
											<p className="text-sm font-medium text-muted-foreground">
												{textGet("profile.legal_name")}
											</p>
											<p className="text-sm font-semibold mt-1">
												{profile.legal_name}
											</p>
										</div>
										<div>
											<p className="text-sm font-medium text-muted-foreground">
												{textGet("profile.trade_name")}
											</p>
											<p className="text-sm font-semibold mt-1">
												{profile.trade_name}
											</p>
										</div>
									</div>

									<div className="flex justify-end pt-2">
										<Button type="submit" disabled={loading}>
											<Save className="mr-2 h-4 w-4" />
											{textGet("profile.save")}
										</Button>
									</div>
								</div>
							)}
						</Form>
					</CardContent>
				</Card>
			</div>
		</DashboardLayout>
	);
};

export default Profile;
