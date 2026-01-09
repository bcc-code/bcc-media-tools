import type { BMMTrack } from "~~/src/gen/api/v1/api_pb";

export type BMMSingleForm = {
    title: string;
    albumId?: string;
    language?: string;
    environment?: string;
    track?: BMMTrack;
    contentType?: "podcast" | "album";
};

export type FileAndLanguage = {
    file: File;
    language: string;
};
