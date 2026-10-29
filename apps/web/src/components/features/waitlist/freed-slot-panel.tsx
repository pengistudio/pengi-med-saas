import { useText } from "@pengi/shared";
import {
	Badge,
	Button,
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
	Skeleton,
	Text,
} from "@pengi/ui";
import { MessageCircle, Phone, PhoneCall, X } from "lucide-react";
import React from "react";
import {
	getWaitlistCandidates,
	type OfferChannel,
	offerWaitlistSlot,
	type WaitlistCandidates,
	type WaitlistEntry,
	type WaitlistSlot,
	waitlistPatientName,
} from "@/api/waitlist-service";
import { entryPreference } from "./waitlist-utils";

type State =
	| { status: "loading" }
	| { status: "error" }
	| { status: "ready"; data: WaitlistCandidates };

/**
 * A freed slot (from the notification's link) and the waitlist entries it
 * matches, in signup order. Reception offers it one patient at a time: by
 * WhatsApp when the clinic can (plan, connection, approved template) and
 * the patient agreed, or by phone ("Llamar" shows the number, then marks it
 * offered). Nothing is offered on its own.
 */
export function FreedSlotPanel({
	slot,
	canManage,
	onClose,
	onOffered,
}: {
	slot: WaitlistSlot;
	/** MANAGE_APPOINTMENT: the offer buttons show. */
	canManage: boolean;
	onClose: () => void;
	/** An entry was offered the slot (the list may reload). */
	onOffered?: () => void;
}) {
	const { textGet, formatDate } = useText();
	const [state, setState] = React.useState<State>({ status: "loading" });
	const [version, setVersion] = React.useState(0);
	const [busy, setBusy] = React.useState<number>();
	const [calling, setCalling] = React.useState<number>();

	React.useEffect(() => {
		let active = true;
		getWaitlistCandidates(slot).then((res) => {
			if (!active) return;
			setState(
				res.success && res.data
					? { status: "ready", data: res.data }
					: { status: "error" },
			);
		});
		return () => {
			active = false;
		};
	}, [slot, version]);

	const offer = async (entry: WaitlistEntry, channel: OfferChannel) => {
		setBusy(entry.id);
		const res = await offerWaitlistSlot(entry.id, slot, channel);
		setBusy(undefined);
		if (res.success) {
			setCalling(undefined);
			setVersion((v) => v + 1);
			onOffered?.();
		}
	};

	const when = `${formatDate(slot.date, "weekday-day-month")} ${slot.start_time}${
		slot.end_time ? `–${slot.end_time}` : ""
	}`;
	const doctor = state.status === "ready" ? state.data.doctor : "";

	return (
		<Card>
			<CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
				<div className="space-y-1">
					<CardTitle className="first-letter:uppercase">
						{textGet("waitlist.slot.title", { when })}
					</CardTitle>
					<CardDescription>
						{doctor
							? textGet("waitlist.slot.with_doctor", { doctor })
							: textGet("waitlist.slot.description")}
					</CardDescription>
				</div>
				<Button
					variant="ghost"
					size="icon"
					aria-label={textGet("waitlist.slot.close")}
					onClick={onClose}
				>
					<X className="h-4 w-4" />
				</Button>
			</CardHeader>
			<CardContent>
				{state.status === "loading" && (
					<div className="space-y-2">
						<Skeleton className="h-14 w-full" />
						<Skeleton className="h-14 w-full" />
					</div>
				)}
				{state.status === "error" && (
					<p className="text-sm text-muted-foreground">
						<Text uuid="waitlist.slot.error" />
					</p>
				)}
				{state.status === "ready" && state.data.candidates.length === 0 && (
					<p className="text-sm text-muted-foreground">
						<Text uuid="waitlist.slot.empty" />
					</p>
				)}
				{state.status === "ready" && state.data.candidates.length > 0 && (
					<ol className="divide-y">
						{state.data.candidates.map((entry, index) => {
							const offered = Boolean(entry.offered_this_slot);
							const accepted = offered && Boolean(entry.offer?.accepted_at);
							const isBusy = busy === entry.id;
							return (
								<li
									key={entry.id}
									className="flex flex-col gap-3 py-3 sm:flex-row sm:items-center sm:justify-between"
								>
									<div className="flex min-w-0 items-start gap-3">
										<span className="mt-0.5 w-5 shrink-0 text-right text-sm text-muted-foreground tabular-nums">
											{index + 1}.
										</span>
										<div className="min-w-0">
											<p className="font-medium">
												{waitlistPatientName(entry)}
											</p>
											<p className="text-sm text-muted-foreground">
												{entryPreference(entry, textGet, formatDate)}
											</p>
											{calling === entry.id && (
												<p className="mt-1 text-sm">
													{entry.patient.phone ? (
														<a
															href={`tel:${entry.patient.phone}`}
															className="font-medium tabular-nums underline-offset-4 hover:underline"
														>
															{entry.patient.phone}
														</a>
													) : (
														<Text uuid="waitlist.slot.no_phone" />
													)}
												</p>
											)}
										</div>
									</div>
									<div className="flex flex-wrap items-center gap-2 sm:justify-end">
										{offered && (
											<Badge variant={accepted ? "default" : "secondary"}>
												{accepted ? (
													<Text uuid="waitlist.offer.accepted" />
												) : (
													<Text uuid="waitlist.offer.offered" />
												)}
											</Badge>
										)}
										{canManage && !offered && (
											<>
												{state.data.whatsapp_available && (
													<Button
														size="sm"
														disabled={isBusy || !entry.can_whatsapp}
														title={
															entry.can_whatsapp
																? undefined
																: textGet("waitlist.offer.whatsapp_unavailable")
														}
														onClick={() => offer(entry, "whatsapp")}
													>
														<MessageCircle className="h-4 w-4" />
														<Text uuid="waitlist.offer.whatsapp" />
													</Button>
												)}
												{calling === entry.id ? (
													<Button
														size="sm"
														variant="outline"
														disabled={isBusy}
														onClick={() => offer(entry, "phone")}
													>
														<PhoneCall className="h-4 w-4" />
														<Text uuid="waitlist.offer.mark_called" />
													</Button>
												) : (
													<Button
														size="sm"
														variant="outline"
														onClick={() => setCalling(entry.id)}
													>
														<Phone className="h-4 w-4" />
														<Text uuid="waitlist.offer.call" />
													</Button>
												)}
											</>
										)}
									</div>
								</li>
							);
						})}
					</ol>
				)}
			</CardContent>
		</Card>
	);
}
