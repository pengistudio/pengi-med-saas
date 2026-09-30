/**
 * Whole years between `birthDate` and `now`, counted on the calendar (a year
 * is added on the birthday itself). Null when there is no birth date: the API
 * sends the zero time ("0001-01-01") for patients without one.
 */
export function ageInYears(
	birthDate: string | Date | undefined | null,
	now: Date = new Date(),
): number | null {
	if (!birthDate) return null;
	const birth = new Date(birthDate);
	if (Number.isNaN(birth.getTime()) || birth.getFullYear() <= 1) return null;
	let age = now.getFullYear() - birth.getFullYear();
	const beforeBirthday =
		now.getMonth() < birth.getMonth() ||
		(now.getMonth() === birth.getMonth() && now.getDate() < birth.getDate());
	if (beforeBirthday) age--;
	return age >= 0 ? age : null;
}

/**
 * The birth date to store for a patient who only reported an age. Someone who
 * is `age` today was born in the year before `now` minus `age` years; the
 * middle of that year keeps the error within six months, and the age is still
 * `age` today.
 */
export function birthDateFromAge(age: number, now: Date = new Date()): Date {
	return new Date(now.getFullYear() - age, now.getMonth() - 6, now.getDate());
}
