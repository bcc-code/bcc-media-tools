import type { DraftStorage } from "~/utils/transcriptionDraft";
import { readDraft, writeDraft } from "~/utils/transcriptionDraft";
import type { Segment } from "~/utils/transcription";
import { toTranscription, withUids } from "~/utils/transcription";

export type SaveState = "idle" | "saved" | "error";

export type UseTranscriptionDraftOptions = {
    api?: ReturnType<typeof useAPI>;
    storage?: DraftStorage;
    debounceMs?: number;
};

/**
 * Owns the transcription document for one asset.
 *
 * There is deliberately one array of segments: the editor renders it, the draft
 * is written from it and the submitted payload is derived from it. Keeping a
 * second copy is what previously made edits disappear.
 */
export function useTranscriptionDraft(
    vxid: string,
    options: UseTranscriptionDraftOptions = {},
) {
    const api = options.api ?? useAPI();

    const segments = ref<Segment[]>([]);
    const loading = ref(true);
    const error = ref<string | null>(null);

    const saveState = ref<SaveState>("idle");
    const savedAt = ref<Date | null>(null);
    const submittedAt = ref<Date | null>(null);
    const submitting = ref(false);

    const getStorage = (): DraftStorage | undefined => {
        if (options.storage) return options.storage;
        if (!import.meta.client) return undefined;
        try {
            return window.localStorage;
        } catch {
            return undefined;
        }
    };

    const persist = () => {
        const storage = getStorage();
        // An empty document must never overwrite a good draft.
        if (!storage || segments.value.length === 0) return;

        const now = new Date();
        const result = writeDraft(storage, vxid, {
            segments: segments.value,
            submittedAt: submittedAt.value?.toISOString(),
            savedAt: now.toISOString(),
        });

        if (result.ok) {
            saveState.value = "saved";
            savedAt.value = now;
        } else {
            saveState.value = "error";
            console.error("Could not save transcription draft", result.error);
        }
    };

    const persistSoon = useDebounceFn(persist, options.debounceMs ?? 500);

    // Loading a draft must not count as a save, or the indicator would show a
    // time at which nothing was written.
    let hydrating = false;
    const hydrate = (next: Segment[]) => {
        hydrating = true;
        segments.value = next;
    };

    watch(segments, () => {
        if (loading.value) return;
        if (hydrating) {
            hydrating = false;
            return;
        }
        persistSoon();
    });

    const reset = async (): Promise<boolean> => {
        loading.value = true;
        error.value = null;
        try {
            const result = await api.getTranscription({ VXID: vxid });
            segments.value = withUids(result.segments as unknown as Segment[]);
            submittedAt.value = null;
            loading.value = false;
            persist();
            return true;
        } catch (e: unknown) {
            error.value = errorMessage(e);
            loading.value = false;
            // `segments` is left alone: a failed reset must not destroy work.
            return false;
        }
    };

    const load = async (): Promise<void> => {
        const storage = getStorage();
        const draft = storage ? readDraft(storage, vxid) : null;

        if (!draft) {
            await reset();
            return;
        }

        hydrate(draft.segments);
        submittedAt.value = draft.submittedAt
            ? new Date(draft.submittedAt)
            : null;
        savedAt.value = new Date(draft.savedAt);
        saveState.value = "saved";
        loading.value = false;
    };

    const submit = async (): Promise<boolean> => {
        submitting.value = true;
        try {
            await api.submitTranscription({
                VXID: vxid,
                transcription: toTranscription(segments.value),
            });
            // The draft is kept: submitting only *starts* the import workflow,
            // and the user loses access to the asset either way.
            submittedAt.value = new Date();
            persist();
            return true;
        } catch (e: unknown) {
            error.value = errorMessage(e);
            return false;
        } finally {
            submitting.value = false;
        }
    };

    return {
        segments,
        loading,
        error,
        saveState,
        savedAt,
        submittedAt,
        submitting,
        load,
        reset,
        submit,
    };
}

function errorMessage(e: unknown): string {
    const raw =
        (e as { message?: string })?.message ||
        (e as object)?.toString?.() ||
        "Unknown error";
    const msg = raw.replace(/^\[.*?\]\s*/, "");
    return msg.charAt(0).toUpperCase() + msg.slice(1);
}
