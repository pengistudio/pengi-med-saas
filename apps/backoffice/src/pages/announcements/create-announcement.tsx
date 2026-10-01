import { useText } from "@pengi/shared";
import {
	Button,
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
	Form,
	FormInput,
	FormRadioGroup,
	FormSelect,
	FormTextArea,
	Spinner,
} from "@pengi/ui";
import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import {
	ANNOUNCEMENT_LEVELS,
	ANNOUNCEMENT_SCOPES,
	announcements,
	type CreateAnnouncementRequest,
} from "@/api/announcement-service";
import {
	type Company,
	type CompanyUser,
	companies as companyResource,
	getCompanyUsers,
} from "@/api/company-service";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const key = (suffix: string) => `backoffice.announcements.${suffix}`;

const formSchema = z
	.object({
		scope: z.enum(ANNOUNCEMENT_SCOPES),
		company_id: z.string(),
		user_id: z.string(),
		title: z.string().trim().min(1).max(120),
		body: z.string().trim().min(1).max(1000),
		level: z.enum(ANNOUNCEMENT_LEVELS),
		action_url: z
			.string()
			.trim()
			.max(500)
			.refine(
				(url) =>
					url === "" ||
					/^https:\/\/\S+$/.test(url) ||
					(url.startsWith("/") && !url.startsWith("//")),
				{ message: key("validation.action_url") },
			),
		// datetime-local value, in the admin's local time; "" = send now.
		scheduled_at: z
			.string()
			.refine((value) => value === "" || new Date(value) > new Date(), {
				message: key("validation.scheduled_future"),
			}),
	})
	.superRefine((values, ctx) => {
		if (values.scope !== "global" && !values.company_id) {
			ctx.addIssue({
				code: "custom",
				path: ["company_id"],
				message: key("validation.company_required"),
			});
		}
		if (values.scope === "user" && !values.user_id) {
			ctx.addIssue({
				code: "custom",
				path: ["user_id"],
				message: key("validation.user_required"),
			});
		}
	});

type FormValues = z.input<typeof formSchema>;

const CreateAnnouncement = () => {
	const { textGet } = useText();
	const navigate = useNavigate();
	const { saving, save } = useResourceItem(announcements);
	const [companies, setCompanies] = React.useState<Company[]>([]);
	const [companyUsers, setCompanyUsers] = React.useState<CompanyUser[]>([]);
	const [scope, setScope] = React.useState<FormValues["scope"]>("global");
	const [companyId, setCompanyId] = React.useState("");

	React.useEffect(() => {
		companyResource.list().then((res) => {
			if (res.success) setCompanies(res.data ?? []);
		});
	}, []);

	// Clear the users of the previous company (adjust state during render).
	const [prevTarget, setPrevTarget] = React.useState({ scope, companyId });
	if (prevTarget.scope !== scope || prevTarget.companyId !== companyId) {
		setPrevTarget({ scope, companyId });
		setCompanyUsers([]);
	}

	React.useEffect(() => {
		if (scope !== "user" || !companyId) return;
		let cancelled = false;
		getCompanyUsers(companyId).then((res) => {
			if (!cancelled && res.success) setCompanyUsers(res.data ?? []);
		});
		return () => {
			cancelled = true;
		};
	}, [scope, companyId]);

	const onValuesChange = React.useCallback((values: Partial<FormValues>) => {
		setScope(values.scope ?? "global");
		setCompanyId(values.company_id ?? "");
	}, []);

	async function onSubmit(values: z.output<typeof formSchema>) {
		const payload: CreateAnnouncementRequest = {
			scope: values.scope,
			title: values.title,
			body: values.body,
			level: values.level,
		};
		if (values.scope !== "global")
			payload.company_id = Number(values.company_id);
		if (values.scope === "user") payload.user_id = Number(values.user_id);
		if (values.action_url) payload.action_url = values.action_url;
		if (values.scheduled_at) {
			payload.scheduled_at = new Date(values.scheduled_at).toISOString();
		}
		await save(payload);
	}

	const scopeOptions = ANNOUNCEMENT_SCOPES.map((value) => ({
		value,
		label: textGet(key(`scope.${value}`)),
	}));
	const levelOptions = ANNOUNCEMENT_LEVELS.map((value) => ({
		value,
		label: textGet(key(`level.${value}`)),
	}));
	const companyOptions = companies.map((c) => ({
		value: String(c.ID),
		label: c.trade_name,
	}));
	const userOptions = companyUsers.map((u) => ({
		value: String(u.user_id),
		label: u.email ? `${u.user_name} · ${u.email}` : u.user_name,
	}));

	return (
		<ResourceEditPage>
			<div className="max-w-2xl mx-auto">
				<Form<typeof formSchema>
					schema={formSchema}
					onSubmit={onSubmit}
					onUpdateValuesCallback={onValuesChange}
					defaultValues={{
						scope: "global",
						company_id: "",
						user_id: "",
						title: "",
						body: "",
						level: "info",
						action_url: "",
						scheduled_at: "",
					}}
				>
					{(field) => (
						<Card>
							<CardHeader>
								<CardTitle>{textGet(key("create.title"))}</CardTitle>
								<CardDescription>
									{textGet(key("create.description"))}
								</CardDescription>
							</CardHeader>
							<CardContent className="space-y-4">
								<FormRadioGroup
									field={field}
									name="scope"
									isRow
									label={textGet(key("col.scope"))}
									options={scopeOptions}
								/>
								{scope !== "global" && (
									<FormSelect
										field={field}
										name="company_id"
										label={textGet(key("col.company"))}
										placeholder={textGet(key("col.company.placeholder"))}
										options={companyOptions}
									/>
								)}
								{scope === "user" && (
									<FormSelect
										field={field}
										name="user_id"
										label={textGet(key("col.user"))}
										placeholder={textGet(key("col.user.placeholder"))}
										options={userOptions}
										disabled={!companyId}
									/>
								)}
								<FormInput
									field={field}
									name="title"
									type="text"
									label={textGet(key("col.title"))}
									placeholder={textGet(key("col.title.placeholder"))}
								/>
								<FormTextArea
									field={field}
									name="body"
									rows={4}
									label={textGet(key("col.body"))}
									placeholder={textGet(key("col.body.placeholder"))}
								/>
								<FormSelect
									field={field}
									name="level"
									label={textGet(key("col.level"))}
									options={levelOptions}
								/>
								<FormInput
									field={field}
									name="action_url"
									type="text"
									isOptional
									label={textGet(key("col.action_url"))}
									placeholder="/billing"
									description={textGet(key("col.action_url.description"))}
								/>
								<FormInput
									field={field}
									name="scheduled_at"
									type="datetime-local"
									isOptional
									label={textGet(key("col.scheduled_at"))}
									description={textGet(key("col.scheduled_at.description"))}
								/>
							</CardContent>
							<CardFooter className="flex justify-between">
								<Button
									type="button"
									variant="outline"
									onClick={() => navigate("/announcements")}
								>
									{textGet("backoffice.common.cancel")}
								</Button>
								<Button type="submit" disabled={saving}>
									{saving && <Spinner />}
									{textGet(key("create.submit"))}
								</Button>
							</CardFooter>
						</Card>
					)}
				</Form>
			</div>
		</ResourceEditPage>
	);
};

export default CreateAnnouncement;
