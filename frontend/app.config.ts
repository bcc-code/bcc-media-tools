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
		}
	}
})