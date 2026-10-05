import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { createI18n } from "vue-i18n";

/*
 * Read from disk rather than importing the JSON: the i18n module pre-compiles
 * messages at import time, so the imported object holds compiled AST nodes
 * instead of the strings this file is here to check.
 */
function locale(name: string): Record<string, unknown> {
    // `import.meta.url` is stubbed in the Nuxt test environment, so resolve
    // against the vitest root (the frontend directory) instead.
    return JSON.parse(
        readFileSync(
            resolve(process.cwd(), "locales", `${name}.json`),
            "utf-8",
        ),
    );
}

const en = locale("en");
const nb = locale("nb");

/*
 * The app augments vue-i18n's types to its own runtime locale names, which the
 * raw files loaded here are not, so this declares the narrow slice of the API
 * the tests use rather than trying to satisfy the app's augmentation.
 */
type TestI18n = {
    global: {
        t: (
            key: string,
            named: Record<string, unknown>,
            options?: { locale?: string },
        ) => string;
    };
};

const i18n = (
    createI18n as unknown as (options: Record<string, unknown>) => TestI18n
)({
    legacy: false,
    locale: "en",
    fallbackLocale: "en",
    messages: { en, nb },
});

const t = i18n.global.t;

function keyPaths(value: unknown, prefix = ""): string[] {
    if (typeof value !== "object" || value === null) return [prefix];
    return Object.entries(value).flatMap(([key, child]) =>
        keyPaths(child, prefix ? `${prefix}.${key}` : key),
    );
}

function messageAt(locale: Record<string, unknown>, path: string): string {
    let value: unknown = locale;
    for (const key of path.split(".")) {
        value = (value as Record<string, unknown> | undefined)?.[key];
    }
    if (typeof value !== "string") {
        throw new Error(`${path} is not a string message`);
    }
    return value;
}

describe("locales", () => {
    it("has the same keys in both languages", () => {
        const enKeys = keyPaths(en).sort();
        const nbKeys = keyPaths(nb).sort();

        expect(nbKeys.filter((k) => !enKeys.includes(k))).toEqual([]);
        expect(enKeys.filter((k) => !nbKeys.includes(k))).toEqual([]);
    });

    it("gives pluralised messages the same number of forms in both languages", () => {
        const mismatched = keyPaths(en)
            .map((path) => ({
                path,
                en: messageAt(en, path).split("|").length,
                nb: messageAt(nb, path).split("|").length,
            }))
            .filter((m) => m.en !== m.nb);

        expect(mismatched).toEqual([]);
    });
});

describe("shorts submit wording", () => {
    const key = "shorts.generation.submitConfirmationMessage";

    it("reads as one short for a single clip", () => {
        expect(t(key, { count: 1 })).toContain("1 short ");
        expect(t(key, { count: 1 })).toContain("send it");
    });

    it("reads as several shorts for a batch", () => {
        expect(t(key, { count: 3 })).toContain("3 shorts");
        expect(t(key, { count: 3 })).toContain("send them");
    });

    it("counts clips the same way in Norwegian", () => {
        expect(t(key, { count: 1 }, { locale: "nb" })).toContain("1 short");
        expect(t(key, { count: 3 }, { locale: "nb" })).toContain("3 shorts");
    });

    it("picks the right form for the clip count label", () => {
        const count = "shorts.generation.clipCount";

        expect(t(count, { count: 0 })).toBe("No clips");
        expect(t(count, { count: 1 })).toBe("1 clip");
        expect(t(count, { count: 4 })).toBe("4 clips");
    });
});
