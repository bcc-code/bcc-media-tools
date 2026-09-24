import { describe, expect, it } from "vitest";
import type { Segment, Word } from "~/utils/transcription";
import {
    insertSegmentAt,
    realignWords,
    segmentText,
    setSegmentText,
    toTranscription,
    toggleSegmentDeleted,
    tokenizeWords,
    updateSegment,
    withUids,
    wordAtOffset,
} from "~/utils/transcription";

function word(text: string, start: number, end: number): Word {
    return { text, start, end, confidence: 1 };
}

function segment(uid: string, texts: string[], start = 0, end = 1): Segment {
    const step = (end - start) / texts.length;
    return {
        uid,
        id: Number(uid.replace(/\D/g, "")) || 0,
        seek: 0,
        start,
        end,
        text: texts.join(" "),
        tokens: [1, 2, 3],
        temperature: 0,
        avg_logprob: 0,
        compression_ration: 0,
        no_speech_prob: 0,
        confidence: 1,
        words: texts.map((t, i) =>
            word(t, start + i * step, start + (i + 1) * step),
        ),
    };
}

const doc = () => [
    segment("a", ["Hei", "alle"], 0, 2),
    segment("b", ["sammen", "her"], 2, 4),
    segment("c", ["i", "dag"], 10, 12),
];

/** The common correction: fix one word, leave the rest of the line alone. */
const fixFirstWord = (s: Segment, text: string) =>
    setSegmentText(s, [text, ...s.words.slice(1).map((w) => w.text)].join(" "));

const times = (words: Word[]) => words.map((w) => [w.start, w.end]);

describe("withUids", () => {
    it("gives every segment an id and keeps the ones already set", () => {
        const [first, second] = withUids([
            { ...segment("x", ["a"]), uid: undefined } as never,
            segment("keep", ["b"]),
        ]);

        expect(first!.uid).toBeTruthy();
        expect(second!.uid).toBe("keep");
    });

    it("does not reuse an id across segments", () => {
        const uids = withUids(
            Array.from({ length: 50 }, () => ({
                ...segment("n", ["x"]),
                uid: undefined,
            })) as never,
        ).map((s) => s.uid);

        expect(new Set(uids).size).toBe(50);
    });
});

describe("segmentText", () => {
    it("joins words and ignores empty ones", () => {
        expect(
            segmentText([word("Hei", 0, 1), word("", 1, 2), word("du", 2, 3)]),
        ).toBe("Hei du");
    });

    it("trims each word so an edit cannot introduce double spaces", () => {
        expect(segmentText([word(" Hei ", 0, 1), word("du ", 1, 2)])).toBe(
            "Hei du",
        );
    });
});

describe("tokenizeWords", () => {
    it("splits on any run of whitespace", () => {
        expect(tokenizeWords("  Hei   du \n der ")).toEqual([
            "Hei",
            "du",
            "der",
        ]);
    });

    it("returns nothing for blank text", () => {
        expect(tokenizeWords("   ")).toEqual([]);
    });
});

describe("realignWords", () => {
    const bounds = { start: 0, end: 4 };
    const previous = [word("Kåre", 0, 1), word("sa", 2, 3), word("det", 3, 4)];

    it("keeps the timings of words that did not change", () => {
        const result = realignWords(previous, "Kåre sa det", bounds);

        expect(times(result)).toEqual([
            [0, 1],
            [2, 3],
            [3, 4],
        ]);
    });

    it("gives a corrected word the span of the one it replaced", () => {
        const result = realignWords(previous, "Kaare sa det", bounds);

        expect(result[0]!.text).toBe("Kaare");
        expect(times(result)).toEqual([
            [0, 1],
            [2, 3],
            [3, 4],
        ]);
    });

    it("does not let a corrected word swallow the silence before it", () => {
        const late = [word("Kåre", 0.5, 1), word("sa", 2, 3)];
        const result = realignWords(late, "Kaare sa", { start: 0, end: 4 });

        expect(times(result)[0]).toEqual([0.5, 1]);
    });

    it("keeps a word's timing when punctuation changes next to an insert", () => {
        // Without normalised matching "Hei," reads as a new word, and it and
        // the inserted one would split the first word's span between them.
        const result = realignWords(
            [word("Hei", 0, 1), word("du", 2, 3)],
            "Hei, kjære du",
            { start: 0, end: 4 },
        );

        expect(times(result)).toEqual([
            [0, 1],
            [1, 2],
            [2, 3],
        ]);
    });

    it("keeps timings when only case or punctuation changed", () => {
        const result = realignWords(previous, "kåre, sa det.", bounds);

        expect(result.map((w) => w.text)).toEqual(["kåre,", "sa", "det."]);
        expect(times(result)).toEqual([
            [0, 1],
            [2, 3],
            [3, 4],
        ]);
    });

    it("spreads an inserted word across the gap between its neighbours", () => {
        const result = realignWords(previous, "Kåre virkelig sa det", bounds);

        expect(result.map((w) => w.text)).toEqual([
            "Kåre",
            "virkelig",
            "sa",
            "det",
        ]);
        expect(times(result)[1]).toEqual([1, 2]);
    });

    it("shares a gap between several inserted words", () => {
        const result = realignWords(previous, "Kåre ja da sa det", bounds);

        expect(times(result)[1]).toEqual([1, 1.5]);
        expect(times(result)[2]).toEqual([1.5, 2]);
    });

    it("uses the segment bounds for words inserted at the edges", () => {
        const result = realignWords([word("sa", 2, 3)], "og sa det", {
            start: 0,
            end: 4,
        });

        expect(times(result)).toEqual([
            [0, 2],
            [2, 3],
            [3, 4],
        ]);
    });

    it("drops a deleted word and leaves the others untouched", () => {
        const result = realignWords(previous, "Kåre det", bounds);

        expect(result.map((w) => w.text)).toEqual(["Kåre", "det"]);
        expect(times(result)).toEqual([
            [0, 1],
            [3, 4],
        ]);
    });

    it("returns nothing when all the text is removed", () => {
        expect(realignWords(previous, "   ", bounds)).toEqual([]);
    });

    it("spreads evenly when there were no words to align against", () => {
        const result = realignWords([], "en to", { start: 0, end: 4 });

        expect(times(result)).toEqual([
            [0, 2],
            [2, 4],
        ]);
    });

    it("never invents a timing outside the segment", () => {
        const result = realignWords(previous, "helt nye ord her", bounds);

        expect(result[0]!.start).toBeGreaterThanOrEqual(bounds.start);
        expect(result.at(-1)!.end).toBeLessThanOrEqual(bounds.end);
    });
});

describe("setSegmentText", () => {
    it("re-derives the segment text from the edited words", () => {
        const result = setSegmentText(
            segment("a", ["Hei", "alle"]),
            "Hei dere",
        );

        expect(result.text).toBe("Hei dere");
        expect(result.words.map((w) => w.text)).toEqual(["Hei", "dere"]);
    });

    it("normalises whitespace", () => {
        expect(setSegmentText(segment("a", ["Hei"]), "  Hei   du ").text).toBe(
            "Hei du",
        );
    });

    it("does not mutate the input", () => {
        const before = segment("a", ["Hei"]);
        setSegmentText(before, "Hallo");

        expect(before.text).toBe("Hei");
    });
});

describe("wordAtOffset", () => {
    const words = [word("Hei", 0, 1), word("du", 1, 2), word("der", 2, 3)];

    it.each([
        [0, "Hei"],
        [3, "Hei"],
        [4, "du"],
        [6, "du"],
        [7, "der"],
        [10, "der"],
    ])("offset %i is in %s", (offset, expected) => {
        expect(wordAtOffset(words, offset)?.text).toBe(expected);
    });

    it("returns nothing for a segment with no words", () => {
        expect(wordAtOffset([], 0)).toBeUndefined();
    });
});

describe("updateSegment", () => {
    it("updates the row with the matching uid and leaves the rest alone", () => {
        const result = updateSegment(doc(), "b", (s) => ({
            ...s,
            text: "endret",
        }));

        expect(result.map((s) => s.text)).toEqual([
            "Hei alle",
            "endret",
            "i dag",
        ]);
    });

    it("still hits the right row after the indexes have shifted", () => {
        const shifted = insertSegmentAt(doc(), 1);
        const result = updateSegment(shifted, "c", (s) => ({
            ...s,
            text: "endret",
        }));

        expect(result.find((s) => s.uid === "c")!.text).toBe("endret");
        expect(result.find((s) => s.uid === "a")!.text).toBe("Hei alle");
    });

    it("is a no-op for an unknown uid", () => {
        expect(
            updateSegment(doc(), "nope", (s) => ({ ...s, text: "x" })),
        ).toEqual(doc());
    });
});

describe("toggleSegmentDeleted", () => {
    it("marks and unmarks without removing the row", () => {
        const marked = toggleSegmentDeleted(doc(), "b");
        expect(marked).toHaveLength(3);
        expect(marked[1]!.deleted).toBe(true);

        const unmarked = toggleSegmentDeleted(marked, "b");
        expect(unmarked[1]!.deleted).toBe(false);
    });

    it("keeps text edits made to other rows", () => {
        const edited = updateSegment(doc(), "a", (s) =>
            fixFirstWord(s, "Hallo"),
        );
        const afterDelete = toggleSegmentDeleted(edited, "c");

        expect(afterDelete.find((s) => s.uid === "a")!.text).toBe("Hallo alle");
    });
});

describe("insertSegmentAt", () => {
    it("fills the gap between two rows", () => {
        const result = insertSegmentAt(doc(), 1);

        expect(result).toHaveLength(4);
        expect([result[2]!.start, result[2]!.end]).toEqual([4, 10]);
        expect(result[2]!.text).toBe("");
    });

    it("inserts before the first row", () => {
        const result = insertSegmentAt(
            [segment("a", ["Hei"], 5, 6), segment("b", ["du"], 6, 7)],
            -1,
        );

        expect([result[0]!.start, result[0]!.end]).toEqual([0, 5]);
        expect(result[1]!.uid).toBe("a");
    });

    it("inserts after the last row", () => {
        const result = insertSegmentAt(doc(), 2);

        expect(result).toHaveLength(4);
        expect([result[3]!.start, result[3]!.end]).toEqual([12, 14]);
    });

    it("does not run past the end of the video", () => {
        const result = insertSegmentAt(doc(), 2, 12.5);

        expect([result[3]!.start, result[3]!.end]).toEqual([12, 12.5]);
    });

    it("borrows time from the next row when there is no gap (B5)", () => {
        const result = insertSegmentAt(doc(), 0);

        expect([result[1]!.start, result[1]!.end]).toEqual([2, 3.6]);
        expect(result[2]!.uid).toBe("b");
        expect(result[2]!.start).toBe(3.6);
        expect(result[2]!.end).toBe(4);
    });

    it("refuses to borrow from a row that is already short", () => {
        const tight = [
            segment("a", ["Hei"], 0, 1),
            segment("b", ["du"], 1, 1.2),
        ];
        const result = insertSegmentAt(tight, 0);

        expect(result[2]!.start).toBe(1);
        expect(result[1]!.start).toBe(result[1]!.end);
    });

    it("works on an empty document", () => {
        const result = insertSegmentAt([], -1);

        expect(result).toHaveLength(1);
        expect([result[0]!.start, result[0]!.end]).toEqual([0, 2]);
    });

    it("gives the new row its own uid and does not mutate the input", () => {
        const before = doc();
        const result = insertSegmentAt(before, 1);

        expect(new Set(result.map((s) => s.uid)).size).toBe(4);
        expect(before).toHaveLength(3);
    });
});

describe("toTranscription", () => {
    it("drops the client-only fields", () => {
        const [first] = toTranscription(doc()).segments;

        expect(first).not.toHaveProperty("uid");
        expect(first).not.toHaveProperty("deleted");
    });

    it("leaves out deleted rows", () => {
        const result = toTranscription(toggleSegmentDeleted(doc(), "b"));

        expect(result.segments.map((s) => s.text)).toEqual([
            "Hei alle",
            "i dag",
        ]);
    });

    it("leaves out rows whose text was emptied", () => {
        const emptied = updateSegment(doc(), "b", (s) => setSegmentText(s, ""));

        expect(toTranscription(emptied).segments).toHaveLength(2);
    });

    it("leaves out an inserted row that was never filled in", () => {
        expect(
            toTranscription(insertSegmentAt(doc(), 1)).segments,
        ).toHaveLength(3);
    });

    it("re-derives text that disagrees with the words", () => {
        const stale = doc().map((s) => ({ ...s, text: "utdatert" }));
        const result = toTranscription(stale);

        expect(result.segments.map((s) => s.text)).toEqual([
            "Hei alle",
            "sammen her",
            "i dag",
        ]);
        expect(result.text).toBe("Hei alle sammen her i dag");
    });

    it("re-derives the full text", () => {
        const edited = updateSegment(doc(), "a", (s) =>
            fixFirstWord(s, "Hallo"),
        );

        expect(toTranscription(edited).text).toBe(
            "Hallo alle sammen her i dag",
        );
    });

    it("carries an edit made before a delete all the way to the payload", () => {
        const edited = updateSegment(doc(), "a", (s) =>
            fixFirstWord(s, "Hallo"),
        );

        expect(toTranscription(toggleSegmentDeleted(edited, "c")).text).toBe(
            "Hallo alle sammen her",
        );
    });
});
