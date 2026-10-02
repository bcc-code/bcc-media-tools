/**
 * A single marker as edited in the UI. Start/End are kept as "HH:MM:SS"
 * strings so text editing is natural; they're parsed to milliseconds only at
 * save/preview time.
 */
export interface EditorialRow {
    id: string;
    name: string;
    contributors: string;
    bibleVerses: string;
    comment: string;
    type: string;
    start: string;
    end: string;
    publishBmm: boolean;
    publishBcc: boolean;
    source: string;
}

/** The marker fields the review view writes one at a time. */
export type EditorialField =
    "name" | "type" | "contributors" | "bibleVerses" | "comment";

export const EDITORIAL_TYPES = [
    "vitnesbyrd",
    "sang",
    "allsang",
    "tale",
    "bønn",
    "tydning",
    "programleder",
    "video",
    "intervju",
    "annet",
];

/**
 * Fill colour per marker type. Fixed pastels rather than theme tokens: the
 * timeline needs ~10 distinguishable hues, and these read the same against the
 * light and the dark surface (always paired with near-black text).
 */
const TYPE_COLORS: Record<string, string> = {
    vitnesbyrd: "#f0aec0",
    sang: "#8ed6c0",
    allsang: "#6fc8c4",
    tale: "#e7be8a",
    bønn: "#c0b4f0",
    tydning: "#9dc4f0",
    programleder: "#edd98c",
    video: "#a9d8e8",
    intervju: "#d9c3a0",
    annet: "#c7cdc9",
};

const TYPE_COLOR_FALLBACK = "#c7cdc9";

export function editorialTypeColor(type: string): string {
    return TYPE_COLORS[type] ?? TYPE_COLOR_FALLBACK;
}

export function formatMs(ms: number): string {
    const total = Math.max(0, Math.floor(ms / 1000));
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${pad(h)}:${pad(m)}:${pad(s)}`;
}

/** Parses "HH:MM:SS(.mmm)", "MM:SS" or "SS" into milliseconds; invalid → 0. */
export function parseTc(tc: string): number {
    const parts = tc
        .trim()
        .split(":")
        .map((p) => Number(p));
    if (parts.some((n) => Number.isNaN(n))) return 0;
    let seconds = 0;
    for (const p of parts) seconds = seconds * 60 + p;
    return Math.max(0, Math.round(seconds * 1000));
}

export function rowDurationMs(row: EditorialRow): number {
    return Math.max(0, parseTc(row.end) - parseTc(row.start));
}

/**
 * Bible verses are stored as one free-text field. Editors write them comma- or
 * semicolon-separated, so split on those to render one chip per reference.
 */
export function splitBibleVerses(value: string): string[] {
    return value
        .split(/[,;]/)
        .map((v) => v.trim())
        .filter(Boolean);
}
