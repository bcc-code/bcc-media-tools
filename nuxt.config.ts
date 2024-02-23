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
        },
        public: {
            auth: {
                domain: process.env.AUTH_DOMAIN ?? "login.bcc.no",
                clientId:
                    process.env.AUTH_CLIENT_ID ??
                    "iaDsfutxWw4eoRHHVryW65JHd49kXaP0",
            },
        },
    },
});
