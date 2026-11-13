import type { ReactElement, SVGProps } from "react";
import { cn } from "../lib/utils";

/**
 * Spot illustrations for empty / error states.
 *
 * Two-tone style: `--primary` (accent) over soft tints of itself and
 * `--muted-foreground` (ink). Everything is derived from CSS variables, so
 * light/dark and per-app theming work with no extra assets.
 */

const INK = "var(--muted-foreground)";
const ACCENT = "var(--primary)";
const TINT = "color-mix(in oklab, var(--primary) 14%, transparent)";
const TINT_STRONG = "color-mix(in oklab, var(--primary) 28%, transparent)";
const WASH = "color-mix(in oklab, var(--primary) 7%, transparent)";
const SOFT = "color-mix(in oklab, var(--muted-foreground) 14%, transparent)";
const SOFT_STRONG =
	"color-mix(in oklab, var(--muted-foreground) 32%, transparent)";

const stroke = {
	strokeWidth: 2.5,
	strokeLinecap: "round",
	strokeLinejoin: "round",
} as const;

/**
 * Shared backdrop: a faint disc behind the subject plus the ground shadow.
 * Gives every illustration the same depth and footprint.
 */
function Backdrop({ cy = 86, ground = 156 }: { cy?: number; ground?: number }) {
	return (
		<>
			<circle cx="120" cy={cy} r="58" fill={WASH} />
			<ellipse cx="120" cy={ground} rx="74" ry="6" fill={SOFT} />
		</>
	);
}

const CLOUD =
	"M88 90A16 16 0 1 1 92 58.5A24 24 0 0 1 139.7 56A18 18 0 1 1 148 90z";
const FOLDER_FRONT =
	"M82 84h100a6 6 0 0 1 5.8 7.5l-12.6 48.5a8 8 0 0 1-7.8 6H72a8 8 0 0 1-7.8-9.9l11.9-47.6A6 6 0 0 1 82 84z";

/** Unplugged cable under a cloud: something failed, try again. */
function ErrorArt() {
	return (
		<>
			<Backdrop />
			{/* small cloud behind, for depth */}
			<path
				d="M152 54a8 8 0 0 1 4-15 11 11 0 0 1 21 1 7 7 0 0 1 3 14z"
				fill={SOFT}
				stroke={SOFT_STRONG}
				{...stroke}
			/>
			{/* main cloud: three lobes on a flat base (paper underlay hides the back cloud) */}
			<path d={CLOUD} fill="var(--background)" />
			<path d={CLOUD} fill={TINT} stroke={ACCENT} {...stroke} />
			{/* left plug (ink) */}
			<path d="M74 126c-14 0-16 14-32 14" stroke={INK} {...stroke} />
			<rect
				x="74"
				y="114"
				width="26"
				height="24"
				rx="6"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<path d="M100 120h8M100 132h8" stroke={INK} {...stroke} />
			{/* right socket end (accent) */}
			<path d="M166 126c14 0 16 14 32 14" stroke={ACCENT} {...stroke} />
			<rect
				x="140"
				y="114"
				width="26"
				height="24"
				rx="6"
				fill={TINT_STRONG}
				stroke={ACCENT}
				{...stroke}
			/>
			<rect
				x="132"
				y="118"
				width="8"
				height="16"
				rx="2"
				fill="var(--background)"
				stroke={ACCENT}
				{...stroke}
			/>
			{/* spark marks in the gap */}
			<path
				d="M114 116l-4-4M120 113v-6M126 116l4-4"
				stroke={ACCENT}
				{...stroke}
			/>
			{/* accents */}
			<path d="M50 54v12M44 60h12" stroke={ACCENT} {...stroke} />
			<circle cx="194" cy="96" r="2.5" fill={INK} />
		</>
	);
}

/** Open, empty folder with a ghost sheet: nothing here yet. */
function EmptyArt() {
	return (
		<>
			<Backdrop />
			{/* folder back with tab */}
			<path
				d="M64 60a8 8 0 0 1 8-8h26l8 10h62a8 8 0 0 1 8 8v68a8 8 0 0 1-8 8H72a8 8 0 0 1-8-8z"
				fill={TINT}
				stroke={ACCENT}
				{...stroke}
			/>
			{/* placeholder sheet, dashed: content to come */}
			<rect
				x="94"
				y="38"
				width="52"
				height="60"
				rx="5"
				stroke={INK}
				{...stroke}
				strokeDasharray="5 6"
			/>
			{/* folder front, tilted open (paper underlay hides the dashed sheet) */}
			<path d={FOLDER_FRONT} fill="var(--background)" />
			<path d={FOLDER_FRONT} fill={TINT_STRONG} stroke={ACCENT} {...stroke} />
			<path d="M112 118h26" stroke={ACCENT} {...stroke} />
			{/* accents */}
			<path d="M190 42v16M182 50h16" stroke={ACCENT} {...stroke} />
			<circle cx="52" cy="72" r="3.5" fill={ACCENT} />
			<circle cx="200" cy="108" r="2.5" fill={INK} />
		</>
	);
}

/** Magnifier over a page stack: 404 / no results. */
function NotFoundArt() {
	return (
		<>
			<Backdrop />
			{/* page behind, for depth */}
			<rect
				x="84"
				y="36"
				width="62"
				height="88"
				rx="6"
				fill={TINT}
				transform="rotate(8 115 80)"
			/>
			{/* front page with folded corner */}
			<path
				d="M72 36h46l18 18v80a6 6 0 0 1-6 6H72a6 6 0 0 1-6-6V42a6 6 0 0 1 6-6z"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<path
				d="M118 36v12a6 6 0 0 0 6 6h12"
				fill={SOFT}
				stroke={INK}
				{...stroke}
			/>
			<path d="M76 58h24" stroke={ACCENT} {...stroke} />
			<path d="M76 72h40M76 84h32M76 96h22" stroke={INK} {...stroke} />
			{/* magnifier: light glass (paper + faint wash) so the "?" reads clearly */}
			<circle cx="138" cy="104" r="24" fill="var(--background)" />
			<circle
				cx="138"
				cy="104"
				r="24"
				fill={WASH}
				stroke={ACCENT}
				{...stroke}
			/>
			<rect
				x="161"
				y="117"
				width="10"
				height="30"
				rx="5"
				fill={ACCENT}
				stroke={ACCENT}
				{...stroke}
				transform="rotate(-45 166 132)"
			/>
			{/* question mark: larger, heavier glyph centred in the lens */}
			<path
				d="M130 97a8 8 0 1 1 11.3 7.3c-2.4 1.1-3.3 2.7-3.3 5.3v1"
				stroke={ACCENT}
				{...stroke}
				strokeWidth={4.5}
			/>
			<circle cx="138" cy="118.5" r="3" fill={ACCENT} />
			{/* accents */}
			<path d="M48 54v12M42 60h12" stroke={ACCENT} {...stroke} />
			<circle cx="190" cy="62" r="3.5" fill={ACCENT} />
			<circle cx="54" cy="118" r="2.5" fill={INK} />
		</>
	);
}

const FUNNEL =
	"M84 34h72a4 4 0 0 1 3 6.6L132 72v18l-24 10V72L81 40.6A4 4 0 0 1 84 34z";
const TRAY = "M64 118h30l6 10h40l6-10h30v22a6 6 0 0 1-6 6H70a6 6 0 0 1-6-6z";

/** Filter funnel over an empty tray: the search matched nothing. */
function NoResultsArt() {
	return (
		<>
			<Backdrop />
			{/* empty tray (ink): nothing came through */}
			<path d={TRAY} fill="var(--background)" />
			<path d={TRAY} fill={SOFT} stroke={INK} {...stroke} />
			<path d="M100 137h40" stroke={INK} {...stroke} strokeDasharray="4 6" />
			{/* filter funnel (accent) */}
			<path d={FUNNEL} fill="var(--background)" />
			<path d={FUNNEL} fill={TINT_STRONG} stroke={ACCENT} {...stroke} />
			<path d="M94 48h52" stroke={ACCENT} {...stroke} />
			{/* accents */}
			<path d="M48 56v12M42 62h12" stroke={ACCENT} {...stroke} />
			<circle cx="192" cy="66" r="3.5" fill={ACCENT} />
			<circle cx="196" cy="104" r="2.5" fill={INK} />
		</>
	);
}

/** Wi-fi bands fading out with an "off" badge: no connection. */
function NoConnectionArt() {
	return (
		<>
			<Backdrop />
			{/* outer band, faded: signal lost */}
			<path
				d="M74 78A65 65 0 0 1 166 78L158.9 85.1A55 55 0 0 0 81.1 85.1z"
				fill={SOFT}
				stroke={SOFT_STRONG}
				{...stroke}
			/>
			<path
				d="M87.5 91.5A46 46 0 0 1 152.5 91.5L145.5 98.5A36 36 0 0 0 94.5 98.5z"
				fill={TINT}
				stroke={ACCENT}
				{...stroke}
			/>
			<path
				d="M100.9 104.9A27 27 0 0 1 139.1 104.9L132 112A17 17 0 0 0 108 112z"
				fill={TINT_STRONG}
				stroke={ACCENT}
				{...stroke}
			/>
			<circle cx="120" cy="124" r="6" fill={ACCENT} />
			{/* off badge (ink) */}
			<circle
				cx="164"
				cy="118"
				r="15"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<path d="M158 112l12 12M170 112l-12 12" stroke={INK} {...stroke} />
			{/* accents */}
			<path d="M48 50v12M42 56h12" stroke={ACCENT} {...stroke} />
			<circle cx="192" cy="60" r="3.5" fill={ACCENT} />
			<circle cx="60" cy="120" r="2.5" fill={INK} />
		</>
	);
}

const SHIELD = "M120 34l42 14v34c0 28-18 48-42 58c-24-10-42-30-42-58V48z";

/** Padlock on a shield: you don't have access to this. */
function NoAccessArt() {
	return (
		<>
			<Backdrop />
			<path d={SHIELD} fill="var(--background)" />
			<path d={SHIELD} fill={TINT} stroke={ACCENT} {...stroke} />
			{/* padlock (ink) */}
			<path d="M108 86V76a12 12 0 0 1 24 0v10" stroke={INK} {...stroke} />
			<rect
				x="100"
				y="86"
				width="40"
				height="32"
				rx="6"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<circle cx="120" cy="99" r="4" fill={INK} />
			<path d="M120 102v7" stroke={INK} {...stroke} />
			{/* accents */}
			<path d="M186 46v12M180 52h12" stroke={ACCENT} {...stroke} />
			<circle cx="54" cy="70" r="3.5" fill={ACCENT} />
			<circle cx="190" cy="112" r="2.5" fill={INK} />
		</>
	);
}

/** Dimmed panel with a lock badge and an upward arrow: available on a higher plan. */
function LockedFeatureArt() {
	return (
		<>
			<Backdrop />
			{/* feature panel, content dimmed */}
			<rect
				x="62"
				y="44"
				width="104"
				height="84"
				rx="8"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<path d="M76 62h36" stroke={ACCENT} {...stroke} />
			<rect x="76" y="96" width="12" height="18" rx="2" fill={SOFT_STRONG} />
			<rect x="96" y="82" width="12" height="32" rx="2" fill={SOFT_STRONG} />
			<rect x="116" y="90" width="12" height="24" rx="2" fill={SOFT_STRONG} />
			{/* lock badge */}
			<circle cx="164" cy="124" r="22" fill="var(--background)" />
			<circle
				cx="164"
				cy="124"
				r="22"
				fill={TINT}
				stroke={ACCENT}
				{...stroke}
			/>
			<path d="M157 121v-4a7 7 0 0 1 14 0v4" stroke={ACCENT} {...stroke} />
			<rect
				x="152"
				y="121"
				width="24"
				height="17"
				rx="4"
				fill={ACCENT}
				stroke={ACCENT}
				{...stroke}
			/>
			<circle cx="164" cy="129.5" r="2.5" fill="var(--background)" />
			{/* upgrade hint */}
			<path d="M196 96V70M187 79l9-9 9 9" stroke={ACCENT} {...stroke} />
			{/* accents */}
			<path d="M44 58v12M38 64h12" stroke={ACCENT} {...stroke} />
			<circle cx="50" cy="120" r="2.5" fill={INK} />
		</>
	);
}

/** Tidy clipboard with every row ticked and a check badge: all caught up. */
function AllClearArt() {
	return (
		<>
			<Backdrop />
			{/* board */}
			<rect
				x="80"
				y="40"
				width="76"
				height="100"
				rx="8"
				fill={TINT}
				stroke={ACCENT}
				{...stroke}
			/>
			{/* sheet */}
			<rect
				x="90"
				y="52"
				width="56"
				height="80"
				rx="4"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			{/* clip */}
			<rect
				x="104"
				y="32"
				width="28"
				height="14"
				rx="4"
				fill={TINT_STRONG}
				stroke={ACCENT}
				{...stroke}
			/>
			{/* ticked rows */}
			<path
				d="M98 72l3 3 6-6M98 90l3 3 6-6M98 108l3 3 6-6"
				stroke={ACCENT}
				{...stroke}
			/>
			<path d="M114 72h24M114 90h20M114 108h16" stroke={INK} {...stroke} />
			{/* check badge */}
			<circle
				cx="158"
				cy="122"
				r="20"
				fill={ACCENT}
				stroke={ACCENT}
				{...stroke}
			/>
			<path
				d="M149 122l6 6 12-12"
				stroke="var(--background)"
				{...stroke}
				strokeWidth={3.5}
			/>
			{/* accents */}
			<path d="M52 50v12M46 56h12" stroke={ACCENT} {...stroke} />
			<circle cx="194" cy="70" r="3.5" fill={ACCENT} />
			<circle cx="54" cy="116" r="2.5" fill={INK} />
		</>
	);
}

/** Big check in a halo with confetti: done. */
function SuccessArt() {
	return (
		<>
			<Backdrop />
			<circle cx="120" cy="86" r="42" fill={TINT} />
			<circle
				cx="120"
				cy="86"
				r="30"
				fill={ACCENT}
				stroke={ACCENT}
				{...stroke}
			/>
			<path
				d="M106 86l9 9 19-19"
				stroke="var(--background)"
				{...stroke}
				strokeWidth={5}
			/>
			{/* confetti */}
			<path d="M62 44v12M56 50h12" stroke={ACCENT} {...stroke} />
			<path d="M186 108v8M182 112h8" stroke={ACCENT} {...stroke} />
			<circle cx="180" cy="50" r="3.5" fill={ACCENT} />
			<circle cx="56" cy="106" r="2.5" fill={INK} />
			<circle cx="200" cy="80" r="2" fill={INK} />
			<path d="M72 80l-8-3M168 70l8-4" stroke={ACCENT} {...stroke} />
			<path d="M82 130l-6 5M160 132l6 5" stroke={INK} {...stroke} />
		</>
	);
}

/** Envelope with a clock badge: the link has expired. */
function LinkExpiredArt() {
	return (
		<>
			<Backdrop />
			{/* envelope (ink) */}
			<rect
				x="58"
				y="50"
				width="100"
				height="68"
				rx="8"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<path d="M62 56l46 32 46-32" stroke={INK} {...stroke} />
			{/* clock badge (accent) */}
			<circle cx="162" cy="118" r="24" fill="var(--background)" />
			<circle
				cx="162"
				cy="118"
				r="24"
				fill={TINT}
				stroke={ACCENT}
				{...stroke}
			/>
			<path d="M162 104v14h10" stroke={ACCENT} {...stroke} />
			{/* accents */}
			<path d="M196 52v12M190 58h12" stroke={ACCENT} {...stroke} />
			<circle cx="46" cy="100" r="3.5" fill={ACCENT} />
			<circle cx="52" cy="68" r="2.5" fill={INK} />
		</>
	);
}

/** Two-unit server, one idle, with a wrench badge: down for maintenance, back soon. */
function MaintenanceArt() {
	return (
		<>
			<Backdrop />
			{/* top unit: running */}
			<rect
				x="64"
				y="42"
				width="84"
				height="38"
				rx="7"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<circle cx="78" cy="61" r="4" fill={ACCENT} />
			<path d="M92 61h22M128 55v12M136 55v12" stroke={INK} {...stroke} />
			{/* bottom unit: idle */}
			<rect
				x="64"
				y="88"
				width="84"
				height="38"
				rx="7"
				fill={SOFT}
				stroke={INK}
				{...stroke}
			/>
			<circle cx="78" cy="107" r="4" fill={SOFT_STRONG} />
			<path d="M92 107h22" stroke={INK} {...stroke} />
			{/* wrench badge */}
			<circle cx="160" cy="120" r="24" fill="var(--background)" />
			<circle
				cx="160"
				cy="120"
				r="24"
				fill={TINT}
				stroke={ACCENT}
				{...stroke}
			/>
			<g transform="translate(160 121) rotate(45)">
				<circle cx="0" cy="-8" r="8" fill={ACCENT} />
				<rect x="-3" y="-4" width="6" height="18" rx="3" fill={ACCENT} />
				{/* open jaw: notch in the head, painted like the badge */}
				<rect
					x="-3"
					y="-17"
					width="6"
					height="9"
					rx="1"
					fill="var(--background)"
				/>
				<rect x="-3" y="-17" width="6" height="9" rx="1" fill={TINT} />
			</g>
			{/* accents */}
			<path d="M46 48v12M40 54h12" stroke={ACCENT} {...stroke} />
			<circle cx="190" cy="58" r="3.5" fill={ACCENT} />
			<circle cx="50" cy="118" r="2.5" fill={INK} />
		</>
	);
}

/** Calendar with a circled due day, a payment card and a renew badge: plan expired, renew. */
function PlanExpiredArt() {
	return (
		<>
			<Backdrop />
			{/* calendar: paper body, tinted header, ink outline */}
			<rect
				x="64"
				y="46"
				width="88"
				height="82"
				rx="8"
				fill="var(--background)"
			/>
			<path d="M64 54a8 8 0 0 1 8-8h72a8 8 0 0 1 8 8v12H64z" fill={TINT} />
			<rect
				x="64"
				y="46"
				width="88"
				height="82"
				rx="8"
				stroke={INK}
				{...stroke}
			/>
			<path d="M64 66h88" stroke={INK} {...stroke} />
			<path d="M86 40v12M130 40v12" stroke={ACCENT} {...stroke} />
			{/* day grid */}
			<g fill={SOFT_STRONG}>
				<rect x="76" y="76" width="8" height="8" rx="2" />
				<rect x="94" y="76" width="8" height="8" rx="2" />
				<rect x="112" y="76" width="8" height="8" rx="2" />
				<rect x="130" y="76" width="8" height="8" rx="2" />
				<rect x="76" y="94" width="8" height="8" rx="2" />
				<rect x="94" y="94" width="8" height="8" rx="2" />
				<rect x="130" y="94" width="8" height="8" rx="2" />
				<rect x="76" y="112" width="8" height="8" rx="2" />
				<rect x="94" y="112" width="8" height="8" rx="2" />
				<rect x="112" y="112" width="8" height="8" rx="2" />
			</g>
			{/* due day, circled */}
			<rect x="112" y="94" width="8" height="8" rx="2" fill={ACCENT} />
			<circle cx="116" cy="98" r="9" stroke={ACCENT} {...stroke} />
			{/* payment card */}
			<rect
				x="128"
				y="102"
				width="60"
				height="38"
				rx="6"
				fill="var(--background)"
			/>
			<rect
				x="128"
				y="102"
				width="60"
				height="38"
				rx="6"
				fill={TINT_STRONG}
				stroke={ACCENT}
				{...stroke}
			/>
			<path d="M128 114h60" stroke={ACCENT} {...stroke} />
			<rect x="138" y="122" width="12" height="9" rx="2" fill={ACCENT} />
			{/* renew badge */}
			<circle
				cx="186"
				cy="104"
				r="11"
				fill="var(--background)"
				stroke={INK}
				{...stroke}
			/>
			<path
				d="M192 104a6 6 0 1 1-6-6M183 95l3 3-3 3"
				stroke={INK}
				{...stroke}
			/>
			{/* accents */}
			<path d="M46 54v12M40 60h12" stroke={ACCENT} {...stroke} />
			<circle cx="52" cy="112" r="3.5" fill={ACCENT} />
			<circle cx="202" cy="64" r="2.5" fill={INK} />
		</>
	);
}

const ARTS = {
	error: ErrorArt,
	empty: EmptyArt,
	"not-found": NotFoundArt,
	"no-results": NoResultsArt,
	"no-connection": NoConnectionArt,
	"no-access": NoAccessArt,
	"locked-feature": LockedFeatureArt,
	"all-clear": AllClearArt,
	success: SuccessArt,
	"link-expired": LinkExpiredArt,
	maintenance: MaintenanceArt,
	"plan-expired": PlanExpiredArt,
} satisfies Record<string, () => ReactElement>;

export type IllustrationName = keyof typeof ARTS;

/** Every registered illustration, for galleries and tests. */
export const ILLUSTRATION_NAMES = Object.keys(ARTS) as IllustrationName[];

export interface IllustrationProps
	extends Omit<SVGProps<SVGSVGElement>, "name"> {
	name: IllustrationName;
}

/** Decorative by default (`aria-hidden`); pass `aria-label` to expose it. */
export function Illustration({ name, className, ...props }: IllustrationProps) {
	const Art = ARTS[name];
	const labelled = props["aria-label"] !== undefined;
	return (
		// biome-ignore lint/a11y/noSvgWithoutTitle: decorative (aria-hidden) unless the caller passes aria-label
		<svg
			viewBox="0 0 240 180"
			fill="none"
			role={labelled ? "img" : undefined}
			aria-hidden={labelled ? undefined : true}
			className={cn("h-auto w-48", className)}
			{...props}
		>
			<Art />
		</svg>
	);
}
