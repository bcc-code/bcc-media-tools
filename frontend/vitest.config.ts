import { defineVitestConfig } from "@nuxt/test-utils/config";

export default defineVitestConfig({
    test: {
        // The Nuxt environment gives tests the same auto-imports the app has,
        // so nothing needs importing purely to be testable.
        environment: "nuxt",
        include: ["test/**/*.spec.ts"],
    },
});
