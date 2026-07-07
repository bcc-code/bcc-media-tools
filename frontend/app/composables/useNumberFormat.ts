/**
 * Returns a `formatNumber` bound to the current i18n locale, so numbers are
 * displayed consistently across the app.
 */
export function useNumberFormat() {
    const { locale } = useI18n();

    return {
        formatNumber: (value: number, options?: Intl.NumberFormatOptions) =>
            formatNumber(value, locale.value, options),
    };
}
