import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import { SegmentsSchema, WordsSchema } from "~~/src/gen/api/v1/api_pb";
import type { Segments } from "~~/src/gen/api/v1/api_pb";
import {
    PRE_ROLL,
    TAIL,
    clipHighlightRange,
    flattenSegments,
    matchingSegments,
    rangeFromSelection,
    overlapsRange,
    wordIndex,
} from "~/utils/shortsTranscript";

function segment(
    text: string,
    start: number,
    end: number,
    words: [string, number, number][] = [],
): Segments {
    return create(SegmentsSchema, {
        text,
        start,
        end,
        words: words.map(([t, s, e]) =>
            create(WordsSchema, { text: t, start: s, end: e, confidence: 1 }),
        ),
    });
}

describe("flattenSegments", () => {
    it("numbers words continuously across segments", () => {
        const flat = flattenSegments([
            segment("one two", 0, 2, [
                ["one", 0, 1],
                ["two", 1, 2],
            ]),
            segment("three", 2, 3, [["three", 2, 3]]),
        ]);

        expect(flat.map((s) => s.words.map((w) => w.index))).toEqual([
            [0, 1],
            [2],
        ]);
    });

    it("covers a segment with no word timings as a single word", () => {
        const flat = flattenSegments([segment("no word timings", 4, 9)]);

        expect(flat[0]?.words).toEqual([
            { index: 0, text: "no word timings", start: 4, end: 9 },
        ]);
    });

    it("drops whitespace-only words", () => {
        const flat = flattenSegments([
            segment("a b", 0, 3, [
                ["a", 0, 1],
                ["   ", 1, 2],
                ["b", 2, 3],
            ]),
        ]);

        expect(flat[0]?.words.map((w) => w.text)).toEqual(["a", "b"]);
    });

    it("trims the surrounding whitespace whisper puts on words", () => {
        const flat = flattenSegments([
            segment(" hei ", 0, 1, [[" hei", 0, 1]]),
        ]);

        expect(flat[0]?.words[0]?.text).toBe("hei");
        expect(flat[0]?.text).toBe("hei");
    });

    it("returns nothing for no segments", () => {
        expect(flattenSegments([])).toEqual([]);
    });

    it("drops a segment whose words are all whitespace", () => {
        const flat = flattenSegments([
            segment("", 0, 2, [
                ["  ", 0, 1],
                ["\t", 1, 2],
            ]),
            segment("real", 2, 3, [["real", 2, 3]]),
        ]);

        expect(flat.map((s) => s.text)).toEqual(["real"]);
    });

    it("drops a segment with neither words nor text", () => {
        expect(flattenSegments([segment("   ", 0, 2)])).toEqual([]);
    });

    it("flattens a transcript of nothing but empty segments to nothing", () => {
        const flat = flattenSegments([segment("", 0, 1), segment("  ", 1, 2)]);

        expect(flat).toEqual([]);
    });

    it("keeps the surviving segments in order", () => {
        const flat = flattenSegments([
            segment("first", 0, 1, [["first", 0, 1]]),
            segment("", 1, 2),
            segment("third", 2, 3, [["third", 2, 3]]),
        ]);

        expect(flat.map((s) => s.text)).toEqual(["first", "third"]);
        expect(flat.map((s) => s.index)).toEqual([0, 2]);
    });
});

describe("rangeFromSelection", () => {
    const flat = flattenSegments([
        segment("one two", 10, 12, [
            ["one", 10, 11],
            ["two", 11, 12],
        ]),
        segment("three", 20, 21, [["three", 20, 21]]),
    ]);
    const words = wordIndex(flat);

    it("pads the selection on both sides", () => {
        expect(rangeFromSelection(words, 0, 1, 60)).toEqual({
            start: 10 - PRE_ROLL,
            end: 12 + TAIL,
        });
    });

    it("spans segments", () => {
        expect(rangeFromSelection(words, 0, 2, 60)).toEqual({
            start: 10 - PRE_ROLL,
            end: 21 + TAIL,
        });
    });

    it("reads a backwards selection the same as a forwards one", () => {
        expect(rangeFromSelection(words, 2, 0, 60)).toEqual(
            rangeFromSelection(words, 0, 2, 60),
        );
    });

    it("does not let the pre-roll run past the start of the media", () => {
        const atZero = wordIndex(
            flattenSegments([segment("hei", 0, 1, [["hei", 0, 1]])]),
        );

        expect(rangeFromSelection(atZero, 0, 0, 60)?.start).toBe(0);
    });

    it("does not let the tail run past the end of the media", () => {
        expect(rangeFromSelection(words, 2, 2, 21)?.end).toBe(21);
    });

    it("keeps end at or after start when the media is shorter than the word", () => {
        const range = rangeFromSelection(words, 2, 2, 5);

        expect(range).toBeDefined();
        expect(range!.end).toBeGreaterThanOrEqual(range!.start);
    });

    it("returns undefined for an index that is not a word", () => {
        expect(rangeFromSelection(words, 0, 99, 60)).toBeUndefined();
    });
});

describe("matchingSegments", () => {
    const flat = flattenSegments([
        segment("Velkommen til Explorers", 0, 2),
        segment("finn ut hvem du skal kontakte", 2, 4),
        segment("EXPLORERS igjen", 4, 6),
    ]);

    it("matches case-insensitively", () => {
        expect(matchingSegments(flat, "explorers")).toEqual([0, 2]);
    });

    it("matches across words within a segment", () => {
        expect(matchingSegments(flat, "hvem du skal")).toEqual([1]);
    });

    it("treats a blank query as no search at all", () => {
        expect(matchingSegments(flat, "   ")).toEqual([]);
    });

    it("returns nothing when there is no match", () => {
        expect(matchingSegments(flat, "zzz")).toEqual([]);
    });
});

describe("clipHighlightRange", () => {
    it("is the range itself for a normal clip", () => {
        expect(clipHighlightRange(10, 40, 600)).toEqual({ start: 10, end: 40 });
    });

    it("has nothing to show for the initial full-asset range", () => {
        expect(clipHighlightRange(0, 600, 600)).toBeUndefined();
    });

    it("still counts as full coverage just inside the bounds", () => {
        expect(clipHighlightRange(0.005, 599.995, 600)).toBeUndefined();
    });

    it("highlights a clip that reaches one end but not the other", () => {
        expect(clipHighlightRange(0, 30, 600)).toEqual({ start: 0, end: 30 });
        expect(clipHighlightRange(570, 600, 600)).toEqual({
            start: 570,
            end: 600,
        });
    });

    it("has nothing to show for an empty or inverted range", () => {
        expect(clipHighlightRange(30, 30, 600)).toBeUndefined();
        expect(clipHighlightRange(40, 30, 600)).toBeUndefined();
    });
});

describe("overlapsRange", () => {
    const word = { index: 0, text: "hei", start: 10, end: 11 };

    it("includes a word inside the range", () => {
        expect(overlapsRange(word, { start: 5, end: 15 })).toBe(true);
    });

    it("includes a word the range only partly covers", () => {
        expect(overlapsRange(word, { start: 10.5, end: 15 })).toBe(true);
        expect(overlapsRange(word, { start: 5, end: 10.5 })).toBe(true);
    });

    it("excludes a word that only touches the boundary", () => {
        expect(overlapsRange(word, { start: 11, end: 20 })).toBe(false);
        expect(overlapsRange(word, { start: 0, end: 10 })).toBe(false);
    });

    it("excludes a word outside the range", () => {
        expect(overlapsRange(word, { start: 20, end: 30 })).toBe(false);
    });

    it("applies the same rule to a whole segment", () => {
        const seg = { start: 10, end: 20 };

        expect(overlapsRange(seg, { start: 19, end: 40 })).toBe(true);
        expect(overlapsRange(seg, { start: 20, end: 40 })).toBe(false);
    });
});
