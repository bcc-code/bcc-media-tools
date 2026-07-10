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
    const perms = usePermissions();
    const api = useAPI();
    const toaster = useToast();

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
        return [
            {
                name: "Export",
                action: "Go to VX export",
                color: "#9aa0a8",
                enabled: perms.canExport.value,
                run: () => openTool("/export/"),
            },
            {
                name: "Export Oslofjord",
                action: "Go to VB export",
                color: "#3c61d8",
                enabled: perms.canVbExport.value,
                run: () => openTool("/vb-export/"),
            },
            {
                name: "Make preview",
                action: "Trigger preview generation",
                color: "#cdbf3a",
                enabled: perms.canCantemoPreview.value,
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
                enabled: perms.canCantemoTranscribe.value,
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
                enabled:
                    perms.canTranscribe.value ||
                    perms.isTranscriptionAdmin.value,
                run: () => openToolWithIdPath("/transcription"),
            },
            {
                name: "Update subtitle from Subtrans",
                action: "Trigger appropriate workflow",
                color: "#3fb84f",
                enabled: perms.canCantemoSubtitles.value,
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
                enabled: perms.canCantemoRelations.value,
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
