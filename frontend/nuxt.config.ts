// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
    modules: [
        "@nuxt/icon",
        "@nuxtjs/i18n",
        "vue-sonner/nuxt",
        "motion-v/nuxt",
        "@vueuse/nuxt",
    ],
    typescript: {
        shim: false,
    },
    ssr: false,
    css: ["~/assets/css/main.css"],
    app: {
        head: {
            bodyAttrs: {
                class: "bg-neutral-100",
            },
            titleTemplate: "%s - BCC Media Tools",
        },
        pageTransition: { name: "page", mode: "out-in" },
    },
    experimental: {
        typedPages: true,
    },
    postcss: {
        plugins: {
            tailwindcss: {},
            autoprefixer: {},
        },
    },
    nitro: {
        preset: "node-server",
    },
    runtimeConfig: {
        public: {
            grpcUrl: "http://localhost:8080",
            rudderstack: {
                writeKey: "",
                dataPlaneUrl: "",
            },
        },
    },
    i18n: {
        defaultLocale: "en",
        langDir: "locales",
        locales: [
            { code: "en", name: "English", file: "en.json" },
            { code: "nb", name: "Norsk", file: "nb.json" },
        ],
        detectBrowserLanguage: false,
        restructureDir: './',
    },
    devServer: {
        port: 8001,
        host: "localhost",
    },
    compatibilityDate: "2024-10-16",
});
