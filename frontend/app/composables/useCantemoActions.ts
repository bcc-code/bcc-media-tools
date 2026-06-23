import { CantemoAction } from "~~/src/gen/api/v1/api_pb";

export type CantemoChip = {
    name: string;
    action: string;
    color: string;
    enabled: boolean;
    run: () => void | Promise<void>;
};

// Builds the permission-gated Cantemo action chips for a given Vidispine item id.
// Shared by the embedded cantemo.vue panel and the VAULT item detail page.
export function useCantemoActions(vxId: MaybeRefOrGetter<string | undefined>) {
    const { me } = useMe();
    const api = useAPI();
    const toaster = useDesignToaster();

    // Name of the chip whose workflow is currently being triggered (disables it).
    const loading = ref<string | null>(null);

    async function trigger(
        name: string,
        action: CantemoAction,
        started: string,
    ) {
        const id = toValue(vxId);
        if (!id || loading.value) return;
        loading.value = name;
        try {
            await api.triggerCantemoAction({ VXID: id, action });
            toaster.create({
                title: started,
                type: "success",
            });
        } catch (err) {
            toaster.create({
                title: "Failed to start",
                description: (err as Error)?.message,
                type: "error",
            });
        } finally {
            loading.value = null;
        }
    }

    // The cantemo panel is embedded as a cross-origin iframe, so navigation chips
    // open the export tools in a new tab.
    function openTool(path: string) {
        const id = toValue(vxId);
        if (!id) return;
        window.open(`${path}?id=${id}`, "_blank");
    }

    // Like openTool, but for tools that take the item id as a path segment
    // (e.g. the transcription editor at /transcription/<vxid>) rather than ?id=.
    function openToolWithIdPath(path: string) {
        const id = toValue(vxId);
        if (!id) return;
        window.open(`${path}/${id}`, "_blank");
    }

    const chips = computed<CantemoChip[]>(() => {
        const m = me.value;
        return [
            {
                name: "Export",
                action: "Go to VX export",
                color: "#9aa0a8",
                enabled: !!(
                    m?.admin ||
                    (m?.export &&
                        (m.export.destinations.length > 0 ||
                            m.export.admin ||
                            m.export.timedMetadata))
                ),
                run: () => openTool("/export/"),
            },
            {
                name: "Export Oslofjord",
                action: "Go to VB export",
                color: "#3c61d8",
                enabled: !!(
                    m?.admin ||
                    (m?.vbExport &&
                        (m.vbExport.destinations.length > 0 ||
                            m.vbExport.admin))
                ),
                run: () => openTool("/vb-export/"),
            },
            {
                name: "Make preview",
                action: "Trigger preview generation",
                color: "#cdbf3a",
                enabled: !!(m?.admin || m?.cantemo?.preview),
                run: () =>
                    trigger(
                        "Make preview",
                        CantemoAction.PREVIEW,
                        "Preview generation started",
                    ),
            },
            {
                name: "Transcribe",
                action: "Trigger transcription",
                color: "#3fb84f",
                enabled: !!(m?.admin || m?.cantemo?.transcribe),
                run: () =>
                    trigger(
                        "Transcribe",
                        CantemoAction.TRANSCRIBE,
                        "Transcription started",
                    ),
            },
            {
                name: "Correct transcription",
                action: "Open the transcription editor",
                color: "#8b5cf6",
                enabled: !!(
                    m?.admin ||
                    (m?.transcription &&
                        (m.transcription.admin || m.transcription.mediabanken))
                ),
                run: () => openToolWithIdPath("/transcription"),
            },
            {
                name: "Update subtitle from Subtrans",
                action: "Trigger appropriate workflow",
                color: "#3fb84f",
                enabled: !!(m?.admin || m?.cantemo?.subtitles),
                run: () =>
                    trigger(
                        "Update subtitle from Subtrans",
                        CantemoAction.SUBTITLE_FROM_SUBTRANS,
                        "Subtitle update started",
                    ),
            },
            {
                name: "Update asset relations",
                action: "Update asset relations flow",
                color: "#3c61d8",
                enabled: !!(m?.admin || m?.cantemo?.relations),
                run: () =>
                    trigger(
                        "Update asset relations",
                        CantemoAction.UPDATE_RELATIONS,
                        "Asset relations update started",
                    ),
            },
        ].filter((c) => c.enabled);
    });

    return { chips, loading };
}
