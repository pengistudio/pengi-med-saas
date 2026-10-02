import { useText } from "@pengi/shared";
import {
	Avatar,
	AvatarFallback,
	Badge,
	Button,
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
	Input,
	InputGroup,
	InputGroupAddon,
	InputGroupButton,
	InputGroupInput,
	Label,
	RadioGroup,
	RadioGroupItem,
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
	Skeleton,
} from "@pengi/ui";
import {
	Check,
	Copy,
	Link2,
	Loader2,
	Lock,
	Search,
	ShieldAlert,
	UserPlus,
	Users,
	X,
} from "lucide-react";
import React from "react";
import {
	generateInviteLink,
	getTeamMembers,
	getTeamRoles,
	type TeamMember,
	type TeamRole,
	updateTeamMemberRole,
} from "@/api/team-service";
import { PageHeader } from "@/components/custom/page-header";
import usePermission from "@/hooks/use-permission";
import { PERMISSIONS } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { useSessionStore } from "@/store/session-store";

// Fixed role catalog (apps/api/features/users/data/role-data.go), in the
// order the filters and the invite dialog list them.
const CANONICAL_ROLES = ["admin", "doctor", "recepcionista", "contador"];

// One color per role: the avatar, the role pill and the filter dot share it,
// so the roster reads by role at a glance.
const ROLE_STYLES: Record<
	string,
	{ pill: string; avatar: string; dot: string }
> = {
	admin: {
		pill: "bg-primary/10 text-primary border-primary/20",
		avatar: "bg-primary/10 text-primary",
		dot: "bg-primary",
	},
	doctor: {
		pill: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-500/20",
		avatar: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
		dot: "bg-emerald-500",
	},
	recepcionista: {
		pill: "bg-sky-500/10 text-sky-700 dark:text-sky-400 border-sky-500/20",
		avatar: "bg-sky-500/10 text-sky-700 dark:text-sky-400",
		dot: "bg-sky-500",
	},
	contador: {
		pill: "bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-500/20",
		avatar: "bg-amber-500/10 text-amber-700 dark:text-amber-400",
		dot: "bg-amber-500",
	},
	user: {
		pill: "bg-muted text-muted-foreground border-border",
		avatar: "bg-muted text-muted-foreground",
		dot: "bg-muted-foreground",
	},
};

const SEARCH_THRESHOLD = 6;

function getRoleStyle(role: string) {
	return ROLE_STYLES[role.toLowerCase()] ?? ROLE_STYLES.user;
}

function getInitials(name: string) {
	return name
		.split(" ")
		.map((n) => n[0])
		.join("")
		.toUpperCase()
		.slice(0, 2);
}

function useRoleLabel() {
	const { textGet } = useText();
	return React.useCallback(
		(role: string) => {
			const key = role.toLowerCase();
			return CANONICAL_ROLES.includes(key)
				? textGet(`team.role_name.${key}`)
				: role;
		},
		[textGet],
	);
}

function MemberRow({
	member,
	isSelf,
	canManageTeam,
	roles,
	updating,
	onRoleChange,
}: {
	member: TeamMember;
	isSelf: boolean;
	canManageTeam: boolean;
	roles: TeamRole[];
	updating: boolean;
	onRoleChange: (member: TeamMember, role: TeamRole) => void;
}) {
	const { textGet } = useText();
	const roleLabel = useRoleLabel();
	const displayName = member.environment_name || member.user_name;
	const style = getRoleStyle(member.role_name);
	const editable = canManageTeam && !isSelf;

	return (
		<li className="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 py-3 sm:flex-nowrap sm:px-5">
			<Avatar className="h-10 w-10 shrink-0">
				<AvatarFallback className={cn("text-sm font-semibold", style.avatar)}>
					{getInitials(displayName)}
				</AvatarFallback>
			</Avatar>

			<div className="min-w-0 flex-1">
				<p className="flex items-center gap-2 text-sm font-medium">
					<span className="truncate">{displayName}</span>
					{isSelf && (
						<Badge variant="secondary" className="h-5 px-1.5 text-[11px]">
							{textGet("team.you")}
						</Badge>
					)}
				</p>
				<p className="flex min-w-0 flex-wrap gap-x-3 text-xs text-muted-foreground">
					<span className="truncate">@{member.user_name}</span>
					{member.email && (
						<a
							href={`mailto:${member.email}`}
							className="truncate hover:text-foreground hover:underline underline-offset-2"
						>
							{member.email}
						</a>
					)}
				</p>
			</div>

			<div className="ml-14 shrink-0 sm:ml-0">
				{editable ? (
					<Select
						value={String(member.role_id)}
						disabled={updating}
						onValueChange={(val) => {
							const role = roles.find((r) => String(r.ID) === val);
							if (role && role.ID !== member.role_id)
								onRoleChange(member, role);
						}}
					>
						<SelectTrigger
							size="sm"
							aria-label={textGet("team.role.change_label", {
								name: displayName,
							})}
							className={cn(
								"h-7 w-auto min-w-0 gap-1.5 rounded-full border px-3 text-xs [&_svg]:size-3",
								style.pill,
							)}
						>
							{updating && <Loader2 className="animate-spin" />}
							<SelectValue>{roleLabel(member.role_name)}</SelectValue>
						</SelectTrigger>
						<SelectContent>
							{roles.map((r) => (
								<SelectItem key={r.ID} value={String(r.ID)}>
									{roleLabel(r.role)}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				) : (
					<Badge
						variant="outline"
						title={
							canManageTeam && isSelf
								? textGet("team.role.own_locked")
								: undefined
						}
						className={cn("h-7 gap-1.5 rounded-full px-3 text-xs", style.pill)}
					>
						{canManageTeam && isSelf && <Lock className="size-3" />}
						{roleLabel(member.role_name)}
					</Badge>
				)}
			</div>
		</li>
	);
}

function SkeletonRow() {
	return (
		<li className="flex items-center gap-4 px-4 py-3 sm:px-5">
			<Skeleton className="h-10 w-10 rounded-full" />
			<div className="flex-1 space-y-2">
				<Skeleton className="h-3 w-40" />
				<Skeleton className="h-3 w-56" />
			</div>
			<Skeleton className="h-7 w-24 rounded-full" />
		</li>
	);
}

const TeamPage = () => {
	const { textGet } = useText();
	const roleLabel = useRoleLabel();
	const { checkPermission } = usePermission();
	const canManageTeam = checkPermission([
		PERMISSIONS.TEAM.PERMISSION_MANAGE_TEAM_MEMBERS,
	]);
	const currentEnvironmentId = useSessionStore((s) => s.environment?.id);

	const [members, setMembers] = React.useState<TeamMember[]>([]);
	const [roles, setRoles] = React.useState<TeamRole[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [updatingEnvironmentId, setUpdatingEnvironmentId] = React.useState<
		number | null
	>(null);
	const [roleFilter, setRoleFilter] = React.useState<string | null>(null);
	const [query, setQuery] = React.useState("");

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
			canManageTeam
				? getTeamRoles()
				: Promise.resolve({ success: true, data: [] }),
		]).then(([membersRes, rolesRes]) => {
			if (membersRes.success && membersRes.data)
				setMembers(membersRes.data as TeamMember[]);
			if (rolesRes.success && rolesRes.data)
				setRoles(rolesRes.data as TeamRole[]);
			setLoading(false);
		});
	}, [canManageTeam]);

	// Roles present in the team, catalog order first, with their head count.
	const roleCounts = React.useMemo(() => {
		const counts = new Map<string, number>();
		for (const m of members) {
			const key = m.role_name.toLowerCase();
			counts.set(key, (counts.get(key) ?? 0) + 1);
		}
		const rank = (r: string) => {
			const i = CANONICAL_ROLES.indexOf(r);
			return i === -1 ? CANONICAL_ROLES.length : i;
		};
		return [...counts.entries()].sort(([a], [b]) => rank(a) - rank(b));
	}, [members]);

	const visibleMembers = React.useMemo(() => {
		const q = query.trim().toLowerCase();
		return members
			.filter((m) => !roleFilter || m.role_name.toLowerCase() === roleFilter)
			.filter(
				(m) =>
					!q ||
					[m.environment_name, m.user_name, m.email].some((v) =>
						v?.toLowerCase().includes(q),
					),
			)
			.sort((a, b) => {
				if (a.environment_id === currentEnvironmentId) return -1;
				if (b.environment_id === currentEnvironmentId) return 1;
				return (a.environment_name || a.user_name).localeCompare(
					b.environment_name || b.user_name,
				);
			});
	}, [members, roleFilter, query, currentEnvironmentId]);

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
			setCopied(false);
			setRoleDialogOpen(false);
			setLinkDialogOpen(true);
		}
	};

	const handleCopy = async () => {
		await navigator.clipboard.writeText(inviteLink);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	// Optimistic: the pill changes at once and reverts if the API refuses.
	const handleRoleChange = async (member: TeamMember, role: TeamRole) => {
		const patch = (role_id: number, role_name: string) =>
			setMembers((prev) =>
				prev.map((m) =>
					m.environment_id === member.environment_id
						? { ...m, role_id, role_name }
						: m,
				),
			);
		patch(role.ID, role.role);
		setUpdatingEnvironmentId(member.environment_id);
		const res = await updateTeamMemberRole(member.environment_id, role.ID);
		if (!res.success) patch(member.role_id, member.role_name);
		setUpdatingEnvironmentId(null);
	};

	const inviteRoles = React.useMemo(
		() =>
			[...roles].sort(
				(a, b) =>
					CANONICAL_ROLES.indexOf(a.role.toLowerCase()) -
					CANONICAL_ROLES.indexOf(b.role.toLowerCase()),
			),
		[roles],
	);

	const showFilters = roleCounts.length > 1;
	const showSearch = members.length > SEARCH_THRESHOLD;

	return (
		<>
			<div className="space-y-6">
				<PageHeader
					title={textGet("team.title")}
					description={textGet("team.description")}
					actions={
						canManageTeam && (
							<Button onClick={openRoleSelector}>
								<UserPlus className="h-4 w-4 mr-2" />
								{textGet("team.invite")}
							</Button>
						)
					}
				/>

				{loading ? (
					<ul className="divide-y rounded-xl border bg-card">
						{[1, 2, 3, 4].map((i) => (
							<SkeletonRow key={i} />
						))}
					</ul>
				) : members.length === 0 ? (
					<div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed py-16">
						<div className="h-12 w-12 rounded-full bg-muted flex items-center justify-center">
							<Users className="h-6 w-6 text-muted-foreground" />
						</div>
						<p className="text-sm text-muted-foreground">
							{textGet("team.members.empty")}
						</p>
						{canManageTeam && (
							<Button variant="outline" size="sm" onClick={openRoleSelector}>
								<UserPlus className="h-4 w-4 mr-2" />
								{textGet("team.invite")}
							</Button>
						)}
					</div>
				) : (
					<section className="rounded-xl border bg-card">
						<header className="flex flex-wrap items-center gap-3 border-b px-4 py-3 sm:px-5">
							{showFilters ? (
								<fieldset className="flex flex-wrap items-center gap-1.5">
									<legend className="sr-only">
										{textGet("team.filter.label")}
									</legend>
									<FilterChip
										active={roleFilter === null}
										onClick={() => setRoleFilter(null)}
										label={textGet("team.filter.all")}
										count={members.length}
									/>
									{roleCounts.map(([role, count]) => (
										<FilterChip
											key={role}
											active={roleFilter === role}
											onClick={() =>
												setRoleFilter(roleFilter === role ? null : role)
											}
											label={roleLabel(role)}
											count={count}
											dot={getRoleStyle(role).dot}
										/>
									))}
								</fieldset>
							) : (
								<p className="text-sm text-muted-foreground">
									{textGet("team.members.total", { count: members.length })}
									{roleCounts[0] && (
										<span className="ml-2 inline-flex items-center gap-1.5">
											<span
												className={cn(
													"size-2 rounded-full",
													getRoleStyle(roleCounts[0][0]).dot,
												)}
											/>
											{roleLabel(roleCounts[0][0])}
										</span>
									)}
								</p>
							)}

							{showSearch && (
								<InputGroup className="ml-auto h-8 w-full sm:w-64">
									<InputGroupAddon>
										<Search />
									</InputGroupAddon>
									<InputGroupInput
										value={query}
										onChange={(e) => setQuery(e.target.value)}
										placeholder={textGet("team.search.placeholder")}
										aria-label={textGet("team.search.placeholder")}
									/>
									{query && (
										<InputGroupAddon align="inline-end">
											<InputGroupButton
												size="icon-xs"
												aria-label={textGet("team.search.clear")}
												onClick={() => setQuery("")}
											>
												<X />
											</InputGroupButton>
										</InputGroupAddon>
									)}
								</InputGroup>
							)}
						</header>

						{visibleMembers.length === 0 ? (
							<p className="px-5 py-10 text-center text-sm text-muted-foreground">
								{textGet("team.search.empty", { query: query.trim() })}
							</p>
						) : (
							<ul className="divide-y">
								{visibleMembers.map((member) => (
									<MemberRow
										key={member.environment_id}
										member={member}
										isSelf={member.environment_id === currentEnvironmentId}
										canManageTeam={canManageTeam}
										roles={roles}
										updating={updatingEnvironmentId === member.environment_id}
										onRoleChange={handleRoleChange}
									/>
								))}
							</ul>
						)}
					</section>
				)}
			</div>

			{/* Step 1: pick the role the invite grants */}
			<Dialog open={roleDialogOpen} onOpenChange={setRoleDialogOpen}>
				<DialogContent className="max-w-md">
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

					<RadioGroup
						aria-label={textGet("team.role_dialog.role_label")}
						value={selectedRole ? String(selectedRole.ID) : ""}
						onValueChange={(val) =>
							setSelectedRole(
								inviteRoles.find((r) => String(r.ID) === val) ?? null,
							)
						}
					>
						{inviteRoles.map((r) => {
							const key = r.role.toLowerCase();
							const id = `invite-role-${r.ID}`;
							return (
								<Label
									key={r.ID}
									htmlFor={id}
									className="flex cursor-pointer items-start gap-3 rounded-lg border p-3 font-normal transition-colors hover:bg-muted/50 has-data-checked:border-primary has-data-checked:bg-primary/5"
								>
									<RadioGroupItem
										id={id}
										value={String(r.ID)}
										className="mt-0.5"
									/>
									<span className="min-w-0 space-y-0.5">
										<span className="flex items-center gap-2 text-sm font-medium">
											<span
												className={cn(
													"size-2 rounded-full",
													getRoleStyle(key).dot,
												)}
											/>
											{roleLabel(r.role)}
										</span>
										{CANONICAL_ROLES.includes(key) && (
											<span className="block text-xs text-muted-foreground">
												{textGet(`team.role_hint.${key}`)}
											</span>
										)}
									</span>
								</Label>
							);
						})}
					</RadioGroup>

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

			{/* Step 2: share the link */}
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
									<strong>{roleLabel(selectedRole.role)}</strong>.{" "}
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
								aria-label={textGet("team.invite_dialog.title")}
								className="shrink-0 transition-all"
							>
								{copied ? (
									<Check className="h-4 w-4" />
								) : (
									<Copy className="h-4 w-4" />
								)}
							</Button>
						</div>
						<p
							aria-live="polite"
							className="min-h-4 text-xs text-center text-emerald-600 font-medium"
						>
							{copied && textGet("team.invite_dialog.copied")}
						</p>
						<div className="flex items-start gap-2 rounded-lg bg-amber-500/10 border border-amber-500/20 px-3 py-2">
							<ShieldAlert className="h-4 w-4 text-amber-600 shrink-0 mt-0.5" />
							<p className="text-xs text-amber-700 dark:text-amber-400">
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
		</>
	);
};

function FilterChip({
	active,
	onClick,
	label,
	count,
	dot,
}: {
	active: boolean;
	onClick: () => void;
	label: string;
	count: number;
	dot?: string;
}) {
	return (
		<button
			type="button"
			aria-pressed={active}
			onClick={onClick}
			className={cn(
				"inline-flex h-7 items-center gap-1.5 rounded-full border px-3 text-xs transition-colors outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
				active
					? "border-foreground/20 bg-foreground text-background"
					: "border-transparent text-muted-foreground hover:bg-muted hover:text-foreground",
			)}
		>
			{dot && <span className={cn("size-2 rounded-full", dot)} />}
			{label}
			<span className="tabular-nums opacity-60">{count}</span>
		</button>
	);
}

export default TeamPage;
