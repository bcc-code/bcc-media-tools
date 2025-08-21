export type Segment = {
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

    for (const s of segments) {
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
    const data = JSON.stringify({
        text: segments.map((s) => s.text).join(" "),
        segments: segments,
    });
    filename = filename.split(".").slice(0, -1).join(".");
    return downloadStringContent(
        data,
        filename + "-edited.json",
        "application/json",
    );
}
