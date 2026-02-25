import { useEffect, useState } from "react";
import { useLocation } from "react-router";
import {
	Accordion,
	AccordionContent,
	AccordionItem,
	AccordionTrigger,
} from "@/components/ui/accordion";
import { cn } from "@/lib/utils";
import { useSidebarStore } from "@/store/sidebar-store";
import NavItem from "./nav-item";

type Props = {
	label: string;
	icon: React.ComponentType<{ className?: string }>;
	accordionItems: {
		label: string;
		href: string;
		icon: React.ComponentType<{ className?: string }>;
	}[];
};

const NavAccordion = (props: Props) => {
	const { isOpen: sidebarOpen, open } = useSidebarStore();
	const location = useLocation();
	const { label, icon, accordionItems } = props;

	// Check if any accordion item is active
	const isAnyItemActive = accordionItems.some(
		(item) => location.pathname === item.href,
	);

	// Controlled accordion state
	const [accordionValue, setAccordionValue] = useState<string[]>(
		isAnyItemActive ? ["item-1"] : [],
	);

	// Close all accordions when sidebar closes
	useEffect(() => {
		if (!sidebarOpen) {
			setAccordionValue([]);
		} else if (isAnyItemActive) {
			setAccordionValue(["item-1"]);
		}
	}, [sidebarOpen, isAnyItemActive]);

	const Icon = icon;
	return (
		<Accordion
			multiple
			value={accordionValue}
			onValueChange={setAccordionValue}
		>
			<AccordionItem value="item-1">
				<AccordionTrigger
					className={cn(
						"flex items-center rounded-lg px-3 py-2 transition-all duration-300 overflow-hidden justify-start hover:no-underline hover:cursor-pointer",
						sidebarOpen ? "gap-3" : "gap-0",
						isAnyItemActive
							? "bg-sidebar-accent text-sidebar-accent-foreground"
							: "text-sidebar-foreground hover:bg-sidebar-accent",
					)}
					onClick={() => {
						if (!sidebarOpen) {
							open();
						}
					}}
				>
					<div className="flex items-center gap-3 flex-1">
						<Icon
							className={cn(
								"h-5 w-5 shrink-0",
								isAnyItemActive && "text-primary",
							)}
						/>
						<span
							className={cn(
								"transition-all duration-300 truncate text-base font-normal",
								sidebarOpen
									? "opacity-100 w-auto"
									: "opacity-0 w-0 overflow-hidden ml-0",
							)}
						>
							{label}
						</span>
					</div>
				</AccordionTrigger>
				<AccordionContent className="pb-0">
					{accordionItems.map((item) => (
						<div key={item.label} className="ml-5 border-l">
							<div className="ml-2 py-1">
								<NavItem label={item.label} href={item.href} icon={item.icon} />
							</div>
						</div>
					))}
				</AccordionContent>
			</AccordionItem>
		</Accordion>
	);
};

export default NavAccordion;
