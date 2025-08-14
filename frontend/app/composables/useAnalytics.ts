export function useAnalytics() {
	const { $analytics } = useNuxtApp()
	return $analytics
}