export interface ExportSelection {
    destinations: string[];
    audioSource: string;
    languages: string[];
    resolutions: { width: number; height: number; downloadable: boolean }[];
    overlay: string;
    withChapters: boolean;
    ignoreSilence: boolean;
    exportAiSubs: boolean;
    subclips: string[];
}
