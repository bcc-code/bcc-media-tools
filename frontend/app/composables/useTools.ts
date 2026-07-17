interface Tool {
    label: string;
    icon: string;
    description: string;
    to: string;
    enabled?: boolean;
}

export function useTools() {
    const { t } = useI18n();
    const route = useRoute();
    const perms = usePermissions();

    const tools = computed<Tool[]>(() => [
        {
            label: t("tools.bmmUpload.title"),
            icon: "tabler:upload",
            description: t("tools.bmmUpload.description"),
            to: "/upload/bmm/",
            enabled: perms.canUploadBmm.value,
        },
        {
            label: t("tools.transcription.title"),
            icon: "tabler:edit",
            description: t("tools.transcription.description"),
            to: "/transcription/",
            enabled:
                perms.canTranscribe.value || perms.isTranscriptionAdmin.value,
        },
        {
            label: t("tools.export.title"),
            icon: "tabler:file-export",
            description: t("tools.export.description"),
            to: "/export/",
            enabled: perms.canExport.value,
        },
        {
            label: t("tools.vbExport.title"),
            icon: "tabler:broadcast",
            description: t("tools.vbExport.description"),
            to: "/vb-export/",
            // Shown when the user has access to any VB destination (or is on the page).
            enabled:
                perms.canVbExport.value || route.path.startsWith("/vb-export"),
        },
        {
            label: "Shorts generation",
            icon: "tabler:device-mobile",
            description: "Generate shorts from existing videos",
            to: "/shorts/",
            enabled: perms.canUseShorts.value,
        },
        {
            label: t("tools.vault.title"),
            icon: "tabler:building-warehouse",
            description: t("tools.vault.description"),
            to: "/vault/",
            enabled: perms.canUseVault.value,
        },
        {
            label: t("tools.editorial.title"),
            icon: "tabler:checklist",
            description: t("tools.editorial.description"),
            to: "/editorial/",
            enabled: perms.canUseEditorial.value,
        },
        {
            label: t("tools.liveIngest.title"),
            icon: "tabler:antenna-bars-5",
            description: t("tools.liveIngest.description"),
            to: "/live-ingest/",
            enabled: perms.canUseLiveIngest.value,
        },
        {
            label: t("tools.admin.title"),
            icon: "tabler:settings",
            description: t("tools.admin.description"),
            to: "/admin/",
            enabled: perms.isAdmin.value,
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
