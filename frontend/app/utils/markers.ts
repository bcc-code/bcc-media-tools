// Markers tool — data model.
//
// NOTE: this tool is currently frontend-only. Markers are persisted to
// localStorage by `useMarkers`. The shapes below are intentionally close to
// what a future ConnectRPC `GetMarkers` / `SubmitMarkers` pair would return so
// the mock data layer can be swapped for `api.*` calls with minimal churn.

export type MarkerType =
    | "name-super"
    | "bible-verse"
    | "song"
    | "chapter"
    | "custom";

// Where a marker came from. "imported" markers originate from the third-party
// timing program (name-supers, bible-verse references, …); "manual" ones are
// created here. The flag is preserved through edits so the future backend can
// reconcile imported markers with their source.
export type MarkerSource = "imported" | "manual";

export type Marker = {
    id: string;
    type: MarkerType;
    // Display text: the name, the verse reference, the song/chapter title.
    label: string;
    note?: string;
    // In/out points in seconds (float, ms precision — matches transcription Word).
    start: number;
    end: number;
    source: MarkerSource;
};

export type MarkerTypeMeta = {
    value: MarkerType;
    icon: string;
    // Tailwind background class used on timeline blocks and list dots.
    color: string;
};

// Single source of truth for the available marker types. Labels are resolved
// via i18n (`markers.types.<value>`); see `markerTypeLabel`.
export const MARKER_TYPES: MarkerTypeMeta[] = [
    { value: "name-super", icon: "tabler:user", color: "bg-blue-500" },
    { value: "bible-verse", icon: "tabler:book-2", color: "bg-purple-500" },
    { value: "song", icon: "tabler:music", color: "bg-emerald-500" },
    { value: "chapter", icon: "tabler:bookmark", color: "bg-amber-500" },
    { value: "custom", icon: "tabler:tag", color: "bg-slate-500" },
];

export function markerTypeMeta(type: MarkerType): MarkerTypeMeta {
    return (
        MARKER_TYPES.find((m) => m.value === type) ??
        MARKER_TYPES[MARKER_TYPES.length - 1]!
    );
}

// Sort by start time, then end time — stable order for the list and timeline.
export function sortMarkers(markers: Marker[]): Marker[] {
    return [...markers].sort((a, b) => a.start - b.start || a.end - b.end);
}
