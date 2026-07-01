import {
    MarkerSource as PbMarkerSource,
    MarkerType as PbMarkerType,
    type Marker as PbMarker,
} from "~~/src/gen/api/v1/api_pb";
import type { Marker, MarkerSource, MarkerType } from "~/utils/markers";

// Markers data layer, backed by the ConnectRPC API. Working state is held
// locally; `save` submits it. See `backend/cmd/server/markers.go` — the backend
// is still an in-memory placeholder pending the third-party timing integration.

const TYPE_TO_PB: Record<MarkerType, PbMarkerType> = {
    "name-super": PbMarkerType.NAME_SUPER,
    "bible-verse": PbMarkerType.BIBLE_VERSE,
    song: PbMarkerType.SONG,
    chapter: PbMarkerType.CHAPTER,
    custom: PbMarkerType.CUSTOM,
};
const TYPE_FROM_PB: Record<number, MarkerType> = {
    [PbMarkerType.NAME_SUPER]: "name-super",
    [PbMarkerType.BIBLE_VERSE]: "bible-verse",
    [PbMarkerType.SONG]: "song",
    [PbMarkerType.CHAPTER]: "chapter",
    [PbMarkerType.CUSTOM]: "custom",
};
const SOURCE_TO_PB: Record<MarkerSource, PbMarkerSource> = {
    imported: PbMarkerSource.IMPORTED,
    manual: PbMarkerSource.MANUAL,
};
const SOURCE_FROM_PB: Record<number, MarkerSource> = {
    [PbMarkerSource.IMPORTED]: "imported",
    [PbMarkerSource.MANUAL]: "manual",
};

function fromPb(m: PbMarker): Marker {
    return {
        id: m.id,
        type: TYPE_FROM_PB[m.type] ?? "custom",
        label: m.label,
        note: m.note || undefined,
        start: m.start,
        end: m.end,
        source: SOURCE_FROM_PB[m.source] ?? "manual",
    };
}

function toPb(m: Marker) {
    return {
        id: m.id,
        type: TYPE_TO_PB[m.type],
        label: m.label,
        note: m.note ?? "",
        start: m.start,
        end: m.end,
        source: SOURCE_TO_PB[m.source],
    };
}

export function useMarkers(vxId: MaybeRefOrGetter<string>) {
    const api = useAPI();

    const markers = ref<Marker[]>([]);
    const loading = ref(true);
    // True when there are local changes not yet submitted (via `save`).
    const dirty = ref(false);

    async function load() {
        loading.value = true;
        try {
            const res = await api.getMarkers({ VXID: toValue(vxId) });
            markers.value = res.markers.map(fromPb);
            dirty.value = false;
        } catch (err) {
            console.error("Failed to load markers", err);
        } finally {
            loading.value = false;
        }
    }
    onMounted(load);

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

    function restore(marker: Marker) {
        if (markers.value.some((m) => m.id === marker.id)) return;
        markers.value = [...markers.value, marker];
        dirty.value = true;
    }

    // Replace the whole working set (used by import).
    function replaceAll(next: Marker[]) {
        markers.value = next;
        dirty.value = true;
    }

    const saving = ref(false);
    async function save() {
        saving.value = true;
        try {
            await api.submitMarkers({
                VXID: toValue(vxId),
                markers: markers.value.map(toPb),
            });
            dirty.value = false;
        } finally {
            saving.value = false;
        }
    }

    return {
        markers,
        loading,
        dirty,
        add,
        update,
        remove,
        restore,
        replaceAll,
        save,
        saving,
    };
}
