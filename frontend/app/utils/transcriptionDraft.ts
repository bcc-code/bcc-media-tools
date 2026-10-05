import type { Segment } from "./transcription";
import { withUids } from "./transcription";

// Storage is passed in rather than reaching for `localStorage`, so this is
// usable outside a browser. `key`/`length` are optional because only the
// pruning functions need to enumerate, and a storage that cannot be enumerated
// is left alone rather than guessed at.

export type TranscriptionDraft = {
    segments: Segment[];
    submittedAt?: string;
    savedAt: string;
};

export type DraftStorage = Pick<Storage, "getItem" | "setItem" | "removeItem"> &
    Partial<Pick<Storage, "key" | "length">>;

export type WriteResult =
    | { ok: true }
    | { ok: false; reason: "quota" | "unavailable"; error: unknown };

/** Submitted drafts are kept this long, so a reload still shows the work. */
export const DRAFT_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000;

/** And no more than this many at once, however recent. */
export const DRAFT_MAX_SUBMITTED = 10;

const KEY_PREFIX = "ts-";

export function draftKey(vxid: string): string {
    return `${KEY_PREFIX}${vxid}`;
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

/** What is in storage for one asset, without paying to decode the segments. */
export type DraftSummary = {
    vxid: string;
    /** Milliseconds since the epoch; 0 when the timestamp is unreadable. */
    savedAt: number;
    submitted: boolean;
    /** False for a blob `decodeDraft` would reject — it can never be loaded. */
    usable: boolean;
};

export type StoreResult = WriteResult & { evicted: string[] };

export type PruneOptions = {
    /** Never pruned, however old: the asset being opened right now. */
    keep?: string;
    maxAgeMs?: number;
    maxSubmitted?: number;
    now?: number;
};

function vxidFromKey(key: string): string | null {
    if (!key.startsWith(KEY_PREFIX)) return null;
    const vxid = key.slice(KEY_PREFIX.length);
    return vxid === "" ? null : vxid;
}

function draftVxids(storage: DraftStorage): string[] {
    const { key, length } = storage;
    if (typeof key !== "function" || typeof length !== "number") return [];

    const vxids: string[] = [];
    for (let i = 0; i < length; i++) {
        const vxid = vxidFromKey(key.call(storage, i) ?? "");
        if (vxid) vxids.push(vxid);
    }
    return vxids;
}

function summarize(storage: DraftStorage, vxid: string): DraftSummary | null {
    const raw = storage.getItem(draftKey(vxid));
    if (raw === null) return null;

    const unusable: DraftSummary = {
        vxid,
        savedAt: 0,
        submitted: false,
        usable: false,
    };

    let parsed: unknown;
    try {
        parsed = JSON.parse(raw);
    } catch {
        return unusable;
    }
    if (!parsed || typeof parsed !== "object") return unusable;

    // Matches what `decodeDraft` is willing to load, so nothing is counted as
    // usable that the editor would then refuse.
    const { segments, savedAt, submittedAt } = parsed as Record<
        string,
        unknown
    >;
    if (!Array.isArray(segments) || segments.length === 0) return unusable;

    const saved = typeof savedAt === "string" ? Date.parse(savedAt) : NaN;

    return {
        vxid,
        savedAt: Number.isNaN(saved) ? 0 : saved,
        submitted: typeof submittedAt === "string",
        usable: true,
    };
}

/**
 * Every draft in storage, newest first.
 *
 * Empty when the storage cannot be enumerated, so callers fall back to leaving
 * everything in place.
 */
export function listDrafts(storage: DraftStorage): DraftSummary[] {
    try {
        return draftVxids(storage)
            .map((vxid) => summarize(storage, vxid))
            .filter((d): d is DraftSummary => d !== null)
            .sort((a, b) => b.savedAt - a.savedAt);
    } catch {
        return [];
    }
}

/**
 * The drafts that may be thrown away to make room, most expendable first.
 *
 * Only submitted work is ever a candidate. A draft that has not been handed in
 * is work the user still has to do something with, and deleting it to save
 * another transcription just moves the loss somewhere they are not looking.
 * Unreadable blobs go first — nothing can load them, so they are pure
 * overhead — then the longest-submitted.
 */
export function evictionOrder(drafts: DraftSummary[]): DraftSummary[] {
    return drafts
        .filter((d) => !d.usable || d.submitted)
        .sort(
            (a, b) =>
                Number(a.usable) - Number(b.usable) || a.savedAt - b.savedAt,
        );
}

/**
 * Drops what is no longer worth the room: blobs that cannot be loaded, and
 * submitted drafts that are either stale or past the cap. Returns the assets
 * that were removed.
 *
 * This is the bound on growth. Without it a user accumulates one blob per asset
 * they ever open, and a handful of hour-long transcriptions is enough to fill
 * the origin's quota and leave the *next* one with nowhere to save.
 */
export function pruneDrafts(
    storage: DraftStorage,
    options: PruneOptions = {},
): string[] {
    const {
        keep,
        maxAgeMs = DRAFT_MAX_AGE_MS,
        maxSubmitted = DRAFT_MAX_SUBMITTED,
        now = Date.now(),
    } = options;

    const candidates = evictionOrder(listDrafts(storage)).filter(
        (d) => d.vxid !== keep,
    );

    // Candidates are oldest-first, so the ones over the cap are the front of
    // the list.
    const submitted = candidates.filter((d) => d.usable);
    const overCap = new Set(
        submitted.slice(0, Math.max(0, submitted.length - maxSubmitted)),
    );

    const doomed = candidates.filter(
        (d) => !d.usable || overCap.has(d) || now - d.savedAt > maxAgeMs,
    );

    for (const d of doomed) removeDraft(storage, d.vxid);
    return doomed.map((d) => d.vxid);
}

/**
 * Writes the draft, making room first if the quota is full.
 *
 * A full quota is rarely about the transcription being saved — it is the ones
 * already submitted and finished with still holding the space. Those are
 * evicted one at a time, retrying after each, so no more is thrown away than
 * the write actually needs. If evicting every expendable draft is still not
 * enough the quota failure is reported as before, and the editor warns the
 * user.
 */
export function storeDraft(
    storage: DraftStorage,
    vxid: string,
    draft: TranscriptionDraft,
): StoreResult {
    let result = writeDraft(storage, vxid, draft);
    if (result.ok || result.reason !== "quota")
        return { ...result, evicted: [] };

    const evicted: string[] = [];
    const victims = evictionOrder(listDrafts(storage)).filter(
        (d) => d.vxid !== vxid,
    );

    for (const victim of victims) {
        removeDraft(storage, victim.vxid);
        evicted.push(victim.vxid);

        result = writeDraft(storage, vxid, draft);
        if (result.ok || result.reason !== "quota") break;
    }

    return { ...result, evicted };
}
