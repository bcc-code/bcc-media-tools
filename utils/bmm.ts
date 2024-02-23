export type BMMSingleForm = {
    originalTitle: string;
    albumId?: string;
    trackId?: string;
    language?: (typeof bmmLanguages)[number];
};
