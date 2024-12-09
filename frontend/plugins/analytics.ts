import { load, ready } from "rudder-sdk-js";

export default defineNuxtPlugin(() => {
	const config = useRuntimeConfig()
	const analytics = new Analytics()

	load(
		config.public.rudderstack.writeKey,
		config.public.rudderstack.dataPlaneUrl,
	);

	ready(() => {
		analytics.initialize()
	})

	return {
		provide: {
			analytics
		},
	};
});