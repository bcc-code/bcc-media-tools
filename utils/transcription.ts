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
