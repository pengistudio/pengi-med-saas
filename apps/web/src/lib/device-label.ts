/**
 * Short label of this browser and platform ("Chrome on macOS"), the default
 * name of a new passkey. Browser and OS names are proper nouns, not copy.
 */
export function deviceLabel(userAgent = navigator.userAgent): string | null {
	const browser = /Edg\//.test(userAgent)
		? "Edge"
		: /OPR\/|Opera/.test(userAgent)
			? "Opera"
			: /Firefox\//.test(userAgent)
				? "Firefox"
				: /Chrome\/|CriOS\//.test(userAgent)
					? "Chrome"
					: /Safari\//.test(userAgent)
						? "Safari"
						: null;
	const platform = /Android/.test(userAgent)
		? "Android"
		: /iPhone|iPad|iPod/.test(userAgent)
			? "iOS"
			: /Windows/.test(userAgent)
				? "Windows"
				: /Mac OS X|Macintosh/.test(userAgent)
					? "macOS"
					: /CrOS/.test(userAgent)
						? "ChromeOS"
						: /Linux/.test(userAgent)
							? "Linux"
							: null;
	if (browser && platform) return `${browser} / ${platform}`;
	return browser ?? platform;
}
