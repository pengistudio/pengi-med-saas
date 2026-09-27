import { useText } from "@pengi/shared";
import { CreditCard, Database, Lock, Mail, Shield, User } from "lucide-react";
import { Link } from "react-router";
import { PageHeader } from "@/components/custom/page-header";
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
	{
		icon: User,
		titleKey: "privacy.section.legal.title",
		descKey: "privacy.section.legal.desc",
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
					<PageHeader
						className="mt-4"
						title={textGet("privacy.page.title")}
						description={textGet("privacy.page.subtitle")}
					/>
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
