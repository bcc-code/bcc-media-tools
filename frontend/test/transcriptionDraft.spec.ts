import { describe, expect, it, vi } from "vitest";
import type { Segment } from "~/utils/transcription";
import type { DraftStorage } from "~/utils/transcriptionDraft";
import {
    decodeDraft,
    draftKey,
    encodeDraft,
    readDraft,
    removeDraft,
    writeDraft,
} from "~/utils/transcriptionDraft";

function segment(uid: string, text: string): Segment {
    return {
        uid,
        id: 0,
        seek: 0,
        start: 0,
        end: 1,
        text,
        tokens: Array.from({ length: 200 }, (_, i) => i),
        temperature: 0,
        avg_logprob: 0,
        compression_ration: 0,
        no_speech_prob: 0,
        confidence: 1,
        words: [{ text, start: 0, end: 1, confidence: 1 }],
    };
}

function memoryStorage(initial: Record<string, string> = {}): DraftStorage {
    const store = new Map(Object.entries(initial));
    return {
        getItem: (k) => store.get(k) ?? null,
        setItem: (k, v) => void store.set(k, v),
        removeItem: (k) => void store.delete(k),
    };
}

function throwingStorage(error: unknown): DraftStorage {
    return {
        getItem: () => {
            throw error;
        },
        setItem: () => {
            throw error;
        },
        removeItem: () => {
            throw error;
        },
    };
}

const quotaError = () => new DOMException("out of space", "QuotaExceededError");

const draft = () => ({
    segments: [segment("a", "Hei"), segment("b", "der")],
    savedAt: "2025-10-20T17:26:44.000Z",
});

describe("encodeDraft", () => {
    it("leaves out the tokens", () => {
        const parsed = JSON.parse(encodeDraft(draft()));

        expect(parsed.segments[0].tokens).toEqual([]);
    });

    it("is substantially smaller than the document with tokens", () => {
        const withTokens = JSON.stringify(draft());

        expect(encodeDraft(draft()).length).toBeLessThan(withTokens.length / 2);
    });

    it("keeps the text and the words", () => {
        const parsed = JSON.parse(encodeDraft(draft()));

        expect(parsed.segments.map((s: Segment) => s.text)).toEqual([
            "Hei",
            "der",
        ]);
        expect(parsed.segments[0].words[0].text).toBe("Hei");
    });
});

describe("decodeDraft", () => {
    it("round-trips a draft", () => {
        const decoded = decodeDraft(encodeDraft(draft()));

        expect(decoded!.segments.map((s) => s.text)).toEqual(["Hei", "der"]);
        expect(decoded!.savedAt).toBe("2025-10-20T17:26:44.000Z");
    });

    it("gives uids to a draft written before they existed", () => {
        const legacy = JSON.stringify({
            segments: [{ ...segment("a", "Hei"), uid: undefined }],
        });

        expect(decodeDraft(legacy)!.segments[0]!.uid).toBeTruthy();
    });

    it.each([
        ["null", null],
        ["empty string", ""],
        ["invalid json", "{not json"],
        ["a non-object", '"hello"'],
        ["no segments", "{}"],
        ["an empty document", '{"segments":[]}'],
    ])("returns null for %s", (_label, raw) => {
        expect(decodeDraft(raw)).toBeNull();
    });
});

describe("writeDraft", () => {
    it("stores the draft under the asset's key", () => {
        const storage = memoryStorage();

        expect(writeDraft(storage, "VX-123", draft())).toEqual({ ok: true });
        expect(storage.getItem(draftKey("VX-123"))).toBeTruthy();
    });

    it("reports a full quota instead of throwing", () => {
        // This is what failed silently before: the error escaped into Vue's
        // scheduler and the user was never told nothing was being saved.
        const result = writeDraft(
            throwingStorage(quotaError()),
            "VX-123",
            draft(),
        );

        expect(result).toMatchObject({ ok: false, reason: "quota" });
    });

    it("reports other storage failures as unavailable", () => {
        const result = writeDraft(
            throwingStorage(new Error("disabled")),
            "VX-123",
            draft(),
        );

        expect(result).toMatchObject({ ok: false, reason: "unavailable" });
    });

    it("keeps drafts for different assets apart", () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-1", draft());
        writeDraft(storage, "VX-2", {
            ...draft(),
            segments: [segment("z", "annen")],
        });

        expect(readDraft(storage, "VX-1")!.segments).toHaveLength(2);
        expect(readDraft(storage, "VX-2")!.segments).toHaveLength(1);
    });
});

describe("readDraft", () => {
    it("returns null when there is nothing stored", () => {
        expect(readDraft(memoryStorage(), "VX-123")).toBeNull();
    });

    it("returns null rather than throwing when storage is unusable", () => {
        expect(
            readDraft(throwingStorage(new Error("blocked")), "VX-123"),
        ).toBeNull();
    });
});

describe("removeDraft", () => {
    it("removes only the asset's own draft", () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-1", draft());
        writeDraft(storage, "VX-2", draft());

        removeDraft(storage, "VX-1");

        expect(readDraft(storage, "VX-1")).toBeNull();
        expect(readDraft(storage, "VX-2")).not.toBeNull();
    });

    it("does not throw when storage is unusable", () => {
        expect(() =>
            removeDraft(throwingStorage(new Error("blocked")), "VX-1"),
        ).not.toThrow();
    });

    it("leaves unrelated keys alone", () => {
        const storage = memoryStorage({ seekOnFocus: "true" });
        removeDraft(storage, "VX-1");

        expect(storage.getItem("seekOnFocus")).toBe("true");
    });
});

describe("draftKey", () => {
    it("is namespaced per asset", () => {
        expect(draftKey("VX-123")).toBe("ts-VX-123");
        expect(draftKey("VX-123")).not.toBe(draftKey("VX-124"));
    });
});

describe("storage contract", () => {
    it("only ever touches its own key", () => {
        const setItem = vi.fn();
        const storage: DraftStorage = {
            getItem: () => null,
            setItem,
            removeItem: vi.fn(),
        };

        writeDraft(storage, "VX-9", draft());

        expect(setItem).toHaveBeenCalledWith("ts-VX-9", expect.any(String));
    });
});
