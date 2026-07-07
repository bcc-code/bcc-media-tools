interface Tool {
    label: string;
    icon: string;
    description: string;
    to: string;
    enabled?: boolean;
}

export function useTools() {
    const { t } = useI18n();
    const { me } = useMe();
    const route = useRoute();

    const tools = computed<Tool[]>(() => [
        {
            label: t("tools.bmmUpload.title"),
            icon: "tabler:upload",
            description: t("tools.bmmUpload.description"),
            to: "/upload/bmm/",
            enabled:
                me.value?.bmm &&
                (me.value.bmm.podcasts.length > 0 || me.value.bmm.admin),
        },
        {
            label: t("tools.transcription.title"),
            icon: "tabler:edit",
            description: t("tools.transcription.description"),
            to: "/transcription/",
            enabled:
                me.value?.transcription &&
                (me.value.transcription.mediabanken ||
                    me.value.transcription.admin),
        },
        {
            label: t("tools.export.title"),
            icon: "tabler:file-export",
            description: t("tools.export.description"),
            to: "/export/",
            enabled:
                me.value?.admin ||
                (me.value?.export &&
                    (me.value.export.destinations.length > 0 ||
                        me.value.export.admin ||
                        me.value.export.timedMetadata)),
        },
        {
            label: t("tools.vbExport.title"),
            icon: "tabler:broadcast",
            description: t("tools.vbExport.description"),
            to: "/vb-export/",
            // Shown when the user has access to any VB destination (or is on the page).
            enabled:
                me.value?.admin ||
                (me.value?.vbExport &&
                    (me.value.vbExport.destinations.length > 0 ||
                        me.value.vbExport.admin)) ||
                route.path.startsWith("/vb-export"),
        },
        {
            label: "Shorts generation",
            icon: "tabler:device-mobile",
            description: "Generate shorts from existing videos",
            to: "/shorts/",
        },
        {
            label: t("tools.vault.title"),
            icon: "tabler:building-warehouse",
            description: t("tools.vault.description"),
            to: "/vault/",
            enabled: me.value?.admin || me.value?.vault?.enabled,
        },
        {
            label: t("tools.jobs.title"),
            icon: "tabler:list-check",
            description: t("tools.jobs.description"),
            to: "/jobs/",
        },
        {
            label: t("tools.admin.title"),
            icon: "tabler:settings",
            description: t("tools.admin.description"),
            to: "/admin/",
            enabled: me.value?.admin,
        },
    ]);

    const enabledTools = computed(() =>
        tools.value.filter((t) => t.enabled != false),
    );

    const currentTool = computed(() =>
        tools.value.find((t) => route.path.startsWith(t.to)),
    );

    return { tools, enabledTools, currentTool };
}
