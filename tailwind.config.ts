import themes from "@bcc-code/design-library-vue";

export default {
    darkMode: "class",
    content: [
        "./components/**/*.{js,vue,ts}",
        "./layouts/**/*.vue",
        "./pages/**/*.vue",
        "./plugins/**/*.{js,ts}",
        "./app.vue",
        "./error.vue",
        "./nuxt.config.{js,ts}",
        "./node_modules/@bcc-code/design-library-vue/dist/design-library-vue.js",
    ],
    theme: {
        extend: {},
    },
    plugins: [themes.tailwindPlugin],
    presets: [themes.bccForbundetTheme],
};
