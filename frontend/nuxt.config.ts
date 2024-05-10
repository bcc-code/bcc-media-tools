// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
    modules: ["nuxt-icon", "@nuxtjs/i18n"],
    typescript: {
        shim: false,
    },
    devtools: { enabled: false },
    ssr: true,
    css: ["~/assets/css/main.css"],
    app: {
        head: {
            bodyAttrs: {
                class: "bg-slate-900 md:bg-slate-800 text-white",
            },
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
        api: {
            cantemo: {
                authToken: "",
                baseUrl: "",
            },
        },
    },
    devServer: {
        port: 80,
        host: "0.0.0.0",
    },
});
