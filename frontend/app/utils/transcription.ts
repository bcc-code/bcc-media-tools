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

export function setWordText(
    segment: Segment,
    index: number,
    text: string,
): Segment {
    const words = segment.words.map((w, i) =>
        i === index ? { ...w, text } : w,
    );
    return { ...segment, words, text: segmentText(words) };
}

// Requires a pause between the rows, which is why the "+" only shows up
// between some of them (B5 in docs/transcription-tool-improvements.md).
export function canInsertAfter(
    segments: Segment[],
    index: number,
    minGap = 1,
): boolean {
    const curr = segments[index];
    const next = segments[index + 1];
    if (!curr || !next) return false;
    return next.start >= curr.end + minGap;
}

export function createSegmentBetween(prev: Segment, next: Segment): Segment {
    return {
        uid: generateRandomId(),
        id: (prev.id + next.id) / 2,
        seek: 0,
        start: prev.end,
        end: next.start,
        text: "",
        tokens: [],
        temperature: 0,
        avg_logprob: 0,
        compression_ration: 0,
        no_speech_prob: 0,
        confidence: 0,
        words: [
            {
                text: "",
                start: prev.end,
                end: next.start,
                confidence: 0,
            },
        ],
    };
}

export function insertSegmentAfter(
    segments: Segment[],
    index: number,
): Segment[] {
    const prev = segments[index];
    const next = segments[index + 1];
    if (!prev || !next) return segments;

    const arr = [...segments];
    arr.splice(index + 1, 0, createSegmentBetween(prev, next));
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
