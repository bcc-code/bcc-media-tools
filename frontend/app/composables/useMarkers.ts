import { useLocalStorage } from "@vueuse/core";
import type { Marker, MarkerType } from "~/utils/markers";

// Frontend-only data layer for the markers tool.
//
// Working state is auto-persisted to localStorage (keyed by VX-id), mirroring
// the transcription editor's local-autosave behaviour. `save()` currently just
// simulates the round-trip to the backend.
//
// TODO(backend): replace the seed/load/save internals with the real RPCs, e.g.
//   load  -> api.getMarkers({ VXID })
//   save  -> api.submitMarkers({ VXID, markers })
// The public surface of this composable should not need to change.

// Demo "imported" markers, standing in for the third-party timing feed. Times
// are kept within the first few minutes so they line up with most previews.
function seedMarkers(): Marker[] {
    return [
        {
            id: generateRandomId(),
            type: "name-super",
            label: "Kåre Johan Hamre",
            note: "Speaker",
            start: 12.5,
            end: 18.0,
            source: "imported",
        },
        {
            id: generateRandomId(),
            type: "bible-verse",
            label: "John 3:16",
            start: 45.0,
            end: 52.5,
            source: "imported",
        },
        {
            id: generateRandomId(),
            type: "name-super",
            label: "Bente Lothe",
            start: 95.2,
            end: 101.0,
            source: "imported",
        },
        {
            id: generateRandomId(),
            type: "song",
            label: "How Great Thou Art",
            start: 130.0,
            end: 240.0,
            source: "imported",
        },
    ];
}

export function useMarkers(vxId: MaybeRefOrGetter<string>) {
    // Plain string key, evaluated once. Each asset is opened in a fresh mount
    // (via the index page), so we don't need a reactive key here.
    const storageKey = `markers-${toValue(vxId)}`;

    // `useLocalStorage` seeds with the imported demo data on first visit, then
    // reads the working copy on subsequent loads. Its internal deep watcher
    // persists every mutation automatically.
    const markers = useLocalStorage<Marker[]>(storageKey, seedMarkers());

    // True when there are local changes not yet submitted (via `save`). Starts
    // clean; every mutation dirties it. With a real backend this would compare
    // against the last submitted state.
    const dirty = ref(false);

    function add(
        partial: Partial<Marker> & Pick<Marker, "start" | "end">,
    ): Marker {
        const marker: Marker = {
            id: generateRandomId(),
            type: "name-super",
            label: "",
            source: "manual",
            ...partial,
        };
        markers.value = [...markers.value, marker];
        dirty.value = true;
        return marker;
    }

    function update(id: string, patch: Partial<Omit<Marker, "id">>) {
        markers.value = markers.value.map((m) =>
            m.id === id ? { ...m, ...patch } : m,
        );
        dirty.value = true;
    }

    function remove(id: string) {
        markers.value = markers.value.filter((m) => m.id !== id);
        dirty.value = true;
    }

    // Restore a previously removed marker (used by the undo toast).
    function restore(marker: Marker) {
        if (markers.value.some((m) => m.id === marker.id)) return;
        markers.value = [...markers.value, marker];
        dirty.value = true;
    }

    const saving = ref(false);
    // Simulated submit-to-backend. Swap for `api.submitMarkers` later.
    async function save() {
        saving.value = true;
        try {
            await new Promise((resolve) => setTimeout(resolve, 400));
            dirty.value = false;
        } finally {
            saving.value = false;
        }
    }

    function countByType(type: MarkerType) {
        return markers.value.filter((m) => m.type === type).length;
    }

    return {
        markers,
        dirty,
        add,
        update,
        remove,
        restore,
        save,
        saving,
        countByType,
    };
}
