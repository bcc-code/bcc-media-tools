type StringWithAutocomplete<T> = T | (string & {});

export type Page = StringWithAutocomplete<
    "upload_index" | "upload_continue" | "upload_success" | "transcription"
>;

type ElementType = "UPLOAD";

export type Events = {
    language_changed: {
        pageCode: string;
        languageFrom: string;
        languageTo: string;
    };
    transcription_loaded: {
        trackId: string;
        language: string;
    };
    upload_started: {
        trackId: string;
        language: string;
        forceOverride: boolean;
    };
    upload_finished: {
        trackId: string;
        language: string;
        success: boolean;
        error?: string;
        size?: number;
        duration: number | undefined;
    };
};
