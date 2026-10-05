import { describe, expect, it, vi } from "vitest";
import type { Segment } from "~/utils/transcription";
import type { DraftStorage } from "~/utils/transcriptionDraft";
import {
    decodeDraft,
    draftKey,
    encodeDraft,
    listDrafts,
    pruneDrafts,
    readDraft,
    removeDraft,
    storeDraft,
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
        avgLogprob: 0,
        compressionRatio: 0,
        noSpeechProb: 0,
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
        key: (i) => [...store.keys()][i] ?? null,
        get length() {
            return store.size;
        },
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

const DAY = 24 * 60 * 60 * 1000;
const NOW = Date.parse("2025-10-20T12:00:00.000Z");

/** A draft saved `daysAgo`, submitted or not. */
const aged = (daysAgo: number, submitted: boolean) => ({
    segments: [segment("a", "Hei")],
    savedAt: new Date(NOW - daysAgo * DAY).toISOString(),
    submittedAt: submitted
        ? new Date(NOW - daysAgo * DAY).toISOString()
        : undefined,
});

/**
 * Storage that refuses to hold more than `limit` characters, the way a full
 * localStorage does.
 */
function quotaStorage(limit: number): DraftStorage {
    const store = new Map<string, string>();
    const used = (without: string) =>
        [...store.entries()]
            .filter(([k]) => k !== without)
            .reduce((n, [k, v]) => n + k.length + v.length, 0);

    return {
        getItem: (k) => store.get(k) ?? null,
        setItem: (k, v) => {
            if (used(k) + k.length + v.length > limit) throw quotaError();
            store.set(k, v);
        },
        removeItem: (k) => void store.delete(k),
        key: (i) => [...store.keys()][i] ?? null,
        get length() {
            return store.size;
        },
    };
}

describe("listDrafts", () => {
    it("finds every draft, newest first", () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-old", aged(5, true));
        writeDraft(storage, "VX-new", aged(1, true));

        expect(listDrafts(storage).map((d) => d.vxid)).toEqual([
            "VX-new",
            "VX-old",
        ]);
    });

    it("reports whether a draft was submitted", () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-1", aged(1, true));
        writeDraft(storage, "VX-2", aged(1, false));

        const byId = Object.fromEntries(
            listDrafts(storage).map((d) => [d.vxid, d.submitted]),
        );
        expect(byId).toEqual({ "VX-1": true, "VX-2": false });
    });

    it("ignores keys belonging to other features", () => {
        const storage = memoryStorage({
            seekOnFocus: "true",
            splitterSize: "[]",
        });
        writeDraft(storage, "VX-1", draft());

        expect(listDrafts(storage).map((d) => d.vxid)).toEqual(["VX-1"]);
    });

    it("marks a blob that cannot be decoded as unusable", () => {
        const storage = memoryStorage({ "ts-VX-9": "{not json" });

        expect(listDrafts(storage)[0]).toMatchObject({
            vxid: "VX-9",
            usable: false,
        });
    });

    it("is empty when the storage cannot be enumerated", () => {
        const storage: DraftStorage = {
            getItem: () => null,
            setItem: () => {},
            removeItem: () => {},
        };

        expect(listDrafts(storage)).toEqual([]);
    });
});

describe("pruneDrafts", () => {
    it("drops submitted drafts that are past their age", () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-stale", aged(8, true));
        writeDraft(storage, "VX-fresh", aged(1, true));

        expect(pruneDrafts(storage, { now: NOW })).toEqual(["VX-stale"]);
        expect(readDraft(storage, "VX-fresh")).not.toBeNull();
    });

    it("never drops work that was not submitted, however old", () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-1", aged(400, false));

        expect(pruneDrafts(storage, { now: NOW })).toEqual([]);
        expect(readDraft(storage, "VX-1")).not.toBeNull();
    });

    it("keeps the asset being opened", () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-1", aged(90, true));

        expect(pruneDrafts(storage, { keep: "VX-1", now: NOW })).toEqual([]);
        expect(readDraft(storage, "VX-1")).not.toBeNull();
    });

    it("keeps only the newest submitted drafts once over the cap", () => {
        const storage = memoryStorage();
        for (let i = 1; i <= 5; i++)
            writeDraft(storage, `VX-${i}`, aged(i, true));

        const removed = pruneDrafts(storage, { maxSubmitted: 2, now: NOW });

        expect(removed.sort()).toEqual(["VX-3", "VX-4", "VX-5"]);
        expect(listDrafts(storage).map((d) => d.vxid)).toEqual([
            "VX-1",
            "VX-2",
        ]);
    });

    it("always removes blobs that cannot be decoded", () => {
        const storage = memoryStorage({ "ts-VX-9": "{not json" });

        expect(pruneDrafts(storage, { now: NOW })).toEqual(["VX-9"]);
        expect(storage.getItem("ts-VX-9")).toBeNull();
    });

    it("leaves unrelated keys alone", () => {
        const storage = memoryStorage({ seekOnFocus: "true" });
        writeDraft(storage, "VX-1", aged(90, true));

        pruneDrafts(storage, { now: NOW });

        expect(storage.getItem("seekOnFocus")).toBe("true");
    });
});

describe("storeDraft", () => {
    it("writes without evicting anything when there is room", () => {
        const storage = quotaStorage(100_000);

        const result = storeDraft(storage, "VX-1", draft());

        expect(result).toMatchObject({ ok: true, evicted: [] });
    });

    it("makes room by dropping submitted drafts when the quota is full", () => {
        // Only one draft fits at a time, so the write can only succeed by
        // evicting the one already there.
        const storage = quotaStorage(
            encodeDraft(aged(5, true)).length + draftKey("VX-old").length,
        );
        writeDraft(storage, "VX-old", aged(5, true));

        const result = storeDraft(storage, "VX-new", aged(0, false));

        expect(result.ok).toBe(true);
        expect(result.evicted).toEqual(["VX-old"]);
        expect(readDraft(storage, "VX-new")).not.toBeNull();
        expect(readDraft(storage, "VX-old")).toBeNull();
    });

    it("evicts the longest-submitted first and stops as soon as it fits", () => {
        const one = encodeDraft(aged(1, true)).length + draftKey("VX-1").length;
        const storage = quotaStorage(one * 3);
        writeDraft(storage, "VX-1", aged(1, true));
        writeDraft(storage, "VX-2", aged(5, true));
        writeDraft(storage, "VX-3", aged(3, true));

        const result = storeDraft(storage, "VX-4", aged(0, false));

        expect(result.ok).toBe(true);
        expect(result.evicted).toEqual(["VX-2"]);
        expect(readDraft(storage, "VX-1")).not.toBeNull();
        expect(readDraft(storage, "VX-3")).not.toBeNull();
    });

    it("will not sacrifice work that has not been submitted", () => {
        const one =
            encodeDraft(aged(1, false)).length + draftKey("VX-1").length;
        const storage = quotaStorage(one);
        writeDraft(storage, "VX-1", aged(1, false));

        const result = storeDraft(storage, "VX-2", draft());

        expect(result).toMatchObject({
            ok: false,
            reason: "quota",
            evicted: [],
        });
        expect(readDraft(storage, "VX-1")).not.toBeNull();
    });

    it("reports the quota failure when evicting everything is still not enough", () => {
        const storage = quotaStorage(10);
        const result = storeDraft(storage, "VX-1", draft());

        expect(result).toMatchObject({ ok: false, reason: "quota" });
    });

    it("does not evict for a failure that is not about space", () => {
        const storage = throwingStorage(new Error("blocked"));

        expect(storeDraft(storage, "VX-1", draft())).toMatchObject({
            ok: false,
            reason: "unavailable",
            evicted: [],
        });
    });
});
