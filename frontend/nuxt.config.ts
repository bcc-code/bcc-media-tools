// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
    modules: ["nuxt-icon", "@nuxtjs/i18n"],
    typescript: {
        shim: false,
    },
    devtools: { enabled: false },
    ssr: false,
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
    imports: {
        presets: [
            {
                from: "@auth0/auth0-vue",
                imports: ["useAuth0"],
            },
        ],
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
            tempDrivePath: "",
            auth0: {
                domain: "login.bcc.no",
                clientId: "",
                clientSecret: "",
            },
            bmm: {
                audience: "",
            },
            configDir: "./config",
            temporalTriggerUrl: "https://temporal-trigger.lan.bcc.media",
        },
        public: {
            auth: {
                domain: "login.bcc.no",
                clientId: "iaDsfutxWw4eoRHHVryW65JHd49kXaP0",
            },
        },
    },
    devServer: {
        port: 80,
        host: "0.0.0.0",
    },
});
