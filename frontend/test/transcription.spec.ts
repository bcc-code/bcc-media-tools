import { describe, expect, it } from "vitest";
import type { Segment, Word } from "~/utils/transcription";
import {
    canInsertAfter,
    insertSegmentAfter,
    segmentText,
    setWordText,
    toTranscription,
    toggleSegmentDeleted,
    updateSegment,
    withUids,
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

describe("setWordText", () => {
    it("changes only the addressed word and re-derives the text", () => {
        const result = setWordText(segment("a", ["Kåre", "sin"]), 0, "Kaare");

        expect(result.words.map((w) => w.text)).toEqual(["Kaare", "sin"]);
        expect(result.text).toBe("Kaare sin");
    });

    it("keeps the word timings", () => {
        const before = segment("a", ["Kåre", "sin"], 0, 2);
        const after = setWordText(before, 0, "Kaare");

        expect(after.words[0]!.start).toBe(before.words[0]!.start);
        expect(after.words[0]!.end).toBe(before.words[0]!.end);
    });

    it("does not mutate the input", () => {
        const before = segment("a", ["Kåre"]);
        setWordText(before, 0, "Kaare");

        expect(before.words[0]!.text).toBe("Kåre");
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
        // The original bug: rows were addressed by index into one array while
        // the edit was written into another, so an insert made edits land on a
        // different row than the one the user typed in.
        const shifted = insertSegmentAfter(doc(), 1);
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
        // Deleting used to rebuild the list from the untouched original, which
        // silently reverted every correction the user had made.
        const edited = updateSegment(doc(), "a", (s) =>
            setWordText(s, 0, "Hallo"),
        );
        const afterDelete = toggleSegmentDeleted(edited, "c");

        expect(afterDelete.find((s) => s.uid === "a")!.text).toBe("Hallo alle");
    });
});

describe("canInsertAfter", () => {
    it("allows an insert where there is a gap", () => {
        expect(canInsertAfter(doc(), 1)).toBe(true);
    });

    it("refuses where the rows are back to back (B5)", () => {
        expect(canInsertAfter(doc(), 0)).toBe(false);
    });

    it("refuses after the last row (B5)", () => {
        expect(canInsertAfter(doc(), 2)).toBe(false);
    });
});

describe("insertSegmentAfter", () => {
    it("inserts an empty segment filling the gap", () => {
        const result = insertSegmentAfter(doc(), 1);

        expect(result).toHaveLength(4);
        expect(result[2]!.start).toBe(4);
        expect(result[2]!.end).toBe(10);
        expect(result[2]!.text).toBe("");
    });

    it("gives the new segment its own uid", () => {
        const result = insertSegmentAfter(doc(), 1);
        const uids = result.map((s) => s.uid);

        expect(new Set(uids).size).toBe(uids.length);
    });

    it("is a no-op past the end", () => {
        expect(insertSegmentAfter(doc(), 2)).toHaveLength(3);
    });

    it("does not mutate the input", () => {
        const before = doc();
        insertSegmentAfter(before, 1);

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
        const emptied = updateSegment(doc(), "b", (s) =>
            setWordText(setWordText(s, 0, ""), 1, ""),
        );

        expect(toTranscription(emptied).segments).toHaveLength(2);
    });

    it("leaves out an inserted row that was never filled in", () => {
        expect(
            toTranscription(insertSegmentAfter(doc(), 1)).segments,
        ).toHaveLength(3);
    });

    it("re-derives each segment's text from its words", () => {
        const edited = updateSegment(doc(), "a", (s) =>
            setWordText(s, 0, "Hallo"),
        );

        expect(toTranscription(edited).segments[0]!.text).toBe("Hallo alle");
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
            setWordText(s, 0, "Hallo"),
        );

        expect(toTranscription(edited).text).toBe(
            "Hallo alle sammen her i dag",
        );
    });

    it("carries an edit made before a delete all the way to the payload", () => {
        const edited = updateSegment(doc(), "a", (s) =>
            setWordText(s, 0, "Hallo"),
        );

        expect(toTranscription(toggleSegmentDeleted(edited, "c")).text).toBe(
            "Hallo alle sammen her",
        );
    });
});
