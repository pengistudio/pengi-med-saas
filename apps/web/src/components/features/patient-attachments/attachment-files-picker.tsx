import { useText } from "@pengi/shared";
import { Button } from "@pengi/ui";
import { Camera, Paperclip, X } from "lucide-react";
import { useRef } from "react";
import { ATTACHMENT_ACCEPT, attachmentFileError } from "@/lib/attachment-files";

interface AttachmentFilesPickerProps {
	id?: string;
	files: File[];
	onChange: (files: File[]) => void;
}

/**
 * Picks several files, plus a "take photo" control that opens the phone's
 * camera (shown on small screens only). Each file shows its own validation
 * error; invalid ones are reported again, not uploaded, when the batch runs.
 */
export function AttachmentFilesPicker({
	id,
	files,
	onChange,
}: AttachmentFilesPickerProps) {
	const { textGet, formatFileSize } = useText();
	const fileRef = useRef<HTMLInputElement>(null);
	const cameraRef = useRef<HTMLInputElement>(null);

	const add = (list: FileList | null) => {
		if (list && list.length > 0) onChange([...files, ...Array.from(list)]);
	};

	return (
		<div className="space-y-2">
			<div className="flex flex-wrap gap-2">
				<input
					ref={fileRef}
					type="file"
					multiple
					accept={ATTACHMENT_ACCEPT}
					aria-hidden="true"
					tabIndex={-1}
					className="sr-only"
					data-testid="attachment-file-input"
					onChange={(e) => {
						add(e.target.files);
						e.target.value = "";
					}}
				/>
				<input
					ref={cameraRef}
					type="file"
					accept="image/*"
					capture="environment"
					aria-hidden="true"
					tabIndex={-1}
					className="sr-only"
					data-testid="attachment-camera-input"
					onChange={(e) => {
						add(e.target.files);
						e.target.value = "";
					}}
				/>
				<Button
					id={id}
					type="button"
					variant="outline"
					onClick={() => fileRef.current?.click()}
				>
					<Paperclip className="mr-2 h-4 w-4" />
					{textGet("clinical.attachment.form.choose_files")}
				</Button>
				<Button
					type="button"
					variant="outline"
					className="md:hidden"
					onClick={() => cameraRef.current?.click()}
				>
					<Camera className="mr-2 h-4 w-4" />
					{textGet("clinical.attachment.form.take_photo")}
				</Button>
			</div>
			{files.length > 0 && (
				<ul className="space-y-1">
					{files.map((file, index) => {
						const error = attachmentFileError(file);
						return (
							<li
								key={`${file.name}-${file.size}-${file.lastModified}-${index}`}
								className="flex items-center gap-2 text-sm"
							>
								<span className="min-w-0 flex-1 truncate">{file.name}</span>
								<span className="shrink-0 text-xs text-muted-foreground">
									{formatFileSize(file.size)}
								</span>
								{error && (
									<span className="shrink-0 text-xs text-destructive">
										{textGet(error)}
									</span>
								)}
								<Button
									type="button"
									variant="ghost"
									size="icon"
									aria-label={textGet("clinical.attachment.form.remove_file")}
									onClick={() => onChange(files.filter((_, i) => i !== index))}
								>
									<X className="h-4 w-4" />
								</Button>
							</li>
						);
					})}
				</ul>
			)}
		</div>
	);
}
