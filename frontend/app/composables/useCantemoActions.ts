import { CantemoAction } from "~~/src/gen/api/v1/api_pb";

export type CantemoChip = {
    id: string;
    label: string;
    description: string;
    /**
     * "open" navigates to a tool, "run" starts a background workflow and
     * reports nothing back. The two are indistinguishable otherwise, which is
     * the main thing that made this panel hard to read.
     */
    kind: "open" | "run";
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
    const { t } = useI18n();

    // Id of the chip whose workflow is currently being triggered (disables it).
    const loading = ref<string | null>(null);

    async function trigger(id: string, action: CantemoAction) {
        const vx = toValue(vxId);
        if (!vx || loading.value) return;
        loading.value = id;
        try {
            await api.triggerCantemoAction({ VXID: vx, action });
            toaster.create({
                title: t("cantemo.started", {
                    action: t(`cantemo.${id}.label`),
                }),
                description: t("cantemo.startedDescription"),
                type: "success",
            });
        } catch (err) {
            toaster.create({
                title: t("cantemo.startFailed"),
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

    const chips = computed<CantemoChip[]>(() =>
        (
            [
                {
                    id: "export",
                    kind: "open",
                    color: "#9aa0a8",
                    enabled: perms.canExport.value,
                    run: () => openTool("/export/"),
                },
                {
                    id: "exportOslofjord",
                    kind: "open",
                    color: "#3c61d8",
                    enabled: perms.canVbExport.value,
                    run: () => openTool("/vb-export/"),
                },
                {
                    id: "editTranscription",
                    kind: "open",
                    color: "#8b5cf6",
                    enabled:
                        perms.canTranscribe.value ||
                        perms.isTranscriptionAdmin.value,
                    run: () => openToolWithIdPath("/transcription"),
                },
                {
                    id: "preview",
                    kind: "run",
                    color: "#cdbf3a",
                    enabled: perms.canCantemoPreview.value,
                    run: () => trigger("preview", CantemoAction.PREVIEW),
                },
                {
                    id: "transcribe",
                    kind: "run",
                    color: "#3fb84f",
                    enabled: perms.canCantemoTranscribe.value,
                    run: () => trigger("transcribe", CantemoAction.TRANSCRIBE),
                },
                {
                    id: "subtitleFromSubtrans",
                    kind: "run",
                    color: "#3fb84f",
                    enabled: perms.canCantemoSubtitles.value,
                    run: () =>
                        trigger(
                            "subtitleFromSubtrans",
                            CantemoAction.SUBTITLE_FROM_SUBTRANS,
                        ),
                },
                {
                    id: "updateRelations",
                    kind: "run",
                    color: "#3c61d8",
                    enabled: perms.canCantemoRelations.value,
                    run: () =>
                        trigger(
                            "updateRelations",
                            CantemoAction.UPDATE_RELATIONS,
                        ),
                },
            ] satisfies Omit<CantemoChip, "label" | "description">[]
        )
            .filter((c) => c.enabled)
            .map((c) => ({
                ...c,
                label: t(`cantemo.${c.id}.label`),
                description: t(`cantemo.${c.id}.description`),
            })),
    );

    return { chips, loading };
}
