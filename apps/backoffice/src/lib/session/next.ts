/**
 * Where to go after login. Only same-app paths are accepted, so a crafted
 * ?next= can't send the user to another site.
 */
export function safeNext(next: string | null): string {
	const inApp =
		next?.startsWith("/") && !next.startsWith("//") && !next.startsWith("/\\");
	return inApp && next ? next : "/";
}
