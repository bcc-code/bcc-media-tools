import type { Segments } from "~~/src/gen/api/v1/api_pb";

/*
 * Whisper puts the start of a word on the first sample of the consonant and the
 * end on the last, so a range cut exactly on those boundaries clips both edges
 * audibly. A fifth of a second in front and a sixth behind lets the word breathe
 * without pulling in the neighbouring one.
 */
export const PRE_ROLL = 0.2;
export const TAIL = 0.15;

/*
 * The panel addresses words by a single running index across the whole
 * transcript, so a selection that spans segments is just a pair of numbers.
 */
export type FlatWord = {
    index: number;
    text: string;
    start: number;
    end: number;
};

export type FlatSegment = {
    index: number;
    start: number;
    end: number;
    text: string;
    words: FlatWord[];
};

/**
 * Flattens API segments into rows of individually addressable words.
 *
 * Segments the ASR did not time word by word still get a single entry covering
 * the whole segment, so every segment can be selected whatever its shape.
 * Whitespace-only words carry no timing worth selecting and are dropped, which
 * leaves gaps in the numbering; nothing depends on it being contiguous, only on
 * it being ordered and unique.
 *
 * A segment left with no words at all is dropped too: it would render as a bare
 * timestamp with nothing to read or select. A transcript made entirely of those
 * flattens to nothing, which is how callers can tell there is nothing to show.
 */
export function flattenSegments(segments: Segments[]): FlatSegment[] {
    let index = 0;
    return segments
        .map((segment, segmentIndex) => {
            const source =
                segment.words.length > 0
                    ? segment.words
                    : [
                          {
                              text: segment.text,
                              start: segment.start,
                              end: segment.end,
                          },
                      ];

            const words = source
                .map((w) => ({
                    index: index++,
                    text: w.text.trim(),
                    start: w.start,
                    end: w.end,
                }))
                .filter((w) => w.text !== "");

            return {
                index: segmentIndex,
                start: segment.start,
                end: segment.end,
                text: segment.text.trim(),
                words,
            };
        })
        .filter((segment) => segment.words.length > 0);
}

export function wordIndex(segments: FlatSegment[]): Map<number, FlatWord> {
    const out = new Map<number, FlatWord>();
    for (const segment of segments) {
        for (const word of segment.words) out.set(word.index, word);
    }
    return out;
}

/**
 * The clip range a word selection stands for, padded and held inside the media.
 * Returns undefined when either end of the selection is not a known word.
 */
export function rangeFromSelection(
    words: Map<number, FlatWord>,
    from: number,
    to: number,
    duration: number,
): { start: number; end: number } | undefined {
    const first = words.get(Math.min(from, to));
    const last = words.get(Math.max(from, to));
    if (!first || !last) return undefined;

    const start = Math.max(0, first.start - PRE_ROLL);
    const end = Math.min(duration, last.end + TAIL);
    return { start, end: Math.max(start, end) };
}

/** Indexes of the segments whose text contains `query`, case-insensitively. */
export function matchingSegments(
    segments: FlatSegment[],
    query: string,
): number[] {
    const q = query.trim().toLowerCase();
    if (!q) return [];
    return segments
        .filter((s) => s.text.toLowerCase().includes(q))
        .map((s) => s.index);
}

export type ClipRange = { start: number; end: number };

/*
 * How close to the asset's own bounds a range has to be before it counts as
 * covering the whole thing. Durations and in/out points are floats that have
 * been through a video element, so they rarely land on exact equality.
 */
const FULL_COVERAGE_EPSILON = 0.01;

/**
 * The range the transcript should highlight, or undefined for no highlight.
 *
 * An empty range has nothing to show, and so does the initial one that covers
 * the whole asset: lighting up every word says nothing about where the clip is.
 */
export function clipHighlightRange(
    start: number,
    end: number,
    duration: number,
): ClipRange | undefined {
    if (end <= start) return undefined;

    const coversEverything =
        start <= FULL_COVERAGE_EPSILON &&
        end >= duration - FULL_COVERAGE_EPSILON;
    if (coversEverything) return undefined;

    return { start, end };
}

/**
 * Whether a word or a segment overlaps the clip at all, rather than sitting
 * wholly inside it. A word half-covered by the range is part of the clip.
 */
export function overlapsRange(
    span: { start: number; end: number },
    range: ClipRange,
): boolean {
    return span.end > range.start && span.start < range.end;
}
