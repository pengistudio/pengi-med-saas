import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../lib/utils";

function Tabs({
	className,
	orientation = "horizontal",
	...props
}: TabsPrimitive.Root.Props) {
	return (
		<TabsPrimitive.Root
			data-slot="tabs"
			data-orientation={orientation}
			className={cn(
				"gap-2 group/tabs flex data-horizontal:flex-col",
				className,
			)}
			{...props}
		/>
	);
}

const tabsListVariants = cva(
	"relative isolate rounded-lg p-[3px] group-data-horizontal/tabs:h-8 data-[variant=line]:rounded-none group/tabs-list text-muted-foreground inline-flex w-fit items-center justify-center group-data-vertical/tabs:h-fit group-data-vertical/tabs:flex-col",
	{
		variants: {
			variant: {
				default: "bg-muted",
				line: "gap-1 bg-transparent",
			},
		},
		defaultVariants: {
			variant: "default",
		},
	},
);

const tabsIndicator = [
	"absolute -z-10 transition-[left,top,width,height] duration-(--motion-slow) ease-spring motion-reduce:transition-none",
	"group-data-[variant=default]/tabs-list:top-(--active-tab-top) group-data-[variant=default]/tabs-list:left-(--active-tab-left) group-data-[variant=default]/tabs-list:h-(--active-tab-height) group-data-[variant=default]/tabs-list:w-(--active-tab-width) group-data-[variant=default]/tabs-list:rounded-md group-data-[variant=default]/tabs-list:bg-background group-data-[variant=default]/tabs-list:shadow-sm dark:group-data-[variant=default]/tabs-list:border dark:group-data-[variant=default]/tabs-list:border-input dark:group-data-[variant=default]/tabs-list:bg-input/30",
	"group-data-[variant=line]/tabs-list:bg-foreground group-data-[variant=line]/tabs-list:group-data-horizontal/tabs:bottom-[-0.5px] group-data-[variant=line]/tabs-list:group-data-horizontal/tabs:left-(--active-tab-left) group-data-[variant=line]/tabs-list:group-data-horizontal/tabs:h-0.5 group-data-[variant=line]/tabs-list:group-data-horizontal/tabs:w-(--active-tab-width)",
	"group-data-[variant=line]/tabs-list:group-data-vertical/tabs:top-(--active-tab-top) group-data-[variant=line]/tabs-list:group-data-vertical/tabs:right-[-2px] group-data-[variant=line]/tabs-list:group-data-vertical/tabs:h-(--active-tab-height) group-data-[variant=line]/tabs-list:group-data-vertical/tabs:w-0.5",
].join(" ");

function TabsList({
	className,
	variant = "default",
	children,
	...props
}: TabsPrimitive.List.Props & VariantProps<typeof tabsListVariants>) {
	return (
		<TabsPrimitive.List
			data-slot="tabs-list"
			data-variant={variant}
			className={cn(tabsListVariants({ variant }), className)}
			{...props}
		>
			{children}
			<TabsPrimitive.Indicator
				data-slot="tabs-indicator"
				className={tabsIndicator}
			/>
		</TabsPrimitive.List>
	);
}

function TabsTrigger({ className, ...props }: TabsPrimitive.Tab.Props) {
	return (
		<TabsPrimitive.Tab
			data-slot="tabs-trigger"
			type="button"
			className={cn(
				"gap-1.5 rounded-md border border-transparent px-1.5 py-0.5 text-sm font-medium [&_svg:not([class*='size-'])]:size-4 focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:outline-ring text-foreground/60 hover:text-foreground dark:text-muted-foreground dark:hover:text-foreground relative inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center whitespace-nowrap transition-[color,box-shadow] duration-(--motion-fast) group-data-vertical/tabs:w-full group-data-vertical/tabs:justify-start focus-visible:ring-[3px] focus-visible:outline-1 disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
				"data-active:text-foreground dark:data-active:text-foreground",
				className,
			)}
			{...props}
		/>
	);
}

function TabsContent({ className, ...props }: TabsPrimitive.Panel.Props) {
	return (
		<TabsPrimitive.Panel
			data-slot="tabs-content"
			className={cn(
				"text-sm flex-1 outline-none",
				"data-[activation-direction=right]:animate-in data-[activation-direction=left]:animate-in fade-in-0 data-[activation-direction=right]:slide-in-from-right-2 data-[activation-direction=left]:slide-in-from-left-2 duration-(--motion-base) ease-out-soft motion-reduce:animate-none",
				className,
			)}
			{...props}
		/>
	);
}

export { Tabs, TabsContent, TabsList, TabsTrigger, tabsListVariants };
