import { useText } from "@pengi/shared";
import { Button, Checkbox, Label } from "@pengi/ui";
import { Check, Copy, Download } from "lucide-react";
import React from "react";

/**
 * The recovery codes, shown once: copy, download as a text file and an
 * explicit "I saved them" before the caller lets the user leave.
 */
export function RecoveryCodesPanel({
	codes,
	onDone,
}: {
	codes: string[];
	onDone: () => void;
}) {
	const { textGet } = useText();
	const [saved, setSaved] = React.useState(false);
	const [copied, setCopied] = React.useState(false);
	const text = codes.join("\n");

	async function copy() {
		await navigator.clipboard.writeText(text);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	}

	function download() {
		const content = `${textGet("profile.two_factor.codes.file_title")}\n\n${text}\n`;
		const url = URL.createObjectURL(
			new Blob([content], { type: "text/plain;charset=utf-8" }),
		);
		const a = document.createElement("a");
		a.href = url;
		a.download = `${textGet("profile.two_factor.codes.file_name")}.txt`;
		a.click();
		URL.revokeObjectURL(url);
	}

	return (
		<div className="space-y-4">
			<p className="text-sm text-muted-foreground">
				{textGet("profile.two_factor.codes.description")}
			</p>
			<ul
				aria-label={textGet("profile.two_factor.codes.title")}
				className="grid grid-cols-2 gap-2 rounded-lg border bg-muted/50 p-4 font-mono text-sm"
			>
				{codes.map((code) => (
					<li key={code} data-testid="recovery-code">
						{code}
					</li>
				))}
			</ul>
			<div className="flex flex-wrap gap-2">
				<Button type="button" variant="outline" onClick={copy}>
					{copied ? (
						<Check className="h-4 w-4" />
					) : (
						<Copy className="h-4 w-4" />
					)}
					{textGet(
						copied
							? "profile.two_factor.codes.copied"
							: "profile.two_factor.codes.copy",
					)}
				</Button>
				<Button type="button" variant="outline" onClick={download}>
					<Download className="h-4 w-4" />
					{textGet("profile.two_factor.codes.download")}
				</Button>
			</div>
			<div className="flex items-center gap-2">
				<Checkbox
					id="recovery-codes-saved"
					checked={saved}
					onCheckedChange={(checked) => setSaved(checked === true)}
				/>
				<Label htmlFor="recovery-codes-saved" className="font-normal">
					{textGet("profile.two_factor.codes.confirm")}
				</Label>
			</div>
			<div className="flex justify-end">
				<Button type="button" disabled={!saved} onClick={onDone}>
					{textGet("profile.two_factor.codes.done")}
				</Button>
			</div>
		</div>
	);
}
