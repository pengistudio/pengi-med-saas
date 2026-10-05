import { useText } from "@pengi/shared";
import {
	Button,
	Popover,
	PopoverContent,
	PopoverTrigger,
	Spinner,
} from "@pengi/ui";
import { Check, Copy, Link, Monitor } from "lucide-react";
import React from "react";
import {
	type DisplayToken,
	generateDisplayToken,
	getDisplayToken,
	getDisplayTokenQr,
} from "@/api/clinical-service";

/**
 * QR code of the TV link, rendered by the API (the link is a credential, so it
 * never goes to a third-party QR service). Mounted with `key={token}`: a new
 * token remounts it, which fetches the new QR and revokes the old object URL.
 */
export function DisplayQr() {
	const { textGet } = useText();
	const [src, setSrc] = React.useState<string | null>(null);
	const [failed, setFailed] = React.useState(false);

	React.useEffect(() => {
		let objectUrl: string | null = null;
		let cancelled = false;
		getDisplayTokenQr().then((res) => {
			if (cancelled) return;
			if (res.success && res.data) {
				objectUrl = URL.createObjectURL(res.data);
				setSrc(objectUrl);
			} else {
				setFailed(true);
			}
		});
		return () => {
			cancelled = true;
			if (objectUrl) URL.revokeObjectURL(objectUrl);
		};
	}, []);

	if (failed) return null;
	return (
		<div className="flex flex-col items-center gap-2">
			<div className="flex size-44 items-center justify-center rounded-lg border border-border bg-white p-1">
				{src ? (
					<img
						src={src}
						alt={textGet("waiting_room.tv.qr_alt")}
						className="size-full"
					/>
				) : (
					<Spinner />
				)}
			</div>
			<p className="text-center text-xs text-muted-foreground">
				{textGet("waiting_room.tv.qr_hint")}
			</p>
		</div>
	);
}

export function TvScreenPopover() {
	const { textGet } = useText();
	// Token and link both come from the API, so the copied link and the QR
	// always point to the same host (FRONTEND_URL).
	const [display, setDisplay] = React.useState<DisplayToken | null>(null);
	const code = display?.token ?? null;
	const link = display?.display_url ?? null;
	const [loading, setLoading] = React.useState(false);
	const [loadFailed, setLoadFailed] = React.useState(false);
	const [rotating, setRotating] = React.useState(false);
	const [confirmingRotate, setConfirmingRotate] = React.useState(false);
	const [copied, setCopied] = React.useState(false);

	// Reading the token never changes it, so opening the popover or copying the
	// link keeps the TV linked. Only an explicit rotation replaces it.
	const loadCode = async () => {
		setLoading(true);
		setLoadFailed(false);
		const res = await getDisplayToken();
		setLoading(false);
		if (!res.success || !res.data) {
			setLoadFailed(true);
			return;
		}
		setDisplay(res.data);
	};

	const handleOpenChange = (open: boolean) => {
		if (open) {
			loadCode();
		} else {
			setConfirmingRotate(false);
		}
	};

	const rotate = async () => {
		setRotating(true);
		const res = await generateDisplayToken();
		setRotating(false);
		setConfirmingRotate(false);
		if (res.success && res.data) setDisplay(res.data);
	};

	const copyLink = async () => {
		if (!link) return;
		await navigator.clipboard.writeText(link);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<Popover onOpenChange={handleOpenChange}>
			<PopoverTrigger render={<Button variant="outline" />}>
				<Monitor />
				{textGet("waiting_room.tv.title")}
			</PopoverTrigger>
			<PopoverContent align="end" className="w-72 space-y-3 p-4">
				<p className="text-xs text-muted-foreground">
					{textGet("waiting_room.tv.pair_hint")}
				</p>
				{code && <DisplayQr key={code} />}
				<div className="flex h-10 items-center justify-between gap-2 rounded-lg bg-muted px-3">
					{loading && !code ? (
						<Spinner />
					) : link ? (
						<span className="min-w-0 truncate font-mono text-xs" title={link}>
							{link}
						</span>
					) : (
						loadFailed && (
							<span className="text-xs text-muted-foreground">
								{textGet("waiting_room.tv.load_error")}
							</span>
						)
					)}
					{link && (
						<Button
							variant="ghost"
							size="icon-sm"
							aria-label={textGet("waiting_room.tv.copy_code")}
							onClick={copyLink}
						>
							{copied ? <Check className="text-primary" /> : <Copy />}
						</Button>
					)}
				</div>
				<div className="flex flex-col gap-1">
					<Button
						variant="outline"
						size="sm"
						className="w-full"
						onClick={copyLink}
						disabled={!code || rotating}
					>
						{copied ? <Check className="text-primary" /> : <Link />}
						{textGet(
							copied
								? "waiting_room.tv.link_copied"
								: "waiting_room.copy_tv_link",
						)}
					</Button>
					{code && !confirmingRotate && (
						<Button
							variant="ghost"
							size="sm"
							className="w-full text-muted-foreground"
							onClick={() => setConfirmingRotate(true)}
						>
							{textGet("waiting_room.tv.regenerate")}
						</Button>
					)}
				</div>
				{confirmingRotate && (
					<div className="space-y-2 rounded-lg border border-destructive/30 p-3">
						<p className="text-xs">
							{textGet("waiting_room.tv.regenerate_warning")}
						</p>
						<div className="flex flex-col gap-1">
							<Button
								variant="destructive"
								size="sm"
								className="w-full"
								onClick={rotate}
								disabled={rotating}
							>
								{rotating && <Spinner />}
								{textGet("waiting_room.tv.regenerate_confirm")}
							</Button>
							<Button
								variant="ghost"
								size="sm"
								className="w-full"
								onClick={() => setConfirmingRotate(false)}
								disabled={rotating}
							>
								{textGet("common.cancel")}
							</Button>
						</div>
					</div>
				)}
			</PopoverContent>
		</Popover>
	);
}
