export default defineNuxtConfig({
    modules: [
        "@nuxt/icon",
        "@nuxtjs/i18n",
        "motion-v/nuxt",
        "@vueuse/nuxt",
        "@nuxt/ui",
    ],
    typescript: {
        shim: false,
    },
    ssr: false,
    css: ["~/assets/css/main.css"],
    app: {
        head: {
            bodyAttrs: {
                class: "bg-neutral-100 dark:bg-neutral-950",
            },
            titleTemplate: "%s - BCC Media Tools",
            link: [
                {
                    rel: "icon",
                    type: "image/x-icon",
                    href: "/images/logo.png",
                },
            ],
        },
    },
    experimental: {
        typedPages: true,
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
        restructureDir: './',
        strategy: 'no_prefix',
    },
    devServer: {
        port: 8001,
        host: "localhost",
    },
    compatibilityDate: "2025-07-12",
});
