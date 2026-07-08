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
    const {
        admin,
        canUpload,
        canTranscription,
        canExport,
        canVBExport,
        canVault,
        canShorts,
        canViewJobs,
    } = usePermissions();

    const tools = computed<Tool[]>(() => [
        {
            label: t("tools.bmmUpload.title"),
            icon: "tabler:upload",
            description: t("tools.bmmUpload.description"),
            to: "/upload/bmm/",
            enabled: canUpload.value,
        },
        {
            label: t("tools.transcription.title"),
            icon: "tabler:edit",
            description: t("tools.transcription.description"),
            to: "/transcription/",
            enabled: canTranscription.value,
        },
        {
            label: t("tools.export.title"),
            icon: "tabler:file-export",
            description: t("tools.export.description"),
            to: "/export/",
            enabled: canExport.value,
        },
        {
            label: t("tools.vbExport.title"),
            icon: "tabler:broadcast",
            description: t("tools.vbExport.description"),
            to: "/vb-export/",
            // Shown when the user has access to any VB destination (or is on the page).
            enabled: canVBExport.value || route.path.startsWith("/vb-export"),
        },
        {
            label: "Shorts generation",
            icon: "tabler:device-mobile",
            description: "Generate shorts from existing videos",
            to: "/shorts/",
            enabled: canShorts.value,
        },
        {
            label: t("tools.vault.title"),
            icon: "tabler:building-warehouse",
            description: t("tools.vault.description"),
            to: "/vault/",
            enabled: canVault.value,
        },
        {
            label: t("tools.jobs.title"),
            icon: "tabler:list-check",
            description: t("tools.jobs.description"),
            to: "/jobs/",
            enabled: canViewJobs.value,
        },
        {
            label: t("tools.admin.title"),
            icon: "tabler:settings",
            description: t("tools.admin.description"),
            to: "/admin/",
            enabled: admin.value,
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
