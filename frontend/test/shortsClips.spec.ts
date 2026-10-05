import { describe, expect, it } from "vitest";
import {
    DEFAULT_CLIP_SECONDS,
    clipLength,
    formatClock,
    isValidClip,
    nextClipRange,
    overlappingClipIds,
    sanitiseClips,
    sortClips,
} from "~/utils/shortsClips";

const clip = (id: string, start: number, end: number) => ({ id, start, end });

describe("nextClipRange", () => {
    it("starts at the playhead", () => {
        expect(nextClipRange(100, 600)).toEqual({
            start: 100,
            end: 100 + DEFAULT_CLIP_SECONDS,
        });
    });

    it("pulls back from the end when there is not a full clip left", () => {
        expect(nextClipRange(595, 600)).toEqual({
            start: 600 - DEFAULT_CLIP_SECONDS,
            end: 600,
        });
    });

    it("spans the whole asset when it is shorter than a clip", () => {
        expect(nextClipRange(5, 10)).toEqual({ start: 0, end: 10 });
    });

    it("never starts before zero", () => {
        expect(nextClipRange(-20, 600)?.start).toBe(0);
    });

    it("is undefined before the duration is known", () => {
        expect(nextClipRange(0, 0)).toBeUndefined();
    });
});

describe("overlappingClipIds", () => {
    it("finds both sides of an overlap", () => {
        const ids = overlappingClipIds([
            clip("a", 0, 30),
            clip("b", 20, 50),
            clip("c", 100, 120),
        ]);

        expect([...ids].sort()).toEqual(["a", "b"]);
    });

    it("does not count clips that merely touch", () => {
        expect(
            overlappingClipIds([clip("a", 0, 30), clip("b", 30, 60)]).size,
        ).toBe(0);
    });

    it("finds an overlap regardless of order", () => {
        const ids = overlappingClipIds([clip("b", 20, 50), clip("a", 0, 30)]);

        expect([...ids].sort()).toEqual(["a", "b"]);
    });

    it("has nothing to report for one clip", () => {
        expect(overlappingClipIds([clip("a", 0, 30)]).size).toBe(0);
    });
});

describe("sortClips", () => {
    it("orders by start, then by end", () => {
        const sorted = sortClips([
            clip("c", 50, 60),
            clip("a", 10, 40),
            clip("b", 10, 20),
        ]);

        expect(sorted.map((c) => c.id)).toEqual(["b", "a", "c"]);
    });

    it("leaves the input alone", () => {
        const input = [clip("b", 50, 60), clip("a", 10, 20)];
        sortClips(input);

        expect(input.map((c) => c.id)).toEqual(["b", "a"]);
    });
});

describe("sanitiseClips", () => {
    it("keeps well-formed clips", () => {
        expect(sanitiseClips([clip("a", 10, 40)], 600)).toEqual([
            clip("a", 10, 40),
        ]);
    });

    it("drops anything that is not an array", () => {
        expect(sanitiseClips(null)).toEqual([]);
        expect(sanitiseClips("nope")).toEqual([]);
        expect(sanitiseClips({ a: 1 })).toEqual([]);
    });

    it("drops entries with missing or wrong-typed fields", () => {
        const result = sanitiseClips(
            [
                { id: "a", start: 0 },
                { id: "b", start: "x", end: 10 },
                { start: 0, end: 10 },
                null,
                "clip",
                clip("ok", 0, 10),
            ],
            600,
        );

        expect(result.map((c) => c.id)).toEqual(["ok"]);
    });

    it("drops duplicate ids, keeping the first", () => {
        const result = sanitiseClips(
            [clip("a", 0, 10), clip("a", 20, 30)],
            600,
        );

        expect(result).toEqual([clip("a", 0, 10)]);
    });

    it("drops zero-length and inverted clips", () => {
        const result = sanitiseClips(
            [clip("a", 10, 10), clip("b", 40, 20), clip("c", 0, 5)],
            600,
        );

        expect(result.map((c) => c.id)).toEqual(["c"]);
    });

    it("clamps a clip that runs past a shorter asset", () => {
        expect(sanitiseClips([clip("a", 10, 900)], 600)).toEqual([
            clip("a", 10, 600),
        ]);
    });

    it("drops a clip that starts beyond the asset entirely", () => {
        expect(sanitiseClips([clip("a", 700, 900)], 600)).toEqual([]);
    });

    it("drops non-finite numbers", () => {
        const result = sanitiseClips([
            { id: "a", start: NaN, end: 10 },
            { id: "b", start: 0, end: Infinity },
        ]);

        expect(result).toEqual([]);
    });
});

describe("clipLength and isValidClip", () => {
    it("measures a clip", () => {
        expect(clipLength(clip("a", 10, 40))).toBe(30);
    });

    it("never reports a negative length", () => {
        expect(clipLength(clip("a", 40, 10))).toBe(0);
    });

    it("rejects clips with no length", () => {
        expect(isValidClip(clip("a", 10, 10))).toBe(false);
        expect(isValidClip(clip("a", 40, 10))).toBe(false);
        expect(isValidClip(clip("a", 10, 40))).toBe(true);
    });
});

describe("formatClock", () => {
    it("formats as m:ss", () => {
        expect(formatClock(0)).toBe("0:00");
        expect(formatClock(9)).toBe("0:09");
        expect(formatClock(75)).toBe("1:15");
    });

    it("does not wrap minutes into hours", () => {
        expect(formatClock(4512)).toBe("75:12");
    });

    it("treats negatives as zero", () => {
        expect(formatClock(-5)).toBe("0:00");
    });
});
