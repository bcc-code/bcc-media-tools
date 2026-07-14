import { RudderAnalytics } from "@rudderstack/analytics-js";

export default defineNuxtPlugin(() => {
    const config = useRuntimeConfig();

    const rudder = new RudderAnalytics();
    rudder.load(
        config.public.rudderstack.writeKey,
        config.public.rudderstack.dataPlaneUrl,
    );

    const analytics = new Analytics(rudder);

    rudder.ready(() => {
        analytics.initialize();
    });

    return {
        provide: {
            analytics,
        },
    };
});
