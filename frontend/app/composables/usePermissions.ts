import type { Permissions } from "~~/src/gen/api/v1/api_pb";

// Single source of truth for what the current user is allowed to do.
// Every capability check in the app should go through here rather than
// reading `me.value?.<perm>` inline, so the gating rules live in one place.
//
// `admin` implies every capability.
export function usePermissions() {
    const { me, loading } = useMe();

    // Small helper so each capability reads as "admin OR the specific rule".
    const cap = (rule: (m: Permissions) => boolean | undefined) =>
        computed(() => {
            const m = me.value;
            if (!m) return false;
            return !!(m.admin || rule(m));
        });

    const isAdmin = computed(() => !!me.value?.admin);

    const isBmmAdmin = cap((m) => !!m.bmm?.admin);
    const canUploadBmm = cap(
        (m) => !!m.bmm && (m.bmm.podcasts.length > 0 || m.bmm.admin),
    );

    const canTranscribe = cap(
        (m) => !!m.transcription && m.transcription.mediabanken,
    );
    const isTranscriptionAdmin = cap((m) => !!m.transcription?.admin);

    const canExport = cap(
        (m) =>
            !!m.export &&
            (m.export.destinations.length > 0 ||
                m.export.admin ||
                m.export.timedMetadata),
    );
    const canExportTimedMetadata = cap((m) => !!m.export?.timedMetadata);
    const canBulkExport = cap((m) => !!m.export?.bulkExport);

    const canVbExport = cap(
        (m) =>
            !!m.vbExport &&
            (m.vbExport.destinations.length > 0 || m.vbExport.admin),
    );
    const canVbBulkExport = cap((m) => !!m.vbExport?.bulkExport);

    const canUseShorts = cap((m) => !!m.shorts?.enabled);

    const canUseVault = cap((m) => !!m.vault?.enabled);

    const canUseEditorial = cap(
        (m) => !!m.editorial && (m.editorial.enabled || m.editorial.admin),
    );
    const canEditEditorial = cap((m) => !!m.editorial?.admin);

    const canUseLiveIngest = cap((m) => !!m.liveIngest?.enabled);

    const canCantemoPreview = cap((m) => !!m.cantemo?.preview);
    const canCantemoTranscribe = cap((m) => !!m.cantemo?.transcribe);
    const canCantemoSubtitles = cap((m) => !!m.cantemo?.subtitles);
    const canCantemoRelations = cap((m) => !!m.cantemo?.relations);

    return {
        me,
        loading,
        isAdmin,
        isBmmAdmin,
        canUploadBmm,
        isTranscriptionAdmin,
        canTranscribe,
        canExport,
        canExportTimedMetadata,
        canBulkExport,
        canVbExport,
        canVbBulkExport,
        canUseShorts,
        canUseVault,
        canUseEditorial,
        canEditEditorial,
        canUseLiveIngest,
        canCantemoPreview,
        canCantemoTranscribe,
        canCantemoSubtitles,
        canCantemoRelations,
    };
}
