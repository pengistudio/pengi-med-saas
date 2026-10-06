/**
 * The short "ding" played when a WhatsApp message arrives, and the user's
 * preference to mute it (per browser, in localStorage).
 */

/** Two short tones, 8 kHz mono WAV (~2 KB), embedded so it needs no request. */
const DING_DATA_URI =
	"data:audio/wav;base64,UklGRvQHAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YdAHAACAgYOEgHl0dn6KkY+CcWdqe5GemoZsW193lammi2hRVHCXsrGSZ0lJaZe3t5hrSkhmkrO2mm9OSWONr7WddFJLYYmqs595V0xfhKayoH1bTl6BorCigF9RXX2eraKEY1Ncepqro4dnVVx3lqmjiWpYXHSTpqOMblpcco+jo45xXV1wjKGikHVgXW6JnqGSeGJebYaboJN6ZV9sg5mflH1oYGuBlp6Vf2pian+TnZWCbWNpfJGbloNvZGl7j5qWhXJmaXmMmJaHdGhpd4qXloh2aWl2iJWVinhranWGk5WLem1qdISRlIt8bmtzgpCUjH1wbHOBjpONf3Jscn+Nko2Ac21yfouRjYJ1bnJ9iZCOg3ZvcXyIj46EeHBxe4eOjoV5cXJ6hY2NhnpycnmEjI2GfHNyeIOLjYd9dHJ4goqMh351c3eBiYyIf3Zzd4CIi4iAeHR3f4eLiIF4dHd+hoqIgXl1d32FioiCenZ3fYSJiIN7dnd8g4iIg3x3d3yDiIiEfXh3e4KHiIR+eHd7gYaIhH55d3uAhoeFf3p4eoCFh4WAenh6f4SHhYB7eHp/hIaFgXx5en6DhoWBfHl6foOGhYF9enp9goWFgn16en2ChYWCfnp6fYGEhYJ+e3p9gYSFg397enyAhIWDf3x7fICDhIN/fHt8f4OEg4B8e3x/goSDgH17fH+ChIOAfXt8f4KDg4F+fHx+gYODgX58fH6Bg4OBfnx8foGDg4F/fXx+gIKDgX99fH6AgoOBf318fYCCg4J/fXx9gIKCgoB+fX1/gYKCgH59fX+BgoKAfn19f4GCgoB+fX1/gYKCgH59fX+AgoKAf319f4CCgoF/fn1+gIGCgX9+fX6AgYKBf35+foCBgYF/fn5+gIGBgYB+fn5/gYGBgH5+fn+BgYGAf35+f4CBgYB/fn5/gIGBgH9+fn+AgYGAf35+f4CBgYB/fn6AgYOAeXd/io2BcW19kpeEaWJ5maKIY1h0nqyOX05uoraUXERnpb+bWj1ho8KgXj5en8GjYz9bmr+mZ0FYlr2oa0NWkburcEVTjbmtdEdSibeueEpQhbSwfExPgbGxgE9Ofa6yhFJNeauyh1VNdqizi1hMcqWzjlxMb6KykV9NbJ+ylGJNapuxl2VOZ5ixmWlPZZWvm2xQY5GunW9SYY6tn3NTX4uroXZVXoiponlXXISno3xZW4GlpH9bWn+jpYJdWnyhpYVfWXmfpodhWXecpopkWXSapoxmWXKYpY5pWnCVpZBrWm6TpJJuW2yQpJRwXGqOo5VzXWmLopd1XmiJoZh4X2aGn5l6YGaEnpp8YmWCnJt+Y2SAm5uBZWR9mZyDZ2N7l5yFaGN5lpyGamN4lJyIbGN2kpyKbmN0kJyLcGRzjpuNcmRxjZuOc2Vwi5qPdWVviZmRd2Zuh5mReWdthZiSe2hsg5eTfGlsgpWUfmprgJSUgGtrfpOUgWxqfZKVg25qe5GVhG9qeo+VhnBqeY6Vh3JqeIyViHNqdouUiXVrdYqUinZrdIiUi3dsc4eTjHlsc4aSjXptcoSSjXtucYORjn1ucYKQjn5vcICPj39wcH+Oj4BxcH6Oj4JycH2Nj4NzcHyMj4R0cHuLj4V1cHqKj4Z2cHmJj4Z3cHiHj4d4cHeGj4h5cXeFjol6cXaEjol7cXaDjYp8cnWCjYp9cnWBjIt+c3SAjIt/dHSAi4uAdHR/iouBdXR+iouCdnR9iYuCd3R8iIuDd3R7h4uEeHR7h4uEeXR6houFenR6hYuGenR5hIuGe3V5g4qHfHV4g4qHfXV4goqHfXZ3gYmIfnZ3gImIf3d3gIiIgHd3f4iIgHh3foeIgXh3foeIgXl3fYaIgnl3fYaIg3p3fIWIg3p3fISIhHt3e4SIhHx3e4OIhHx3eoOIhX13eoKHhX14eoGHhX54eoGHhn54eYCGhn95eYCGhn95eX+GhoB5eX+FhoB6eX6FhoF6eX6FhoF7eX6EhoJ7eX2EhoJ8eX2DhoJ8eXyDhoN8eXyChoN9eXyChoN9eXyChYR+enuBhYR+enuBhYR/enuAhYR/enuAhYR/e3uAhISAe3t/hISAe3t/hISAe3t+g4SBfHp+g4SBfHt+g4SBfHt+goSCfXt9goSCfXt9goSCfXt9gYSCfnt9gYSCfnt8gYSDfnt8gYSDf3t8gISDf3x8gIODf3x8gIODgHx8f4ODgHx8f4ODgHx8f4KDgH18f4KDgX18foKDgX18foKDgX18foKDgX58foGDgX58foGDgn58fYGDgn58fYGDgn98fYCDgn98fYCDgn99fYCCgn99fYCCgn99fX+CgoB9fX+CgoB9fX+CgoB9fX+CgoB9fX+BgoB+fX+BgoF+fX6BgoF+fX6BgoF+fX6BgoF+fX6BgoF/fX6AgoF/fX6AgoF/fX6AgoF/fX6AgoF/fX6AgoF/fX2AgoKAfn1/gYKAfn1/gYKAfn1/gYKAfn1/gYKAfn1/gYKAfn1/gYKAfn1/gYKAfn1+gIGBf35+gIGBf35+gIGBf35+gIGBf35+gIGBf35+gIGBf35+gIGBf35+gIGBgH5+f4GBgH5+f4GBgH5+f4GBgH5+f4GBgH5+f4GBgH5+f4CBgH9+f4CBgH9+f4CBgH9+f4CBgH9+f4CBgH9+f4CBgH9+fw==";

const MUTED_KEY = "whatsapp.sound.muted";

/** The user muted the WhatsApp sound. Storage can be blocked: then it plays. */
export function isNotificationSoundMuted(): boolean {
	try {
		return localStorage.getItem(MUTED_KEY) === "1";
	} catch {
		return false;
	}
}

export function setNotificationSoundMuted(muted: boolean): void {
	try {
		if (muted) localStorage.setItem(MUTED_KEY, "1");
		else localStorage.removeItem(MUTED_KEY);
	} catch {
		// Blocked storage: the preference just doesn't persist.
	}
}

/**
 * Plays the ding unless muted or the tab is hidden. Browsers may block
 * autoplay before the user interacts with the page; that failure is ignored.
 */
export function playNotificationSound(): void {
	if (isNotificationSoundMuted()) return;
	if (typeof document !== "undefined" && document.hidden) return;
	try {
		const audio = new Audio(DING_DATA_URI);
		audio.volume = 0.6;
		audio.play()?.catch(() => {});
	} catch {
		// No audio support (tests, old browsers).
	}
}
