import { useText } from "@pengi/shared";
import { Button, Spinner } from "@pengi/ui";
import { BellRing, Check } from "lucide-react";
import React from "react";
import {
	type BookingTarget,
	getWaitlist,
	joinWaitlist,
	leaveWaitlist,
	type WaitlistEntry,
} from "@/api/portal-service";

type State =
	| { status: "loading" }
	| { status: "idle" }
	| { status: "joined"; entry: WaitlistEntry };

const doctorOf = (target: BookingTarget) =>
	"doctor_id" in target ? target.doctor_id : undefined;

/** The patient's active entry for this type and doctor (any doctor in specialty mode). */
const sameRequest = (
	entry: WaitlistEntry,
	typeId: number,
	doctorId: number | undefined,
) => entry.type_id === typeId && (entry.doctor_id ?? undefined) === doctorId;

/**
 * "Avísame si se libera un turno": when the chosen doctor (or specialty) has
 * no free time for the type, the signed-in patient joins the clinic's
 * waitlist; reception contacts them when a slot frees. Once joined, they can
 * leave it.
 */
export function WaitlistButton({
	typeId,
	target,
}: {
	typeId: number;
	target: BookingTarget;
}) {
	const { textGet } = useText();
	const doctorId = doctorOf(target);
	const [state, setState] = React.useState<State>({ status: "loading" });
	const [busy, setBusy] = React.useState(false);

	React.useEffect(() => {
		let active = true;
		getWaitlist().then((res) => {
			if (!active) return;
			const entry = res.success
				? (res.data ?? []).find((e) => sameRequest(e, typeId, doctorId))
				: undefined;
			setState(entry ? { status: "joined", entry } : { status: "idle" });
		});
		return () => {
			active = false;
		};
	}, [typeId, doctorId]);

	if (state.status === "loading") return null;

	if (state.status === "joined") {
		return (
			<div className="space-y-3 rounded-2xl bg-secondary/60 px-4 py-3">
				<p className="flex items-start gap-2 text-sm">
					<Check className="mt-0.5 size-4 shrink-0 text-primary" />
					{textGet("portal.waitlist.joined")}
				</p>
				<Button
					type="button"
					variant="ghost"
					size="sm"
					className="rounded-full"
					disabled={busy}
					onClick={async () => {
						setBusy(true);
						const res = await leaveWaitlist(state.entry.id);
						setBusy(false);
						if (res.success) setState({ status: "idle" });
					}}
				>
					{busy && <Spinner />}
					{textGet("portal.waitlist.leave")}
				</Button>
			</div>
		);
	}

	return (
		<Button
			type="button"
			className="rounded-full"
			disabled={busy}
			onClick={async () => {
				setBusy(true);
				const res = await joinWaitlist(typeId, doctorId);
				setBusy(false);
				if (res.success && res.data) {
					setState({ status: "joined", entry: res.data });
				}
			}}
		>
			{busy ? <Spinner /> : <BellRing />}
			{textGet("portal.waitlist.join")}
		</Button>
	);
}
