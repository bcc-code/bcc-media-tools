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

export function downloadTranscription(segments: Segment[], filename: string) {
    const data = JSON.stringify({
        text: segments.map((s) => s.text).join(" "),
        segments: segments,
    });
    const blob = new Blob([data], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    filename = filename.split(".").slice(0, -1).join(".");
    a.download = filename + "-edited.json";
    a.click();
    URL.revokeObjectURL(url);
}
