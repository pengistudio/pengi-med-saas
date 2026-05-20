import { CreditCard, Database, Lock, Mail, Shield } from "lucide-react";
import { Link } from "react-router";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const SECTIONS = [
	{
		icon: CreditCard,
		titleKey: "privacy.section.no_card.title",
		descKey: "privacy.section.no_card.desc",
	},
	{
		icon: Shield,
		titleKey: "privacy.section.processor.title",
		descKey: "privacy.section.processor.desc",
	},
	{
		icon: Database,
		titleKey: "privacy.section.data.title",
		descKey: "privacy.section.data.desc",
	},
	{
		icon: Lock,
		titleKey: "privacy.section.security.title",
		descKey: "privacy.section.security.desc",
	},
	{
		icon: Mail,
		titleKey: "privacy.section.contact.title",
		descKey: "privacy.section.contact.desc",
	},
] as const;

const PrivacyPage = () => {
	const { textGet } = useText();

	return (
		<DashboardLayout>
			<div className="max-w-2xl space-y-8">
				<div>
					<Link
						to="/subscription"
						className="text-sm text-muted-foreground hover:text-foreground transition-colors"
					>
						← {textGet("subscription.page.title")}
					</Link>
					<div className="flex items-center gap-3 mt-4 mb-1">
						<div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
							<Shield className="h-5 w-5 text-primary" />
						</div>
						<h1 className="text-3xl font-bold tracking-tight">
							{textGet("privacy.page.title")}
						</h1>
					</div>
					<p className="text-sm text-muted-foreground">
						{textGet("privacy.page.subtitle")}
					</p>
				</div>

				<div className="space-y-8">
					{SECTIONS.map(({ icon: Icon, titleKey, descKey }) => (
						<section key={titleKey} className="flex gap-4">
							<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted mt-0.5">
								<Icon className="h-4 w-4 text-muted-foreground" />
							</div>
							<div>
								<h2 className="font-semibold mb-1">{textGet(titleKey)}</h2>
								<p className="text-sm text-muted-foreground leading-relaxed">
									{textGet(descKey)}
								</p>
							</div>
						</section>
					))}
				</div>
			</div>
		</DashboardLayout>
	);
};

export default PrivacyPage;
