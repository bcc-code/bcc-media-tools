export default defineI18nConfig(async () => {
    const locales = ["en", "no"];

    const messages: any = {};

    for (const l of locales) {
        messages[l] = await import(`~/locales/${l}.json`);
    }

    return {
        availableLocales: locales,
        fallbackLocale: "en",
        messages,
        missingWarn: false,
    };
});
