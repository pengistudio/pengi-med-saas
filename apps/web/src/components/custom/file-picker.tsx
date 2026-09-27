import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Paperclip } from "lucide-react";
import { useEffect, useId, useRef } from "react";
import { cn } from "@/lib/utils";

interface FilePickerProps {
	/** The chosen file; the picker shows its name. Pass `null` to clear it. */
	file: File | null | undefined;
	onChange: (file: File | null) => void;
	accept?: string;
	disabled?: boolean;
	id?: string;
	className?: string;
	"aria-invalid"?: boolean;
	"aria-describedby"?: string;
}

/**
 * File input with translated labels: a button that opens the native picker and
 * the chosen file's name, instead of the browser's "Choose file" text.
 * Controlled: the parent owns the file, so resetting a form clears the name.
 */
export function FilePicker({
	file,
	onChange,
	accept,
	disabled,
	id,
	className,
	"aria-invalid": ariaInvalid,
	"aria-describedby": ariaDescribedBy,
}: FilePickerProps) {
	const { textGet } = useText();
	const inputRef = useRef<HTMLInputElement>(null);
	const generatedId = useId();
	const buttonId = id ?? generatedId;
	const nameId = `${buttonId}-name`;

	// Once the parent clears the file, clear the input too, so picking the same
	// file again still fires a change.
	useEffect(() => {
		if (!file && inputRef.current) inputRef.current.value = "";
	}, [file]);

	return (
		<div className={cn("flex min-w-0 items-center gap-3", className)}>
			<input
				ref={inputRef}
				type="file"
				accept={accept}
				disabled={disabled}
				tabIndex={-1}
				aria-hidden="true"
				className="sr-only"
				onChange={(event) => onChange(event.target.files?.[0] ?? null)}
			/>
			<Button
				id={buttonId}
				type="button"
				variant="outline"
				disabled={disabled}
				aria-invalid={ariaInvalid}
				aria-describedby={cn(nameId, ariaDescribedBy)}
				onClick={() => inputRef.current?.click()}
			>
				<Paperclip className="mr-2 h-4 w-4" />
				{textGet("common.file_picker.choose")}
			</Button>
			<span
				id={nameId}
				className={cn(
					"min-w-0 truncate text-sm",
					file ? "text-foreground" : "text-muted-foreground",
				)}
			>
				{file ? file.name : textGet("common.file_picker.none")}
			</span>
		</div>
	);
}
