// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
    modules: ["@nuxt/icon", "@nuxtjs/i18n"],
    typescript: {
        shim: false,
    },
    devtools: { enabled: false },
    ssr: false,
    css: ["~/assets/css/main.css"],
    app: {
        head: {
            bodyAttrs: {
                class: "bg-neutral-100",
            },
            titleTemplate: "%s - BCC Media Tools"
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
        }
    },
    devServer: {
        port: 8001,
        host: "localhost",
    },
    compatibilityDate: "2024-10-16",
});