import {
	Check,
	Copy,
	Link2,
	Loader2,
	ShieldAlert,
	UserPlus,
	Users,
} from "lucide-react";
import React from "react";
import {
	generateInviteLink,
	getTeamMembers,
	getTeamRoles,
	type TeamMember,
	type TeamRole,
} from "@/api/team-service";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useSessionStore } from "@/store/session-store";

const ROLE_COLORS: Record<string, string> = {
	admin: "bg-primary/10 text-primary border-primary/20",
	doctor: "bg-emerald-500/10 text-emerald-600 border-emerald-500/20",
	user: "bg-muted text-muted-foreground border-border",
};

function getRoleColor(role: string) {
	return ROLE_COLORS[role.toLowerCase()] ?? ROLE_COLORS.user;
}

function getInitials(name: string) {
	return name
		.split(" ")
		.map((n) => n[0])
		.join("")
		.toUpperCase()
		.slice(0, 2);
}

function MemberCard({ member }: { member: TeamMember }) {
	const displayName = member.environment_name || member.user_name;
	return (
		<Card className="hover:shadow-md transition-shadow duration-200">
			<CardContent className="pt-6 pb-5 flex flex-col items-center text-center gap-3">
				<Avatar className="h-14 w-14 ring-2 ring-border ring-offset-2 ring-offset-card">
					<AvatarFallback className="text-base font-semibold bg-primary/10 text-primary">
						{getInitials(displayName)}
					</AvatarFallback>
				</Avatar>

				<div className="space-y-0.5 min-w-0 w-full">
					<p className="font-semibold text-sm truncate">{displayName}</p>
					<p className="text-xs text-muted-foreground truncate">
						@{member.user_name}
					</p>
					{member.email && (
						<p className="text-xs text-muted-foreground truncate">
							{member.email}
						</p>
					)}
				</div>

				<Badge
					variant="outline"
					className={cn("text-xs capitalize", getRoleColor(member.role_name))}
				>
					{member.role_name}
				</Badge>
			</CardContent>
		</Card>
	);
}

function SkeletonCard() {
	return (
		<Card>
			<CardContent className="pt-6 pb-5 flex flex-col items-center gap-3">
				<div className="h-14 w-14 rounded-full bg-muted animate-pulse" />
				<div className="space-y-2 w-full">
					<div className="h-3 bg-muted rounded animate-pulse mx-auto w-24" />
					<div className="h-3 bg-muted rounded animate-pulse mx-auto w-16" />
				</div>
				<div className="h-5 w-14 bg-muted rounded-full animate-pulse" />
			</CardContent>
		</Card>
	);
}

const TeamPage = () => {
	const { textGet } = useText();
	const { environment } = useSessionStore();
	const isAdmin = environment?.role === "admin";

	const [members, setMembers] = React.useState<TeamMember[]>([]);
	const [roles, setRoles] = React.useState<TeamRole[]>([]);
	const [loading, setLoading] = React.useState(true);

	// Invite flow state
	const [roleDialogOpen, setRoleDialogOpen] = React.useState(false);
	const [selectedRole, setSelectedRole] = React.useState<TeamRole | null>(null);
	const [inviteLink, setInviteLink] = React.useState("");
	const [linkDialogOpen, setLinkDialogOpen] = React.useState(false);
	const [generating, setGenerating] = React.useState(false);
	const [copied, setCopied] = React.useState(false);

	React.useEffect(() => {
		Promise.all([
			getTeamMembers(),
			isAdmin ? getTeamRoles() : Promise.resolve({ success: true, data: [] }),
		]).then(([membersRes, rolesRes]) => {
			if (membersRes.success && membersRes.data)
				setMembers(membersRes.data as TeamMember[]);
			if (rolesRes.success && rolesRes.data)
				setRoles(rolesRes.data as TeamRole[]);
			setLoading(false);
		});
	}, [isAdmin]);

	const openRoleSelector = () => {
		setSelectedRole(null);
		setRoleDialogOpen(true);
	};

	const handleGenerateLink = async () => {
		if (!selectedRole) return;
		setGenerating(true);
		const res = await generateInviteLink(selectedRole.ID);
		setGenerating(false);
		if (res.success && res.data) {
			const token = (res.data as { token: string }).token;
			setInviteLink(`${window.location.origin}/signup?token=${token}`);
			setRoleDialogOpen(false);
			setLinkDialogOpen(true);
		}
	};

	const handleCopy = async () => {
		await navigator.clipboard.writeText(inviteLink);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<DashboardLayout>
			<div className="space-y-6">
				{/* Header */}
				<div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
					<div>
						<h1 className="text-2xl font-bold tracking-tight">
							{textGet("team.title")}
						</h1>
						<p className="text-sm text-muted-foreground mt-1">
							{textGet("team.description")}
						</p>
					</div>
					{isAdmin && (
						<Button
							onClick={openRoleSelector}
							className="shrink-0 self-start sm:self-auto"
						>
							<UserPlus className="h-4 w-4 mr-2" />
							{textGet("team.invite")}
						</Button>
					)}
				</div>

				{/* Stats banner */}
				<div className="flex items-center gap-3 px-4 py-3 rounded-xl border bg-muted/30">
					<div className="h-9 w-9 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
						<Users className="h-5 w-5 text-primary" />
					</div>
					<div>
						<p className="text-xs text-muted-foreground">
							{textGet("team.members.title")}
						</p>
						<p className="text-lg font-bold leading-tight">
							{loading ? "—" : members.length}{" "}
							<span className="text-sm font-normal text-muted-foreground">
								{textGet("team.members.count")}
							</span>
						</p>
					</div>
				</div>

				{/* Members grid */}
				{loading ? (
					<div className="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
						{[1, 2, 3, 4].map((i) => (
							<SkeletonCard key={i} />
						))}
					</div>
				) : members.length === 0 ? (
					<div className="flex flex-col items-center justify-center py-20 gap-3">
						<div className="h-14 w-14 rounded-full bg-muted flex items-center justify-center">
							<Users className="h-7 w-7 text-muted-foreground" />
						</div>
						<p className="text-sm text-muted-foreground">
							{textGet("team.members.empty")}
						</p>
						{isAdmin && (
							<Button variant="outline" size="sm" onClick={openRoleSelector}>
								<UserPlus className="h-4 w-4 mr-2" />
								{textGet("team.invite")}
							</Button>
						)}
					</div>
				) : (
					<div className="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
						{members.map((member) => (
							<MemberCard key={member.environment_id} member={member} />
						))}
					</div>
				)}
			</div>

			{/* Step 1: Role selector dialog */}
			<Dialog open={roleDialogOpen} onOpenChange={setRoleDialogOpen}>
				<DialogContent className="max-w-sm">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2">
							<div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center">
								<UserPlus className="h-4 w-4 text-primary" />
							</div>
							{textGet("team.role_dialog.title")}
						</DialogTitle>
						<DialogDescription>
							{textGet("team.role_dialog.description")}
						</DialogDescription>
					</DialogHeader>

					<div className="space-y-2">
						<Label>{textGet("team.role_dialog.role_label")}</Label>
						<Select
							value={selectedRole?.role ?? ""}
							onValueChange={(val) => {
								setSelectedRole(roles.find((r) => r.role === val) ?? null);
							}}
						>
							<SelectTrigger className="w-full capitalize">
								<SelectValue
									placeholder={textGet("team.role_dialog.role_placeholder")}
								/>
							</SelectTrigger>
							<SelectContent>
								{roles.map((r) => (
									<SelectItem key={r.ID} value={r.role} className="capitalize">
										{r.role}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					<DialogFooter className="gap-2">
						<Button variant="outline" onClick={() => setRoleDialogOpen(false)}>
							{textGet("team.invite_dialog.close")}
						</Button>
						<Button
							onClick={handleGenerateLink}
							disabled={!selectedRole || generating}
						>
							{generating ? (
								<Loader2 className="h-4 w-4 mr-2 animate-spin" />
							) : (
								<Link2 className="h-4 w-4 mr-2" />
							)}
							{textGet("team.role_dialog.generate")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Step 2: Invite link dialog */}
			<Dialog open={linkDialogOpen} onOpenChange={setLinkDialogOpen}>
				<DialogContent className="max-w-md">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2">
							<div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center">
								<Link2 className="h-4 w-4 text-primary" />
							</div>
							{textGet("team.invite_dialog.title")}
						</DialogTitle>
						<DialogDescription>
							{selectedRole && (
								<span>
									{textGet("team.invite_dialog.role_for")}{" "}
									<strong className="capitalize">{selectedRole.role}</strong>.{" "}
								</span>
							)}
							{textGet("team.invite_dialog.description")}
						</DialogDescription>
					</DialogHeader>

					<div className="space-y-3">
						<div className="flex gap-2">
							<Input
								readOnly
								value={inviteLink}
								className="font-mono text-xs bg-muted/50"
								onClick={(e) => (e.target as HTMLInputElement).select()}
							/>
							<Button
								size="icon"
								variant={copied ? "default" : "outline"}
								onClick={handleCopy}
								className="shrink-0 transition-all"
							>
								{copied ? (
									<Check className="h-4 w-4" />
								) : (
									<Copy className="h-4 w-4" />
								)}
							</Button>
						</div>
						{copied && (
							<p className="text-xs text-center text-emerald-600 font-medium">
								{textGet("team.invite_dialog.copied")}
							</p>
						)}
						<div className="flex items-start gap-2 rounded-lg bg-amber-500/10 border border-amber-500/20 px-3 py-2">
							<ShieldAlert className="h-4 w-4 text-amber-600 shrink-0 mt-0.5" />
							<p className="text-xs text-amber-700">
								{textGet("team.invite_dialog.warning")}
							</p>
						</div>
					</div>

					<DialogFooter>
						<Button
							variant="outline"
							className="w-full"
							onClick={() => setLinkDialogOpen(false)}
						>
							{textGet("team.invite_dialog.close")}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</DashboardLayout>
	);
};

export default TeamPage;
