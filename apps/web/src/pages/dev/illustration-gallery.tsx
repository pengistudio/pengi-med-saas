import {
	Button,
	Card,
	CardContent,
	ILLUSTRATION_NAMES,
	Illustration,
} from "@pengi/ui";
import { useState } from "react";

// Dev-only tool (route registered only when import.meta.env.DEV). Strings are
// intentionally not i18n keys: it never ships to production.
const SIZES = [96, 192, 288] as const;

export default function IllustrationGallery() {
	const [dark, setDark] = useState(false);
	const [size, setSize] = useState<(typeof SIZES)[number]>(192);

	return (
		<div
			className={`${dark ? "dark" : ""} min-h-screen bg-background text-foreground`}
		>
			<div className="mx-auto max-w-5xl space-y-6 p-6">
				<div className="flex flex-wrap items-center justify-between gap-3">
					<div>
						<h1 className="text-2xl font-bold">Illustrations</h1>
						<p className="text-sm text-muted-foreground">
							{ILLUSTRATION_NAMES.length} registered in @pengi/ui. Dev only.
						</p>
					</div>
					<div className="flex flex-wrap gap-2">
						{SIZES.map((s) => (
							<Button
								key={s}
								size="sm"
								variant={size === s ? "default" : "outline"}
								onClick={() => setSize(s)}
							>
								{s}px
							</Button>
						))}
						<Button size="sm" variant="outline" onClick={() => setDark(!dark)}>
							{dark ? "Light" : "Dark"}
						</Button>
					</div>
				</div>

				<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{ILLUSTRATION_NAMES.map((name) => (
						<Card key={name}>
							<CardContent className="flex flex-col items-center gap-3 p-6">
								<Illustration
									name={name}
									style={{ width: size }}
									className="max-w-full"
								/>
								<code className="text-xs text-muted-foreground">{name}</code>
							</CardContent>
						</Card>
					))}
				</div>

				<h2 className="pt-4 text-lg font-semibold">In context</h2>
				<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{ILLUSTRATION_NAMES.map((name) => (
						<Card key={name}>
							<CardContent className="flex flex-col items-center gap-2 p-8 text-center">
								<Illustration name={name} />
								<p className="font-semibold">Sample title</p>
								<p className="text-sm text-muted-foreground">
									Short sample description of what happened and what to do next.
								</p>
								<Button size="sm" className="mt-2">
									Action
								</Button>
							</CardContent>
						</Card>
					))}
				</div>
			</div>
		</div>
	);
}
