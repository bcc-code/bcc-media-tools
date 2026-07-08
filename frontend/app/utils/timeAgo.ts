import type { UseTimeAgoMessages } from "@vueuse/core";

// Locale message tables for VueUse's `formatTimeAgo` / `useTimeAgo`. VueUse only
// ships English, so we provide the strings ourselves. The `past`/`future`
// wrappers only add "ago"/"in" when the value contains a digit, leaving
// worded units like "last month" untouched (mirrors VueUse's default).

const en: UseTimeAgoMessages = {
    justNow: "just now",
    past: (n) => (/\d/.test(n) ? `${n} ago` : n),
    future: (n) => (/\d/.test(n) ? `in ${n}` : n),
    second: (n) => `${n} second${n === 1 ? "" : "s"}`,
    minute: (n) => `${n} minute${n === 1 ? "" : "s"}`,
    hour: (n) => `${n} hour${n === 1 ? "" : "s"}`,
    day: (n) => `${n} day${n === 1 ? "" : "s"}`,
    week: (n) => `${n} week${n === 1 ? "" : "s"}`,
    month: (n, past) =>
        n === 1 ? (past ? "last month" : "next month") : `${n} months`,
    year: (n, past) =>
        n === 1 ? (past ? "last year" : "next year") : `${n} years`,
    invalid: "",
};

const nb: UseTimeAgoMessages = {
    justNow: "akkurat nå",
    past: (n) => (/\d/.test(n) ? `for ${n} siden` : n),
    future: (n) => (/\d/.test(n) ? `om ${n}` : n),
    second: (n) => `${n} sekund${n === 1 ? "" : "er"}`,
    minute: (n) => `${n} minutt${n === 1 ? "" : "er"}`,
    hour: (n) => `${n} time${n === 1 ? "" : "r"}`,
    day: (n) => `${n} dag${n === 1 ? "" : "er"}`,
    week: (n) => `${n} uke${n === 1 ? "" : "r"}`,
    month: (n, past) =>
        n === 1 ? (past ? "forrige måned" : "neste måned") : `${n} måneder`,
    year: (n, past) => (n === 1 ? (past ? "i fjor" : "neste år") : `${n} år`),
    invalid: "",
};

const MESSAGES: Record<string, UseTimeAgoMessages> = { en, nb };

// Returns the time-ago message table for a locale, falling back to English.
export function timeAgoMessages(locale: string): UseTimeAgoMessages {
    return MESSAGES[locale] ?? en;
}
