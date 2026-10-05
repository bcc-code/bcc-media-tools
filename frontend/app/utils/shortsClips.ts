export type Clip = {
    id: string;
    start: number;
    end: number;
};

/** How long a newly added clip is, before the user trims it. */
export const DEFAULT_CLIP_SECONDS = 30;

/**
 * Where a newly added clip should sit: starting at the playhead, so it lands
 * where the user is looking, and pulled back from the end if there is not a
 * whole clip's worth of video left.
 *
 * Returns undefined when the media has no duration yet.
 */
export function nextClipRange(
    playhead: number,
    duration: number,
    length = DEFAULT_CLIP_SECONDS,
): { start: number; end: number } | undefined {
    if (!(duration > 0)) return undefined;

    const span = Math.min(length, duration);
    const start = Math.min(Math.max(0, playhead), duration - span);
    return { start, end: start + span };
}

/** A clip is valid if it has a positive length; the backend refuses the rest. */
export function isValidClip(clip: Clip): boolean {
    return clip.end > clip.start;
}

export function clipLength(clip: Clip): number {
    return Math.max(0, clip.end - clip.start);
}

/**
 * Ids of clips that overlap at least one other clip.
 *
 * Overlapping is allowed — cutting two variants of the same moment is a real
 * thing to want — so this only drives a hint, never a block.
 */
export function overlappingClipIds(clips: Clip[]): Set<string> {
    const overlapping = new Set<string>();

    for (let i = 0; i < clips.length; i++) {
        for (let j = i + 1; j < clips.length; j++) {
            const a = clips[i]!;
            const b = clips[j]!;
            if (a.end > b.start && a.start < b.end) {
                overlapping.add(a.id);
                overlapping.add(b.id);
            }
        }
    }

    return overlapping;
}

/** Clips in timeline order, which is how the list and the submit are read. */
export function sortClips(clips: Clip[]): Clip[] {
    return [...clips].sort((a, b) => a.start - b.start || a.end - b.end);
}

/**
 * Drops anything that is not a usable clip.
 *
 * Clips are restored from localStorage, so they can be any shape an older
 * version of the page wrote, or that someone edited by hand. A bad entry must
 * not take the editor down with it.
 */
export function sanitiseClips(value: unknown, duration?: number): Clip[] {
    if (!Array.isArray(value)) return [];

    const seen = new Set<string>();
    const clips: Clip[] = [];

    for (const entry of value) {
        if (typeof entry !== "object" || entry === null) continue;
        const { id, start, end } = entry as Partial<Clip>;

        if (typeof id !== "string" || id === "" || seen.has(id)) continue;
        if (typeof start !== "number" || !Number.isFinite(start)) continue;
        if (typeof end !== "number" || !Number.isFinite(end)) continue;

        const clamped = {
            id,
            start: Math.max(0, start),
            end: duration != undefined ? Math.min(end, duration) : end,
        };
        if (!isValidClip(clamped)) continue;

        seen.add(id);
        clips.push(clamped);
    }

    return clips;
}

const pad = (n: number) => String(n).padStart(2, "0");

/**
 * A position or a length, as `m:ss` or `h:mm:ss`.
 *
 * `total` is the length of the asset the value belongs to, and decides the shape
 * for *every* value from that asset: reading `80:15` in one place and
 * `01:20:19` in another for the same moment is worse than either on its own.
 * Omit it for a duration, which stays `m:ss` until it passes an hour itself.
 */
export function formatClock(seconds: number, total = 0): string {
    const value = Math.max(0, Math.round(seconds));
    const minutes = Math.floor(value / 60);
    const secs = value % 60;

    if (Math.max(total, value) < 3600) return `${minutes}:${pad(secs)}`;
    return `${Math.floor(minutes / 60)}:${pad(minutes % 60)}:${pad(secs)}`;
}
