import type { Segment } from "./transcription";
import { withUids } from "./transcription";

// Storage is passed in rather than reaching for `localStorage`, so this is
// usable outside a browser.

export type TranscriptionDraft = {
    segments: Segment[];
    submittedAt?: string;
    savedAt: string;
};

export type DraftStorage = Pick<Storage, "getItem" | "setItem" | "removeItem">;

export type WriteResult =
    | { ok: true }
    | { ok: false; reason: "quota" | "unavailable"; error: unknown };

export function draftKey(vxid: string): string {
    return `ts-${vxid}`;
}

/**
 * `tokens` are dropped: they are the bulk of the payload and nothing in this
 * repo or in bcc-media-flows reads them. Keeping them pushed long
 * transcriptions over the localStorage quota, which failed silently.
 */
export function encodeDraft(draft: TranscriptionDraft): string {
    return JSON.stringify({
        ...draft,
        segments: draft.segments.map((s) => ({ ...s, tokens: [] })),
    });
}

export function decodeDraft(raw: string | null): TranscriptionDraft | null {
    if (!raw) return null;

    let parsed: unknown;
    try {
        parsed = JSON.parse(raw);
    } catch {
        return null;
    }

    if (!parsed || typeof parsed !== "object") return null;

    const segments = (parsed as { segments?: unknown }).segments;
    if (!Array.isArray(segments) || segments.length === 0) return null;

    const { submittedAt, savedAt } = parsed as Record<string, unknown>;

    return {
        segments: withUids(segments),
        submittedAt: typeof submittedAt === "string" ? submittedAt : undefined,
        savedAt:
            typeof savedAt === "string" ? savedAt : new Date().toISOString(),
    };
}

export function readDraft(
    storage: DraftStorage,
    vxid: string,
): TranscriptionDraft | null {
    try {
        return decodeDraft(storage.getItem(draftKey(vxid)));
    } catch {
        return null;
    }
}

/** Reports failure rather than throwing, so a full quota cannot go unnoticed. */
export function writeDraft(
    storage: DraftStorage,
    vxid: string,
    draft: TranscriptionDraft,
): WriteResult {
    try {
        storage.setItem(draftKey(vxid), encodeDraft(draft));
        return { ok: true };
    } catch (error) {
        return {
            ok: false,
            reason: quotaExceeded(error) ? "quota" : "unavailable",
            error,
        };
    }
}

export function removeDraft(storage: DraftStorage, vxid: string) {
    try {
        storage.removeItem(draftKey(vxid));
    } catch {
        // Already unreachable.
    }
}

function quotaExceeded(error: unknown): boolean {
    return (
        error instanceof DOMException &&
        (error.name === "QuotaExceededError" ||
            error.name === "NS_ERROR_DOM_QUOTA_REACHED" ||
            error.code === 22)
    );
}
