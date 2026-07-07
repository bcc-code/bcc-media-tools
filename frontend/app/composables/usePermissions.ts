/**
 * Frontend mirror of the backend `Can*` permission methods
 * (`backend/api/v1/permissions.go`). This is the single place the UI derives
 * permission predicates, so it can't drift field-by-field from the server.
 *
 * These gate UI visibility only — the backend still authorizes every action, so
 * hiding something here is never the security boundary.
 */
export function usePermissions() {
    const { me } = useMe();

    const admin = computed(() => !!me.value?.admin);

    // CanUpload: admin, BMM admin, or at least one assigned language.
    const canUpload = computed(
        () =>
            admin.value ||
            (!!me.value?.bmm &&
                (me.value.bmm.admin || me.value.bmm.languages.length > 0)),
    );

    // CanExport: admin, export admin, a destination, or timed-metadata access.
    const canExport = computed(
        () =>
            admin.value ||
            (!!me.value?.export &&
                (me.value.export.admin ||
                    me.value.export.destinations.length > 0 ||
                    me.value.export.timedMetadata)),
    );

    const canExportTimedMetadata = computed(
        () =>
            admin.value ||
            (!!me.value?.export &&
                (me.value.export.admin || me.value.export.timedMetadata)),
    );

    const canBulkExport = computed(
        () => admin.value || !!me.value?.export?.bulkExport,
    );

    const canVBExport = computed(
        () =>
            admin.value ||
            (!!me.value?.vbExport &&
                (me.value.vbExport.admin ||
                    me.value.vbExport.destinations.length > 0)),
    );

    const canBulkVBExport = computed(
        () => admin.value || !!me.value?.vbExport?.bulkExport,
    );

    const canPreview = computed(
        () => admin.value || !!me.value?.cantemo?.preview,
    );
    const canTranscribe = computed(
        () => admin.value || !!me.value?.cantemo?.transcribe,
    );
    const canCantemoSubtitles = computed(
        () => admin.value || !!me.value?.cantemo?.subtitles,
    );
    const canCantemoRelations = computed(
        () => admin.value || !!me.value?.cantemo?.relations,
    );

    const canVault = computed(() => admin.value || !!me.value?.vault?.enabled);

    // Transcription-editor access deliberately does NOT honor global admin —
    // it requires the transcription permission itself (matches the backend
    // handler in transcription.go).
    const canTranscription = computed(
        () =>
            !!me.value?.transcription &&
            (me.value.transcription.admin ||
                me.value.transcription.mediabanken),
    );

    // The stricter transcription-admin flag (not implied by global admin),
    // used for admin-only affordances inside the transcription editor.
    const transcriptionAdmin = computed(() => !!me.value?.transcription?.admin);

    // CanViewJobs: admin or access to any tool that produces workflows.
    const canViewJobs = computed(
        () =>
            admin.value ||
            canExport.value ||
            canVBExport.value ||
            canUpload.value ||
            (!!me.value?.bmm && me.value.bmm.podcasts.length > 0) ||
            canTranscription.value ||
            canVault.value,
    );

    // Destination-scoped checks mirror CanExportTo / CanVBExportTo. Kept as
    // functions since they take an argument; they still track `me` reactively
    // when called during render.
    const canExportTo = (destination: string) =>
        admin.value ||
        (!!me.value?.export &&
            (me.value.export.admin ||
                me.value.export.destinations.includes(destination)));

    const canVBExportTo = (destination: string) =>
        admin.value ||
        (!!me.value?.vbExport &&
            (me.value.vbExport.admin ||
                me.value.vbExport.destinations.includes(destination)));

    return {
        admin,
        canUpload,
        canExport,
        canExportTimedMetadata,
        canBulkExport,
        canVBExport,
        canBulkVBExport,
        canPreview,
        canTranscribe,
        canCantemoSubtitles,
        canCantemoRelations,
        canVault,
        canTranscription,
        transcriptionAdmin,
        canViewJobs,
        canExportTo,
        canVBExportTo,
    };
}
