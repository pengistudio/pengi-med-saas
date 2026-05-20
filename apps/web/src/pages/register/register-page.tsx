import {
	Building2,
	CalendarDays,
	CheckCircle2,
	ClipboardList,
	ReceiptText,
	UsersRound,
} from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { register } from "@/api/auth-service";
import GentooPenguin from "@/assets/gentoo.png";
import SelectLanguage from "@/components/custom/select-language";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormPasswordInput } from "@/components/forms/form-input-password";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Text } from "@/components/ui/text";
import { useText } from "@/hooks/use-text";

const formSchema = z
	.object({
		company_name: z.string().min(2).max(100),
		username: z
			.string()
			.min(3)
			.max(50)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
		email: z.email(),
		password: z
			.string()
			.min(6)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
		confirm_password: z
			.string()
			.min(6)
			.regex(/^\S+$/, { message: "form.validation.no_spaces" }),
	})
	.refine((data) => data.password === data.confirm_password, {
		message: "form.validation.passwords_no_match",
		path: ["confirm_password"],
	});

const features = [
	{ icon: UsersRound, label: "Gestión de pacientes y expedientes ICD-11" },
	{ icon: CalendarDays, label: "Agenda de citas integrada" },
	{ icon: ClipboardList, label: "Recetas médicas en PDF" },
	{ icon: Building2, label: "Multi-consultorio y roles" },
	{ icon: ReceiptText, label: "Facturación electrónica SRI" },
];

const BrandPanel = () => (
	<div
		className="hidden lg:flex lg:w-[45%] xl:w-[42%] flex-col justify-between p-12 relative overflow-hidden"
		style={{
			background:
				"linear-gradient(135deg, oklch(59.929% 0.0998 185.282), oklch(62% 0.14 280))",
		}}
	>
		<div
			className="absolute -top-24 -right-24 w-96 h-96 rounded-full blur-3xl pointer-events-none opacity-30"
			style={{ background: "oklch(100% 0 0 / 20%)" }}
		/>
		<div
			className="absolute -bottom-32 -left-16 w-80 h-80 rounded-full blur-3xl pointer-events-none opacity-20"
			style={{ background: "oklch(100% 0 0 / 15%)" }}
		/>

		<div className="flex items-center gap-2 relative z-10">
			<img
				src={GentooPenguin}
				alt="Gentoo"
				className="w-9 h-9"
				style={{ filter: "brightness(0) invert(1)" }}
			/>
			<span className="text-white font-bold text-xl tracking-tight">
				Gentoo
			</span>
		</div>

		<div className="relative z-10 space-y-8">
			<div className="space-y-3">
				<h2 className="text-white font-bold text-3xl leading-tight">
					Tu consultorio,
					<br />
					completamente digital
				</h2>
				<p className="text-white/70 text-base leading-relaxed">
					Todo lo que necesitas para gestionar tu práctica médica en un solo
					lugar.
				</p>
			</div>

			<ul className="space-y-4">
				{features.map(({ icon: Icon, label }) => (
					<li key={label} className="flex items-center gap-3">
						<div
							className="w-8 h-8 rounded-lg flex items-center justify-center shrink-0"
							style={{ background: "oklch(100% 0 0 / 15%)" }}
						>
							<Icon size={16} color="white" strokeWidth={1.8} />
						</div>
						<span className="text-white/90 text-sm">{label}</span>
					</li>
				))}
			</ul>
		</div>

		<div className="relative z-10 flex items-center gap-2">
			<CheckCircle2
				size={16}
				color="white"
				strokeWidth={2}
				className="opacity-70"
			/>
			<span className="text-white/70 text-sm">
				14 días gratis · Sin tarjeta de crédito
			</span>
		</div>
	</div>
);

const RegisterPage = () => {
	const { textGet } = useText();
	const [load, setLoad] = React.useState(false);
	const [done, setDone] = React.useState(false);
	const navigate = useNavigate();

	if (done) {
		return (
			<div className="min-h-screen flex">
				<BrandPanel />
				<div className="flex-1 flex flex-col bg-muted/30">
					<div className="flex items-center justify-between p-6 lg:hidden">
						<div className="flex items-center gap-2">
							<img src={GentooPenguin} alt="Gentoo" className="w-7 h-7" />
							<span className="font-bold text-lg">Gentoo</span>
						</div>
					</div>
					<div className="flex-1 flex items-center justify-center px-6 py-12">
						<div className="w-full max-w-lg space-y-4">
							<h1 className="text-2xl font-bold text-foreground">
								<Text uuid="register.check_email.title" />
							</h1>
							<p className="text-muted-foreground">
								<Text uuid="register.check_email.description" />
							</p>
							<Button
								variant="outline"
								className="mt-2"
								onClick={() => navigate("/login")}
							>
								<Text uuid="register.go_to_login" />
							</Button>
						</div>
					</div>
					<div className="p-6 flex justify-center">
						<SelectLanguage />
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen flex">
			<BrandPanel />

			{/* Form panel */}
			<div className="flex-1 flex flex-col bg-muted/30">
				{/* Mobile header */}
				<div className="flex items-center justify-between p-6 lg:hidden">
					<div className="flex items-center gap-2">
						<img src={GentooPenguin} alt="Gentoo" className="w-7 h-7" />
						<span className="font-bold text-lg">Gentoo</span>
					</div>
					<Button variant="ghost" size="sm" onClick={() => navigate("/login")}>
						<Text uuid="register.go_to_login" />
					</Button>
				</div>

				{/* Centered form */}
				<div className="flex-1 flex items-center justify-center px-6 py-12">
					<Form<typeof formSchema>
						schema={formSchema}
						onSubmit={onSubmit}
						className="w-full max-w-lg"
						defaultValues={{
							company_name: "",
							username: "",
							email: "",
							password: "",
							confirm_password: "",
						}}
					>
						{(field) => (
							<div className="space-y-6">
								{/* Heading */}
								<div className="space-y-1">
									<h1 className="text-2xl font-bold text-foreground">
										<Text uuid="register.title" />
									</h1>
									<p className="text-muted-foreground text-sm">
										<Text uuid="register.subtitle" />
									</p>
								</div>

								{/* Fields */}
								<div className="space-y-4">
									<FormInput
										field={field}
										name="company_name"
										type="text"
										placeholder={textGet("register.company_name.placeholder")}
										label={textGet("register.company_name")}
										autoComplete="organization"
									/>
									<FormInput
										field={field}
										name="username"
										type="text"
										placeholder={textGet("register.username.placeholder")}
										label={textGet("register.username")}
										autoComplete="username"
									/>
									<FormInput
										field={field}
										name="email"
										type="email"
										placeholder={textGet("register.email.placeholder")}
										label={textGet("register.email")}
										autoComplete="email"
									/>
									<div className="grid grid-cols-2 gap-4">
										<FormPasswordInput
											field={field}
											name="password"
											placeholder={textGet("register.password.placeholder")}
											label={textGet("register.password")}
										/>
										<FormPasswordInput
											field={field}
											name="confirm_password"
											placeholder={textGet(
												"register.confirm_password.placeholder",
											)}
											label={textGet("register.confirm_password")}
										/>
									</div>
								</div>

								{/* Actions */}
								<div className="space-y-3">
									<Button type="submit" className="w-full" disabled={load}>
										{load && <Spinner />}
										<Text uuid="register.submit_button" />
									</Button>
									<Button
										variant="ghost"
										className="w-full"
										type="button"
										onClick={() => navigate("/login")}
									>
										<Text uuid="register.go_to_login" />
									</Button>
								</div>
							</div>
						)}
					</Form>
				</div>

				{/* Footer */}
				<div className="p-6 flex justify-center">
					<SelectLanguage />
				</div>
			</div>
		</div>
	);

	async function onSubmit(values: z.infer<typeof formSchema>) {
		setLoad(true);
		const res = await register({
			company_name: values.company_name,
			username: values.username,
			email: values.email,
			password: values.password,
		});
		setLoad(false);
		if (res.success) {
			setDone(true);
		}
	}
};

export default RegisterPage;
