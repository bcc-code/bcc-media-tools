// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
    modules: ["nuxt-icon"],
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
            {
                from: "@bcc-code/design-library-vue",
                imports: ["BccAlert", "BccInput"],
            },
        ],
    },
    nitro: {
        preset: "firebase",
        firebase: {
            gen: 2,
            httpsOptions: {
                region: "europe-west4",
                maxInstances: 10,
            },
            nodeVersion: "20",
        },
    },
    runtimeConfig: {
        api: {
            auth: {
                cert: process.env.AUTH_CERTIFICATE,
            },
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
