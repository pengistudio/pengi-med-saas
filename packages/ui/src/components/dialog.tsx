"use client";

import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { XIcon } from "lucide-react";
import type * as React from "react";
import { createContext, useContext } from "react";
import { useUiText } from "../context/text-context";
import { phoneSheet, phoneSheetFooter } from "../lib/phone-sheet";
import { cn } from "../lib/utils";
import { Button } from "./button";

function Dialog({ ...props }: DialogPrimitive.Root.Props) {
	return <DialogPrimitive.Root data-slot="dialog" {...props} />;
}

function DialogTrigger({ ...props }: DialogPrimitive.Trigger.Props) {
	return <DialogPrimitive.Trigger data-slot="dialog-trigger" {...props} />;
}

function DialogPortal({ ...props }: DialogPrimitive.Portal.Props) {
	return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />;
}

function DialogClose({ ...props }: DialogPrimitive.Close.Props) {
	return <DialogPrimitive.Close data-slot="dialog-close" {...props} />;
}

function DialogOverlay({
	className,
	...props
}: DialogPrimitive.Backdrop.Props) {
	return (
		<DialogPrimitive.Backdrop
			data-slot="dialog-overlay"
			className={cn(
				"data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 bg-black/10 duration-100 supports-backdrop-filter:backdrop-blur-xs fixed inset-0 isolate z-50",
				className,
			)}
			{...props}
		/>
	);
}

// Lets DialogHeader carry the close button, so it stays in view while the
// dialog scrolls; dialogs without a header keep it in the corner.
const DialogCloseContext = createContext(false);

function DialogCloseButton({ className }: { className?: string }) {
	const { textGet } = useUiText();
	return (
		<DialogPrimitive.Close
			data-slot="dialog-close"
			render={
				<Button
					variant="ghost"
					className={cn("absolute top-2 right-2 max-sm:size-9", className)}
					size="icon-sm"
				/>
			}
		>
			<XIcon />
			<span className="sr-only">{textGet("dialog.close")}</span>
		</DialogPrimitive.Close>
	);
}

function DialogContent({
	className,
	children,
	showCloseButton = true,
	...props
}: DialogPrimitive.Popup.Props & {
	showCloseButton?: boolean;
}) {
	return (
		<DialogPortal>
			<DialogOverlay />
			<DialogPrimitive.Popup
				data-slot="dialog-content"
				className={cn(
					"group/dialog bg-background data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 ring-foreground/10 grid max-w-[calc(100%-2rem)] gap-4 rounded-xl p-4 text-sm ring-1 duration-100 sm:max-w-sm fixed top-1/2 left-1/2 z-50 w-full -translate-x-1/2 -translate-y-1/2 outline-none",
					// Taller than the screen: scroll inside, header and footer pinned.
					"max-h-[calc(100dvh-2rem)] overflow-y-auto overscroll-contain",
					phoneSheet,
					className,
				)}
				{...props}
			>
				<DialogCloseContext.Provider value={showCloseButton}>
					{children}
				</DialogCloseContext.Provider>
				{showCloseButton && (
					<DialogCloseButton className="group-has-[[data-slot=dialog-header]]/dialog:hidden" />
				)}
			</DialogPrimitive.Popup>
		</DialogPortal>
	);
}

function DialogHeader({
	className,
	children,
	...props
}: React.ComponentProps<"div">) {
	const showCloseButton = useContext(DialogCloseContext);
	return (
		<div
			data-slot="dialog-header"
			className={cn(
				// Pinned to the top of a scrolling dialog (past its p-4).
				"sticky -top-4 z-10 -mx-4 -mt-4 -mb-2 bg-background px-4 pt-4 pb-2 gap-2 flex flex-col",
				showCloseButton && "pr-12",
				className,
			)}
			{...props}
		>
			{children}
			{showCloseButton && <DialogCloseButton />}
		</div>
	);
}

function DialogFooter({
	className,
	showCloseButton = false,
	children,
	...props
}: React.ComponentProps<"div"> & {
	showCloseButton?: boolean;
}) {
	const { textGet } = useUiText();
	return (
		<div
			data-slot="dialog-footer"
			className={cn(
				// Pinned to the bottom of a scrolling dialog (past its p-4), so its tint
				// must be opaque.
				"sticky -bottom-4 z-10 bg-[color-mix(in_oklab,var(--muted)_50%,var(--background))] -mx-4 -mb-4 rounded-b-xl border-t p-4 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
				phoneSheetFooter,
				className,
			)}
			{...props}
		>
			{children}
			{showCloseButton && (
				<DialogPrimitive.Close render={<Button variant="outline" />}>
					{textGet("dialog.close")}
				</DialogPrimitive.Close>
			)}
		</div>
	);
}

function DialogTitle({ className, ...props }: DialogPrimitive.Title.Props) {
	return (
		<DialogPrimitive.Title
			data-slot="dialog-title"
			className={cn("text-base leading-none font-medium", className)}
			{...props}
		/>
	);
}

function DialogDescription({
	className,
	...props
}: DialogPrimitive.Description.Props) {
	return (
		<DialogPrimitive.Description
			data-slot="dialog-description"
			className={cn(
				"text-muted-foreground *:[a]:hover:text-foreground text-sm *:[a]:underline *:[a]:underline-offset-3",
				className,
			)}
			{...props}
		/>
	);
}

export {
	Dialog,
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogOverlay,
	DialogPortal,
	DialogTitle,
	DialogTrigger,
};
