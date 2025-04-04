// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
    modules: ["@nuxt/icon", "@nuxtjs/i18n", "vue-sonner/nuxt", "motion-v/nuxt"],
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
            rudderstack: {
                writeKey: "",
                dataPlaneUrl: "",
            }
        },
    },
    devServer: {
        port: 8001,
        host: "localhost",
    },
    compatibilityDate: "2024-10-16",
});
