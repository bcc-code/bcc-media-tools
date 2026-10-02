import { describe, expect, it, vi } from "vitest";
import { useTranscriptionDraft } from "~/composables/useTranscriptionDraft";
import type { Segment } from "~/utils/transcription";
import {
    setSegmentText,
    setSegmentsDeleted,
    toggleSegmentDeleted,
} from "~/utils/transcription";
import type { DraftStorage } from "~/utils/transcriptionDraft";
import {
    draftKey,
    encodeDraft,
    readDraft,
    writeDraft,
} from "~/utils/transcriptionDraft";

const DEBOUNCE = 1;
const settled = () => new Promise((r) => setTimeout(r, DEBOUNCE + 5));

function rawSegment(text: string, start = 0, end = 1) {
    return {
        id: 0,
        seek: 0,
        start,
        end,
        text,
        tokens: [1, 2, 3],
        temperature: 0,
        avgLogprob: 0,
        compressionRatio: 0,
        noSpeechProb: 0,
        confidence: 1,
        words: [{ text, start, end, confidence: 1 }],
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

type FakeApi = {
    getTranscription: (req: { VXID: string }) => Promise<unknown>;
    submitTranscription: (req: unknown) => Promise<unknown>;
};

function fakeApi(overrides: Partial<FakeApi> = {}) {
    return {
        getTranscription: vi.fn(
            overrides.getTranscription ??
                (async () => ({
                    text: "Hei der",
                    segments: [
                        rawSegment("Hei", 0, 1),
                        rawSegment("der", 2, 3),
                    ],
                })),
        ),
        submitTranscription: vi.fn(
            overrides.submitTranscription ?? (async () => ({})),
        ),
    };
}

function setup(
    opts: { api?: ReturnType<typeof fakeApi>; storage?: DraftStorage } = {},
) {
    const api = opts.api ?? fakeApi();
    const storage = opts.storage ?? memoryStorage();
    const draft = useTranscriptionDraft("VX-123", {
        api: api as unknown as ReturnType<typeof useAPI>,
        storage,
        debounceMs: DEBOUNCE,
    });
    return { ...draft, api, storage };
}

/** Rewrites the first segment's text, the way the editor does. */
function edit(segments: { value: Segment[] }, text: string) {
    segments.value = segments.value.map((s, i) =>
        i === 0 ? setSegmentText(s, text) : s,
    );
}

describe("loading", () => {
    it("fetches from the API when there is no draft", async () => {
        const { load, segments, api, loading } = setup();
        await load();

        expect(api.getTranscription).toHaveBeenCalledWith({ VXID: "VX-123" });
        expect(segments.value.map((s) => s.text)).toEqual(["Hei", "der"]);
        expect(loading.value).toBe(false);
    });

    it("gives every loaded segment a uid", async () => {
        const { load, segments } = setup();
        await load();

        expect(segments.value.every((s) => Boolean(s.uid))).toBe(true);
    });

    it("prefers the local draft over the API", async () => {
        const storage = memoryStorage();
        const first = setup({ storage });
        await first.load();
        edit(first.segments, "Hallo");
        await settled();

        const second = setup({ storage });
        await second.load();

        expect(second.segments.value[0]!.text).toBe("Hallo");
        expect(second.api.getTranscription).not.toHaveBeenCalled();
    });

    it("reports an API failure without leaving the page loading", async () => {
        const api = fakeApi({
            getTranscription: async () => {
                throw new Error("[unknown] not enough permissions");
            },
        });
        const { load, error, loading } = setup({ api });
        await load();

        expect(error.value).toBe("Not enough permissions");
        expect(loading.value).toBe(false);
    });
});

describe("autosave", () => {
    it("writes the edit to storage", async () => {
        const { load, segments, storage } = setup();
        await load();
        edit(segments, "Hallo");
        await settled();

        expect(readDraft(storage, "VX-123")!.segments[0]!.text).toBe("Hallo");
    });

    it("survives a reload, which is what users were doing by hand", async () => {
        const storage = memoryStorage();
        const session = setup({ storage });
        await session.load();
        edit(session.segments, "Hallo");
        await settled();

        const reloaded = setup({ storage });
        await reloaded.load();

        expect(reloaded.segments.value[0]!.text).toBe("Hallo");
    });

    it("keeps edits made to other rows when one is deleted", async () => {
        const { load, segments } = setup();
        await load();
        edit(segments, "Hallo");
        segments.value = toggleSegmentDeleted(
            segments.value,
            segments.value[1]!.uid,
        );
        await settled();

        expect(segments.value[0]!.text).toBe("Hallo");
    });

    it("reports a failed write instead of losing it quietly", async () => {
        const storage: DraftStorage = {
            getItem: () => null,
            setItem: () => {
                throw new DOMException("full", "QuotaExceededError");
            },
            removeItem: () => {},
        };
        const { load, segments, saveState } = setup({ storage });
        await load();
        edit(segments, "Hallo");
        await settled();

        expect(saveState.value).toBe("error");
    });

    it("records when the save happened", async () => {
        const { load, segments, savedAt } = setup();
        await load();
        edit(segments, "Hallo");
        await settled();

        expect(savedAt.value).toBeInstanceOf(Date);
    });

    it("does not count restoring a draft as a save", async () => {
        const storage = memoryStorage();
        const first = setup({ storage });
        await first.load();
        edit(first.segments, "Hallo");
        await settled();
        const writtenAt = readDraft(storage, "VX-123")!.savedAt;

        const second = setup({ storage });
        await second.load();
        await settled();

        expect(readDraft(storage, "VX-123")!.savedAt).toBe(writtenAt);
        expect(second.savedAt.value!.toISOString()).toBe(writtenAt);
    });

    it("never writes an empty document over a good draft", async () => {
        const storage = memoryStorage();
        const { load, segments } = setup({ storage });
        await load();
        edit(segments, "Hallo");
        await settled();

        segments.value = [];
        await settled();

        expect(readDraft(storage, "VX-123")!.segments[0]!.text).toBe("Hallo");
    });
});

describe("reset", () => {
    it("replaces the draft with the server's copy", async () => {
        const { load, segments, reset, storage } = setup();
        await load();
        edit(segments, "Hallo");
        await settled();

        expect(await reset()).toBe(true);
        expect(segments.value[0]!.text).toBe("Hei");
        expect(readDraft(storage, "VX-123")!.segments[0]!.text).toBe("Hei");
    });

    it("leaves the work alone when the fetch fails", async () => {
        let calls = 0;
        const { load, segments, reset, error } = setup({
            api: fakeApi({
                getTranscription: async () => {
                    if (calls++ > 0) throw new Error("network");
                    return {
                        text: "Hei der",
                        segments: [rawSegment("Hei"), rawSegment("der", 2, 3)],
                    };
                },
            }),
        });
        await load();
        edit(segments, "Hallo");

        expect(await reset()).toBe(false);
        expect(segments.value[0]!.text).toBe("Hallo");
        expect(error.value).toBe("Network");
    });
});

describe("submit", () => {
    it("sends the edited text, not the original", async () => {
        const { load, segments, submit, api } = setup();
        await load();
        edit(segments, "Hallo");

        expect(await submit()).toBe(true);
        expect(api.submitTranscription).toHaveBeenCalledWith({
            VXID: "VX-123",
            transcription: expect.objectContaining({ text: "Hallo der" }),
        });
    });

    it("leaves out deleted rows and the client-only fields", async () => {
        const { load, segments, submit, api } = setup();
        await load();
        segments.value = toggleSegmentDeleted(
            segments.value,
            segments.value[1]!.uid,
        );
        await submit();

        const sent = api.submitTranscription.mock.calls[0]![0] as {
            transcription: { segments: Segment[] };
        };
        expect(sent.transcription.segments).toHaveLength(1);
        expect(sent.transcription.segments[0]).not.toHaveProperty("uid");
    });

    it("keeps the local draft, because submitting only starts the workflow", async () => {
        const { load, segments, submit, storage } = setup();
        await load();
        edit(segments, "Hallo");
        await submit();

        expect(storage.getItem(draftKey("VX-123"))).not.toBeNull();
        expect(readDraft(storage, "VX-123")!.segments[0]!.text).toBe("Hallo");
    });

    it("records that it was submitted", async () => {
        const { load, submit, submittedAt, storage } = setup();
        await load();
        await submit();

        expect(submittedAt.value).toBeInstanceOf(Date);
        expect(readDraft(storage, "VX-123")!.submittedAt).toBeTruthy();
    });

    it("reports a failure and does not mark the draft submitted", async () => {
        const { load, submit, submittedAt, error, storage } = setup({
            api: fakeApi({
                submitTranscription: async () => {
                    throw new Error("workflow unavailable");
                },
            }),
        });
        await load();

        expect(await submit()).toBe(false);
        expect(submittedAt.value).toBeNull();
        expect(error.value).toBe("Workflow unavailable");
        expect(readDraft(storage, "VX-123")!.submittedAt).toBeUndefined();
    });

    it("clears the submitting flag either way", async () => {
        const { load, submit, submitting } = setup({
            api: fakeApi({
                submitTranscription: async () => {
                    throw new Error("nope");
                },
            }),
        });
        await load();
        await submit();

        expect(submitting.value).toBe(false);
    });
});

// One drag in delete mode can mark a hundred rows, so taking it back has to be
// one step too.
describe("undo", () => {
    const markAll = (segments: { value: Segment[] }) => {
        segments.value = setSegmentsDeleted(
            segments.value,
            segments.value.map((s) => s.uid),
            true,
        );
    };

    it("has nothing to undo on a freshly loaded document", async () => {
        const { load, canUndo, undo } = setup();
        await load();

        expect(canUndo.value).toBe(false);
        expect(undo()).toBe(false);
    });

    it("takes back a bulk marking as a single step", async () => {
        const { load, segments, snapshot, undo, canUndo } = setup();
        await load();

        snapshot();
        markAll(segments);
        expect(segments.value.every((s) => s.deleted)).toBe(true);

        expect(undo()).toBe(true);
        expect(segments.value.some((s) => s.deleted)).toBe(false);
        expect(canUndo.value).toBe(false);
    });

    it("unwinds several markings one at a time, most recent first", async () => {
        const { load, segments, snapshot, undo } = setup();
        await load();

        snapshot();
        segments.value = toggleSegmentDeleted(
            segments.value,
            segments.value[0]!.uid,
        );
        snapshot();
        segments.value = toggleSegmentDeleted(
            segments.value,
            segments.value[1]!.uid,
        );

        undo();
        expect(segments.value.map((s) => !!s.deleted)).toEqual([true, false]);

        undo();
        expect(segments.value.map((s) => !!s.deleted)).toEqual([false, false]);
    });

    it("keeps the restored document, so it is saved like any other change", async () => {
        const storage = memoryStorage();
        const { load, segments, snapshot, undo } = setup({ storage });
        await load();

        snapshot();
        markAll(segments);
        await settled();
        undo();
        await settled();

        const saved = readDraft(storage, "VX-123");
        expect(saved!.segments.some((s) => s.deleted)).toBe(false);
    });

    it("drops the history when the document is replaced wholesale", async () => {
        const { load, segments, snapshot, reset, canUndo } = setup();
        await load();

        snapshot();
        markAll(segments);
        expect(canUndo.value).toBe(true);

        // Undoing back into a document the user has discarded would be worse
        // than having no undo at all.
        await reset();
        expect(canUndo.value).toBe(false);
    });

    it("forgets the oldest steps rather than growing without bound", async () => {
        const { load, segments, snapshot, undo, canUndo } = setup();
        await load();

        for (let i = 0; i < 60; i++) {
            snapshot();
            segments.value = toggleSegmentDeleted(
                segments.value,
                segments.value[0]!.uid,
            );
        }

        let steps = 0;
        while (canUndo.value && undo()) steps++;
        expect(steps).toBe(50);
    });

    it("reports what is marked, for the count in the header", async () => {
        const { load, segments, deletedCount } = setup();
        await load();

        expect(deletedCount.value).toBe(0);
        markAll(segments);
        expect(deletedCount.value).toBe(2);
    });
});

describe("housekeeping", () => {
    const DAY = 24 * 60 * 60 * 1000;
    /** Bulky, so that one of these fills a quota two small drafts would fit in. */
    const old = (submitted: boolean) => ({
        segments: Array.from({ length: 20 }, (_, i) => ({
            ...rawSegment("Gammel"),
            uid: `x${i}`,
        })),
        savedAt: new Date(Date.now() - 30 * DAY).toISOString(),
        submittedAt: submitted
            ? new Date(Date.now() - 30 * DAY).toISOString()
            : undefined,
    });

    it("clears out old submitted drafts when the editor opens", async () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-done", old(true));
        const { load } = setup({ storage });

        await load();

        expect(storage.getItem(draftKey("VX-done"))).toBeNull();
    });

    it("leaves another asset's unsubmitted work alone", async () => {
        const storage = memoryStorage();
        writeDraft(storage, "VX-wip", old(false));
        const { load } = setup({ storage });

        await load();

        expect(storage.getItem(draftKey("VX-wip"))).not.toBeNull();
    });

    it("saves anyway by making room when the quota is full", async () => {
        const store = new Map<string, string>();
        const used = (without: string) =>
            [...store.entries()]
                .filter(([k]) => k !== without)
                .reduce((n, [, v]) => n + v.length, 0);
        // Room for the bulky submitted draft, or for the one being edited —
        // not both.
        const limit = encodeDraft(old(true)).length;
        const storage: DraftStorage = {
            getItem: (k) => store.get(k) ?? null,
            setItem: (k, v) => {
                if (used(k) + v.length > limit)
                    throw new DOMException("full", "QuotaExceededError");
                store.set(k, v);
            },
            removeItem: (k) => void store.delete(k),
            key: (i) => [...store.keys()][i] ?? null,
            get length() {
                return store.size;
            },
        };
        // Submitted a while ago, but not old enough for the prune on open.
        writeDraft(storage, "VX-done", {
            ...old(true),
            savedAt: new Date().toISOString(),
            submittedAt: new Date().toISOString(),
        });

        const { load, segments, saveState } = setup({ storage });
        await load();
        edit(segments, "Hallo");
        await settled();

        expect(saveState.value).toBe("saved");
        expect(readDraft(storage, "VX-123")!.segments[0]!.text).toBe("Hallo");
        expect(storage.getItem(draftKey("VX-done"))).toBeNull();
    });
});
