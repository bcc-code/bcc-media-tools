import type { BMMTrack, LanguageList } from "~/src/gen/api/v1/api_pb";

export type Track = {
    title: string;
    id: number;
    type: "track";
    published_at: string;
    languages: LanguageList
};

export type TrackSubType = "audiobook";

export type BMMSingleForm = {
    title: string;
    albumId?: string;
    language?: string;
    environment?: string;
    track?: BMMTrack;
};

export type FileAndLanguage = {
    file: File;
    language: string;
};
