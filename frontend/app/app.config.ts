export default defineAppConfig({
	ui: {
		colors: {
			primary: 'indigo',
			neutral: 'neutral'
		},

		// Custom default classes for components
		select: {
			slots: {
				content: 'min-w-fit'
			}
		},
		modal: {
			slots: {
				overlay: 'bg-black/50 dark:bg-black/80',
			}
		}
	}
})