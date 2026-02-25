import { useMediaQuery } from "./user-media-query";

export function useResponsive() {
	const isDesktop = useMediaQuery("(min-width: 768px)");
	const isMobile = !isDesktop;
	return { isDesktop, isMobile };
}
