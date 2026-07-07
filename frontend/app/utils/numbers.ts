/**
 * Format a number for localized, consistent display (thousands separators,
 * decimals, etc.) using the Intl API.
 *
 * The i18n locale codes ("en", "nb") are valid BCP-47 tags, so they can be
 * passed straight through. Prefer the `useNumberFormat` composable inside
 * components so the current locale is applied automatically.
 */
export function formatNumber(
    value: number,
    locale = "en",
    options?: Intl.NumberFormatOptions,
): string {
    return new Intl.NumberFormat(locale, options).format(value);
}
