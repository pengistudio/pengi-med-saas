import { Card, Text } from "@pengi/ui";
import { ChevronRight, Stethoscope, UserPlus } from "lucide-react";
import { cn } from "@/lib/utils";

type VisitType = "first" | "followup";

const OPTIONS: {
	value: VisitType;
	icon: typeof UserPlus;
	titleKey: string;
	descriptionKey: string;
	accent: string;
}[] = [
	{
		value: "first",
		icon: UserPlus,
		titleKey: "form.create_medical_record.visit_type.first",
		descriptionKey: "form.create_medical_record.visit_type.first.description",
		accent: "emerald",
	},
	{
		value: "followup",
		icon: Stethoscope,
		titleKey: "form.create_medical_record.visit_type.followup",
		descriptionKey:
			"form.create_medical_record.visit_type.followup.description",
		accent: "blue",
	},
];

const ACCENT_CLASSES: Record<string, { icon: string; ring: string }> = {
	emerald: {
		icon: "bg-emerald-500 text-white",
		ring: "hover:ring-emerald-500/60 focus-visible:ring-emerald-500",
	},
	blue: {
		icon: "bg-blue-500 text-white",
		ring: "hover:ring-blue-500/60 focus-visible:ring-blue-500",
	},
};

const VisitTypeChooser = ({
	onSelect,
}: {
	onSelect: (visitType: VisitType) => void;
}) => {
	return (
		<div className="max-w-3xl mx-auto py-8 sm:py-16 space-y-8">
			<div className="text-center space-y-1.5">
				<h2 className="text-2xl font-semibold tracking-tight">
					<Text uuid="form.create_medical_record.visit_type.title" />
				</h2>
				<p className="text-sm text-muted-foreground">
					<Text uuid="form.create_medical_record.visit_type.description" />
				</p>
			</div>

			<div className="grid sm:grid-cols-2 grid-cols-1 gap-5">
				{OPTIONS.map(
					({ value, icon: Icon, titleKey, descriptionKey, accent }) => {
						const colors = ACCENT_CLASSES[accent];
						return (
							<Card
								key={value}
								role="button"
								tabIndex={0}
								onClick={() => onSelect(value)}
								onKeyDown={(e) => {
									if (e.key === "Enter" || e.key === " ") {
										e.preventDefault();
										onSelect(value);
									}
								}}
								className={cn(
									"group cursor-pointer p-6 transition-all hover:-translate-y-0.5 hover:shadow-lg hover:ring-2 outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
									colors.ring,
								)}
							>
								<div
									className={cn(
										"flex h-12 w-12 items-center justify-center rounded-xl mb-4",
										colors.icon,
									)}
								>
									<Icon className="h-6 w-6" />
								</div>
								<h3 className="text-lg font-semibold leading-snug">
									<Text uuid={titleKey} />
								</h3>
								<p className="text-sm text-muted-foreground mt-1">
									<Text uuid={descriptionKey} />
								</p>
								<div className="flex items-center gap-1 mt-5 text-sm font-medium text-muted-foreground group-hover:text-foreground transition-colors">
									<Text uuid="form.create_medical_record.visit_type.select" />
									<ChevronRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
								</div>
							</Card>
						);
					},
				)}
			</div>
		</div>
	);
};

export default VisitTypeChooser;
