export type Track = {
    title: string;
    id: number;
    type: "track";
    published_at: string;
};

export type TrackSubType = "audiobook";

export type BMMSingleForm = {
    title: string;
    albumId?: string;
    trackId?: string;
    language?: (typeof bmmLanguages)[number];
};
