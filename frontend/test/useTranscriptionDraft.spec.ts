import { describe, expect, it, vi } from "vitest";
import { useTranscriptionDraft } from "~/composables/useTranscriptionDraft";
import type { Segment } from "~/utils/transcription";
import { setSegmentText, toggleSegmentDeleted } from "~/utils/transcription";
import type { DraftStorage } from "~/utils/transcriptionDraft";
import { draftKey, readDraft } from "~/utils/transcriptionDraft";

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
        avg_logprob: 0,
        compression_ration: 0,
        no_speech_prob: 0,
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
