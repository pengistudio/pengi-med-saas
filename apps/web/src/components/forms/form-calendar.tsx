import { format } from "date-fns";
import { CalendarIcon, Clock } from "lucide-react";
import {
	Controller,
	type FieldValues,
	type Path,
	type UseFormReturn,
} from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
} from "@/components/ui/field";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useText } from "@/hooks/use-text";
import { cn } from "@/lib/utils";

// Generate time slots (e.g., 08:00, 09:00, ..., 20:00)
const generateTimeSlots = (startHour = 8, endHour = 20, interval = 1) => {
	const slots = [];
	for (let i = startHour; i <= endHour; i += interval) {
		const hourString = i.toString().padStart(2, "0");
		slots.push(`${hourString}:00`);
	}
	return slots;
};

const TIME_SLOTS = generateTimeSlots();

type FormCalendarProps<
	T extends Record<string, unknown>,
	Input extends FieldValues = T,
> = {
	field: UseFormReturn<Input>;
	name: Path<Input>;
	label?: string;
	isOptional?: boolean;
	description?: string;
	enableTime?: boolean;
	className?: string;
	showMonthYearDropdowns?: boolean;
};

// Represents the value object when time is enabled
export type FormCalendarTimeValue = {
	date?: Date;
	startTime?: string;
	endTime?: string;
};

function FormCalendar<
	T extends Record<string, unknown>,
	Input extends FieldValues = T,
>({
	name,
	field,
	isOptional,
	label,
	description,
	enableTime = false,
	className,
	showMonthYearDropdowns = false,
}: FormCalendarProps<T, Input>) {
	const { textGet } = useText();

	return (
		<Controller
			control={field.control}
			name={name}
			render={({ field: inputField, fieldState }) => {
				// Handle both Date (when !enableTime) and FormCalendarTimeValue (when enableTime)
				const value = inputField.value;
				const isTimeObj =
					enableTime &&
					value &&
					typeof value === "object" &&
					value !== null &&
					!("getTime" in value);

				const selectedDate = isTimeObj
					? (value as FormCalendarTimeValue).date
					: (value as Date | undefined);
				const startTime = isTimeObj
					? (value as FormCalendarTimeValue).startTime
					: undefined;
				const endTime = isTimeObj
					? (value as FormCalendarTimeValue).endTime
					: undefined;

				const handleDateSelect = (date: Date | undefined) => {
					if (enableTime) {
						inputField.onChange({
							...(isTimeObj ? value : {}),
							date,
						});
					} else {
						inputField.onChange(date);
					}
				};

				const handleTimeSelect = (
					type: "start" | "end",
					timeStr: string | null,
				) => {
					if (!enableTime || !timeStr) return;
					const currentVal = isTimeObj ? value : { date: selectedDate };
					inputField.onChange({
						...currentVal,
						[type === "start" ? "startTime" : "endTime"]: timeStr,
					});
				};

				return (
					<Field
						data-invalid={fieldState.invalid}
						className={cn("flex flex-col", className)}
					>
						{label && (
							<FieldLabel htmlFor={name}>
								{label}{" "}
								{isOptional && (
									<span className="text-xs text-muted-foreground font-normal">
										({textGet("form.optional") || "opcional"})
									</span>
								)}
							</FieldLabel>
						)}
						<Popover>
							<PopoverTrigger
								render={
									<Button
										id={name}
										variant={"outline"}
										className={cn(
											"w-full justify-start text-left font-normal",
											!selectedDate && "text-muted-foreground",
											fieldState.invalid &&
												"border-destructive focus-visible:ring-destructive",
										)}
									/>
								}
							>
								<CalendarIcon className="mr-2 h-4 w-4" />
								{selectedDate ? (
									format(selectedDate, "PPP")
								) : (
									<span>
										{textGet("form.calendar.pick_date") || "Pick a date"}
									</span>
								)}
							</PopoverTrigger>
							<PopoverContent className="w-auto p-0" align="start">
								<Calendar
									mode="single"
									selected={selectedDate}
									onSelect={handleDateSelect}
									initialFocus
									captionLayout={showMonthYearDropdowns ? "dropdown" : "label"}
								/>
								{enableTime && (
									<div className="p-3 border-t border-border bg-muted/30">
										<div className="flex items-center gap-2">
											<Clock className="w-4 h-4 text-muted-foreground" />
											<span className="text-sm font-medium">
												{textGet("form.calendar.time_interval") ||
													"Time Interval"}
											</span>
										</div>
										<div className="grid grid-cols-2 gap-2 mt-2">
											<div className="space-y-1">
												<span className="text-xs text-muted-foreground block">
													{textGet("form.calendar.start") || "Start"}
												</span>
												<Select
													value={startTime}
													onValueChange={(v) => handleTimeSelect("start", v)}
												>
													<SelectTrigger className="h-8 text-xs">
														<SelectValue placeholder="HH:mm" />
													</SelectTrigger>
													<SelectContent>
														{TIME_SLOTS.map((slot) => (
															<SelectItem
																key={`start-${slot}`}
																value={slot}
																className="text-xs"
															>
																{slot}
															</SelectItem>
														))}
													</SelectContent>
												</Select>
											</div>
											<div className="space-y-1">
												<span className="text-xs text-muted-foreground block">
													{textGet("form.calendar.end") || "End"}
												</span>
												<Select
													value={endTime}
													onValueChange={(v) => handleTimeSelect("end", v)}
												>
													<SelectTrigger className="h-8 text-xs">
														<SelectValue placeholder="HH:mm" />
													</SelectTrigger>
													<SelectContent>
														{TIME_SLOTS.map((slot) => (
															<SelectItem
																key={`end-${slot}`}
																value={slot}
																className="text-xs"
															>
																{slot}
															</SelectItem>
														))}
													</SelectContent>
												</Select>
											</div>
										</div>
									</div>
								)}
							</PopoverContent>
						</Popover>
						{description && <FieldDescription>{description}</FieldDescription>}
						{fieldState.invalid && <FieldError errors={[fieldState.error]} />}
					</Field>
				);
			}}
		/>
	);
}

export { FormCalendar, type FormCalendarProps };
