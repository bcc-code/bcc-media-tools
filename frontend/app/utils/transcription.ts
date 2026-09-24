import { generateRandomId } from "./id";

export type Segment = {
    // Client-side only, stripped before submitting. Rows are addressed by uid
    // because array indexes shift and edits landed on the wrong segment.
    uid: string;
    deleted?: boolean;
    id: number;
    seek: number;
    start: number;
    end: number;
    text: string;
    tokens: number[];
    temperature: number;
    avg_logprob: number;
    compression_ration: number;
    no_speech_prob: number;
    confidence: number;
    words: Word[];
};

/** A segment as it arrives from the API or a whisper JSON file. */
export type RawSegment = Omit<Segment, "uid" | "deleted">;

export function withUids(
    segments: (RawSegment & { uid?: string })[],
): Segment[] {
    return segments.map((s) => ({
        ...s,
        uid: s.uid ?? generateRandomId(),
    }));
}

export function segmentText(words: Word[]): string {
    return words
        .map((w) => w.text.trim())
        .filter((t) => t !== "")
        .join(" ");
}

/** The single place that decides what leaves the editor. */
export function toTranscription(segments: Segment[]): TranscriptionResult {
    const kept = segments
        .filter((s) => !s.deleted)
        .map(({ uid: _uid, deleted: _deleted, ...rest }) => ({
            ...rest,
            text: segmentText(rest.words),
        }))
        // An emptied row is how you remove speech the ASR invented, and an
        // inserted row may never be filled in. Neither should become a cue.
        .filter((s) => s.text !== "");

    return {
        text: kept.map((s) => s.text).join(" "),
        segments: kept as Segment[],
    };
}

export function updateSegment(
    segments: Segment[],
    uid: string,
    update: (segment: Segment) => Segment,
): Segment[] {
    return segments.map((s) => (s.uid === uid ? update(s) : s));
}

export function toggleSegmentDeleted(
    segments: Segment[],
    uid: string,
): Segment[] {
    return updateSegment(segments, uid, (s) => ({ ...s, deleted: !s.deleted }));
}

export function tokenizeWords(text: string): string[] {
    return text.split(/\s+/).filter((t) => t !== "");
}

// Matching ignores case and punctuation so that fixing "hei," to "Hei." still
// counts as the same word and keeps its original timing.
function matchKey(token: string): string {
    return token.toLowerCase().replace(/[^\p{L}\p{N}]/gu, "") || token;
}

/** Index pairs of the longest common subsequence of two token lists. */
function commonTokens(a: string[], b: string[]): [number, number][] {
    const n = a.length;
    const m = b.length;
    const lengths: number[][] = Array.from({ length: n + 1 }, () =>
        new Array<number>(m + 1).fill(0),
    );

    for (let i = n - 1; i >= 0; i--) {
        for (let j = m - 1; j >= 0; j--) {
            lengths[i]![j] =
                a[i] === b[j]
                    ? lengths[i + 1]![j + 1]! + 1
                    : Math.max(lengths[i + 1]![j]!, lengths[i]![j + 1]!);
        }
    }

    const pairs: [number, number][] = [];
    let i = 0;
    let j = 0;
    while (i < n && j < m) {
        if (a[i] === b[j]) {
            pairs.push([i, j]);
            i++;
            j++;
        } else if (lengths[i + 1]![j]! >= lengths[i]![j + 1]!) {
            i++;
        } else {
            j++;
        }
    }
    return pairs;
}

function spreadWords(tokens: string[], from: number, to: number): Word[] {
    const step = Math.max(0, to - from) / tokens.length;
    return tokens.map((text, i) => ({
        text,
        start: from + i * step,
        end: from + (i + 1) * step,
        confidence: 0,
    }));
}

/**
 * Re-derives word timings after the segment's text was edited.
 *
 * This is a correction pass, so almost every word survives: unchanged words
 * keep their original start/end exactly, and only inserted runs need timings,
 * spread across the span between the words that did survive. A replaced word
 * therefore inherits the span of the one it replaced.
 */
export function realignWords(
    previous: Word[],
    text: string,
    bounds: { start: number; end: number },
): Word[] {
    const tokens = tokenizeWords(text);
    if (tokens.length === 0) return [];
    if (previous.length === 0) {
        return spreadWords(tokens, bounds.start, bounds.end);
    }

    const kept = commonTokens(
        previous.map((w) => matchKey(w.text)),
        tokens.map(matchKey),
    );

    const words = new Array<Word | undefined>(tokens.length);
    // Which previous word each surviving token came from, so a run of new
    // tokens can be given the time of the words it replaced.
    const source = new Array<number | undefined>(tokens.length);
    for (const [prevIndex, tokenIndex] of kept) {
        words[tokenIndex] = {
            ...previous[prevIndex]!,
            text: tokens[tokenIndex]!,
        };
        source[tokenIndex] = prevIndex;
    }

    for (let i = 0; i < words.length;) {
        if (words[i]) {
            i++;
            continue;
        }
        let end = i;
        while (end < words.length && !words[end]) end++;

        const leftPrev = i > 0 ? source[i - 1]! : -1;
        const rightPrev = end < words.length ? source[end]! : previous.length;
        const replaced = previous.slice(leftPrev + 1, rightPrev);

        // Replacing words takes over exactly the time they occupied; inserting
        // between two survivors uses the pause between them.
        const from = replaced.length
            ? replaced[0]!.start
            : leftPrev >= 0
              ? previous[leftPrev]!.end
              : bounds.start;
        const to = replaced.length
            ? replaced.at(-1)!.end
            : rightPrev < previous.length
              ? previous[rightPrev]!.start
              : bounds.end;

        const filled = spreadWords(tokens.slice(i, end), from, to);
        for (let k = 0; k < filled.length; k++) words[i + k] = filled[k];

        i = end;
    }

    return words as Word[];
}

/** Replaces a segment's whole text, keeping the timings of surviving words. */
export function setSegmentText(segment: Segment, text: string): Segment {
    const words = realignWords(segment.words, text, segment);
    return { ...segment, words, text: segmentText(words) };
}

/** The word containing a character offset into the segment's text. */
export function wordAtOffset(words: Word[], offset: number): Word | undefined {
    let position = 0;
    for (const word of words) {
        position += word.text.length;
        if (offset <= position) return word;
        position += 1; // the joining space
    }
    return words.at(-1);
}

/** Default length of an inserted segment where there is no gap to fill. */
const INSERTED_SECONDS = 2;
/** A segment we carve time out of keeps at least this much. */
const MIN_SEGMENT_SECONDS = 0.4;

function emptySegment(start: number, end: number): Segment {
    return {
        uid: generateRandomId(),
        id: 0,
        seek: 0,
        start,
        end,
        text: "",
        tokens: [],
        temperature: 0,
        avg_logprob: 0,
        compression_ration: 0,
        no_speech_prob: 0,
        confidence: 0,
        words: [],
    };
}

/**
 * Inserts an empty segment after `index`; -1 puts it before the first row.
 *
 * Speech the ASR dropped is not always followed by a pause, so an insert must
 * be possible anywhere. Where there is a gap the new row fills it; where there
 * is none it borrows time from the following row, which is where the missing
 * speech actually was.
 */
export function insertSegmentAt(
    segments: Segment[],
    index: number,
    mediaDuration?: number,
): Segment[] {
    const prev = index >= 0 ? segments[index] : undefined;
    const next = segments[index + 1];
    const arr = [...segments];

    if (!prev && !next) {
        return [emptySegment(0, INSERTED_SECONDS)];
    }

    if (!next) {
        const start = prev!.end;
        const end =
            mediaDuration !== undefined
                ? Math.max(
                      start,
                      Math.min(start + INSERTED_SECONDS, mediaDuration),
                  )
                : start + INSERTED_SECONDS;
        arr.splice(index + 1, 0, emptySegment(start, end));
        return arr;
    }

    const from = prev ? prev.end : 0;
    const gap = next.start - from;

    if (gap > 0) {
        arr.splice(index + 1, 0, emptySegment(from, next.start));
        return arr;
    }

    const carved = Math.min(
        INSERTED_SECONDS,
        Math.max(0, next.end - next.start - MIN_SEGMENT_SECONDS),
    );
    arr.splice(index + 1, 0, emptySegment(from, from + carved));
    arr[index + 2] = { ...next, start: from + carved };
    return arr;
}

export type Word = {
    text: string;
    start: number;
    end: number;
    confidence: number;
};

export type TranscriptionResult = {
    text: string;
    segments: Segment[];
};

function downloadStringContent(
    content: string,
    filename: string,
    type: string,
) {
    const blob = new Blob([content], { type });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.download = filename;
    a.href = url;
    a.click();
    URL.revokeObjectURL(url);
}

export function formatTime(seconds: number) {
    // Calculate hours, minutes, seconds, and milliseconds
    const hours = Math.floor(seconds / 3600);
    seconds = seconds % 3600; // Remaining seconds
    const minutes = Math.floor(seconds / 60);
    seconds = seconds % 60; // Remaining seconds
    const milliseconds = Math.round((seconds % 1) * 1000); // Get milliseconds
    seconds = Math.floor(seconds); // Get whole seconds

    // Pad with zeros to ensure HH:MM:SS.ttt format
    const formattedHours = hours.toString().padStart(2, "0");
    const formattedMinutes = minutes.toString().padStart(2, "0");
    const formattedSeconds = seconds.toString().padStart(2, "0");
    const formattedMilliseconds = milliseconds.toString().padStart(3, "0");

    return `${formattedHours}:${formattedMinutes}:${formattedSeconds}.${formattedMilliseconds}`;
}

export function secondsFromFormattedTime(time: string) {
    const [timeparts, milliseconds] = time.split(".");
    const [hours, minutes, seconds] = timeparts!.split(":");
    return (
        parseInt(hours!) * 3600 +
        parseInt(minutes!) * 60 +
        parseInt(seconds!) +
        parseInt(milliseconds!) / 1000
    );
}

export function downloadTranscriptionSRT(
    segments: Segment[],
    filename: string,
    wordlevel = false,
) {
    let srt = "";

    let i = 0;

    for (const s of segments.filter((s) => !s.deleted)) {
        if (wordlevel) {
            for (const w of s.words) {
                i++;
                srt += `${i}\n`;
                srt += `${formatTime(w.start)} --> ${formatTime(w.end)}\n`;
                srt += `${w.text}\n\n`;
            }
        } else {
            i++;
            srt += `${i}\n`;
            srt += `${formatTime(s.start)} --> ${formatTime(s.end)}\n`;
            srt += `${s.text}\n\n`;
        }
    }

    return downloadStringContent(srt, filename + ".srt", "text/plain");
}

export function downloadTranscriptionJSON(
    segments: Segment[],
    filename: string,
) {
    const data = JSON.stringify(toTranscription(segments));
    filename = filename.split(".").slice(0, -1).join(".") || filename;
    return downloadStringContent(
        data,
        filename + "-edited.json",
        "application/json",
    );
}
