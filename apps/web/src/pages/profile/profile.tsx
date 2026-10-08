import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Form,
	FormInput,
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
	Text,
	useViewport,
} from "@pengi/ui";
import {
	Building2,
	CalendarClock,
	Mail,
	PenLine,
	Save,
	Shield,
	Stethoscope,
} from "lucide-react";
import React from "react";
import { useSearchParams } from "react-router";
import { z } from "zod";
import {
	getProfile,
	type ProfileData,
	updateProfile,
} from "@/api/user-service";
import { DoctorScheduleSection } from "@/components/features/agenda/doctor-schedule-section";
import { specialtyLabel } from "@/components/features/doctors/doctor-utils";
import { MyDoctorProfileCard } from "@/components/features/doctors/my-doctor-profile-card";
import { ElectronicSignatureCard } from "@/components/features/profile/electronic-signature-card";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { useDoctorStatus } from "@/store/doctors-store";
import { selectEnvironment, useSessionStore } from "@/store/session-store";

const profileSchema = z.object({
	email: z.email(),
	environment_name: z.string().min(1),
});

type Section = "account" | "doctor" | "schedule" | "signature";

function initials(name: string) {
	const words = name.trim().split(/\s+/).filter(Boolean);
	return (
		(words[0]?.[0] ?? "") + (words.length > 1 ? (words[1]?.[0] ?? "") : "")
	).toUpperCase();
}

/** A read-only fact of the account: label above, value below. */
function Fact({
	label,
	children,
}: {
	label: string;
	children: React.ReactNode;
}) {
	return (
		<div className="grid gap-0.5">
			<dt className="text-xs text-muted-foreground">{label}</dt>
			<dd className="text-sm font-medium">{children}</dd>
		</div>
	);
}

/**
 * Profile page. On a desktop the sections sit beside a vertical menu and
 * only one shows at a time, so nothing needs a long scroll; the open one is
 * kept in `?section=` so other screens can link straight to it. On a phone
 * every section is stacked in one column.
 */
const Profile = () => {
	const [profile, setProfile] = React.useState<ProfileData | null>(null);
	const [loading, setLoading] = React.useState(false);
	const { textGet } = useText();
	const environment = useSessionStore(selectEnvironment);
	const { checkPermission } = usePermission();
	const { isDesktop } = useViewport();
	const status = useDoctorStatus();
	const [searchParams, setSearchParams] = useSearchParams();

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

	const doctor = status?.doctor ?? null;
	const canCreateOwnDoctor =
		checkPermission([PERMISSIONS.DOCTORS.PERMISSION_MANAGE_DOCTORS]) ||
		checkPermission([
			PERMISSIONS.MEDICAL_RECORD.PERMISSION_CREATE_MEDICAL_RECORD,
		]);
	const canSign = checkPermission([
		PERMISSIONS.MEDICAL_RECORD.PERMISSION_SIGN_MEDICAL_DOCUMENT,
	]);

	const sections: { value: Section; label: string; icon: React.ReactNode }[] = [
		{
			value: "account",
			label: textGet("profile.section.account"),
			icon: <Building2 />,
		},
		...(doctor || (status && canCreateOwnDoctor)
			? [
					{
						value: "doctor" as const,
						label: textGet("profile.section.doctor"),
						icon: <Stethoscope />,
					},
				]
			: []),
		...(doctor
			? [
					{
						value: "schedule" as const,
						label: textGet("agenda.schedule.title"),
						icon: <CalendarClock />,
					},
				]
			: []),
		...(canSign
			? [
					{
						value: "signature" as const,
						label: textGet("signature.title"),
						icon: <PenLine />,
					},
				]
			: []),
	];

	const requested = searchParams.get("section");
	const active: Section =
		sections.find((s) => s.value === requested)?.value ?? "account";

	function openSection(value: unknown) {
		const next = new URLSearchParams(searchParams);
		if (value === "account") next.delete("section");
		else next.set("section", String(value));
		setSearchParams(next, { replace: true });
	}

	if (!profile) {
		return (
			<div className="flex items-center justify-center h-64">
				<p className="text-muted-foreground animate-pulse">
					{textGet("dashboard.loading")}
				</p>
			</div>
		);
	}

	const displayName = doctor?.full_name || profile.user_name;

	// Each section once; the layout below decides how they are shown.
	const content: Record<Section, React.ReactNode> = {
		account: (
			<Card>
				<CardHeader>
					<CardTitle>
						<Text uuid="profile.environment_info" />
					</CardTitle>
					<CardDescription>
						<Text uuid="profile.environment_info.description" />
					</CardDescription>
				</CardHeader>
				<CardContent className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_16rem]">
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
								<div className="flex justify-end">
									<Button type="submit" disabled={loading}>
										<Save className="mr-2 h-4 w-4" />
										{textGet("profile.save")}
									</Button>
								</div>
							</div>
						)}
					</Form>

					{/* Set by the clinic, not editable here. */}
					<dl className="grid content-start gap-4 border-t pt-6 lg:border-t-0 lg:border-l lg:pt-0 lg:pl-8">
						<Fact label={textGet("profile.username")}>{profile.user_name}</Fact>
						<Fact label={textGet("profile.role")}>
							<span className="capitalize">{profile.role}</span>
						</Fact>
						<Fact label={textGet("profile.legal_name")}>
							{profile.legal_name}
						</Fact>
						<Fact label={textGet("profile.trade_name")}>
							{profile.trade_name}
						</Fact>
					</dl>
				</CardContent>
			</Card>
		),
		doctor: <MyDoctorProfileCard />,
		schedule: <DoctorScheduleSection scope="me" />,
		signature: <ElectronicSignatureCard />,
	};

	return (
		<div className="mx-auto w-full max-w-6xl space-y-6">
			{/* Who you are in this clinic: the avatar carries the doctor's agenda color. */}
			<header className="flex items-center gap-4">
				<div
					aria-hidden
					className="flex size-14 shrink-0 items-center justify-center rounded-full bg-primary/10 text-lg font-semibold text-primary ring-2 ring-offset-2 ring-offset-background"
					style={
						doctor?.color
							? ({
									"--tw-ring-color": doctor.color,
									color: doctor.color,
									backgroundColor: `${doctor.color}1a`,
								} as React.CSSProperties)
							: undefined
					}
				>
					{initials(displayName)}
				</div>
				<div className="min-w-0">
					<h1 className="truncate text-xl font-semibold tracking-tight">
						{displayName}
					</h1>
					<p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
						{doctor && <span>{specialtyLabel(doctor, textGet)}</span>}
						<span className="inline-flex items-center gap-1">
							<Shield className="size-3.5 text-primary" />
							<span className="capitalize">{profile.role}</span>
						</span>
						{profile.trade_name && <span>{profile.trade_name}</span>}
					</p>
				</div>
			</header>

			{isDesktop ? (
				<Tabs
					value={active}
					onValueChange={openSection}
					orientation="vertical"
					className="items-start gap-6"
				>
					<TabsList
						variant="line"
						aria-label={textGet("profile.title")}
						className="sticky top-4 w-52 shrink-0 items-stretch border-r pr-3"
					>
						{sections.map((s) => (
							<TabsTrigger key={s.value} value={s.value} className="h-9 px-3">
								{s.icon}
								{s.label}
							</TabsTrigger>
						))}
					</TabsList>
					<div className="min-w-0 flex-1">
						{sections.map((s) => (
							<TabsContent key={s.value} value={s.value}>
								{content[s.value]}
							</TabsContent>
						))}
					</div>
				</Tabs>
			) : (
				// On a phone every section is stacked: one scroll, no menu.
				<div className="space-y-6">
					{sections.map((s) => (
						<React.Fragment key={s.value}>{content[s.value]}</React.Fragment>
					))}
				</div>
			)}
		</div>
	);
};

export default Profile;
