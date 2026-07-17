<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import {
    BMMPermissionSchema,
    CantemoPermissionSchema,
    EditorialPermissionSchema,
    ExportPermissionSchema,
    LiveIngestPermissionSchema,
    PermissionsSchema,
    ShortsPermissionSchema,
    TranscriptionPermissionSchema,
    VaultPermissionSchema,
    VBExportPermissionSchema,
} from "~~/src/gen/api/v1/api_pb";
import type { Permissions } from "~~/src/gen/api/v1/api_pb";
import { motion } from "motion-v";

const props = defineProps<{
    email: string;
    permissions: Permissions;
    languages: string[];
    vxDestinations: string[];
    vbDestinations: string[];
}>();

defineEmits<{
    remove: [];
}>();

// Ensure all permission fields exist to avoid undefined errors
function withDefaultPermissions(p: Permissions): Permissions {
    return create(PermissionsSchema, {
        admin: p.admin ?? false,
        bmm: p.bmm ?? create(BMMPermissionSchema),
        transcription: p.transcription ?? create(TranscriptionPermissionSchema),
        export: p.export ?? create(ExportPermissionSchema),
        vbExport: p.vbExport ?? create(VBExportPermissionSchema),
        cantemo: p.cantemo ?? create(CantemoPermissionSchema),
        vault: p.vault ?? create(VaultPermissionSchema),
        shorts: p.shorts ?? create(ShortsPermissionSchema),
        editorial: p.editorial ?? create(EditorialPermissionSchema),
        liveIngest: p.liveIngest ?? create(LiveIngestPermissionSchema),
        email: p.email ?? "",
    });
}

const exportDestinations = computed(() =>
    destinationOptions(props.vxDestinations),
);
const vbExportDestinations = computed(() =>
    destinationOptions(props.vbDestinations),
);

const perms = reactive(withDefaultPermissions(props.permissions));
const api = useAPI();

watch(perms, () => {
    api.updatePermissions({
        email: props.email,
        permissions: perms,
    });
});

const isOpen = ref(false);
</script>

<template>
    <div
        class="gradient-border bg-surface-default shadow-resting flex flex-col overflow-hidden rounded-xl"
    >
        <LayoutGroup>
            <AnimatePresence>
                <motion.button
                    class="bg-surface-raise flex items-center justify-between p-4"
                    layout
                    @click="isOpen = !isOpen"
                >
                    <div class="flex items-center gap-2">
                        <h3 class="text-text-default">{{ email }}</h3>
                        <Icon
                            name="tabler:chevron-down"
                            :class="[
                                'transition duration-200',
                                { '-rotate-180': isOpen },
                            ]"
                        />
                    </div>
                    <DesignButton
                        size="small"
                        variant="tertiary"
                        intent="danger"
                        icon="tabler:trash"
                        @click.stop="$emit('remove')"
                    >
                        Remove
                    </DesignButton>
                </motion.button>
                <motion.div
                    v-if="isOpen"
                    layout
                    class="divide-border-1 border-border-1 grid max-w-full grid-cols-[1fr_3fr] divide-y overflow-hidden border-t"
                    :initial="{ height: 0 }"
                    :animate="{ height: 'auto' }"
                    :exit="{ height: 0 }"
                >
                    <AdminPermissionViewSection title="General">
                        <DesignSwitch
                            v-model="perms.admin"
                            label="Admin"
                            description="Can manage users and their roles"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection v-if="perms.bmm" title="BMM">
                        <DesignSwitch
                            v-model="perms.bmm.admin"
                            label="BMM Admin"
                            description="Has full access to BMM upload tools"
                        />
                        <DesignSwitch
                            v-model="perms.bmm.integration"
                            label="Integration environment"
                            description="Can upload to the integration environment"
                        />
                        <div class="space-y-1">
                            <label class="text-body-3 text-text-muted block">
                                Podcasts
                            </label>
                            <DesignSelect
                                v-model="perms.bmm.podcasts"
                                multiple
                                :items="[
                                    { label: 'Fra Kåre', value: 'fra-kaare' },
                                    {
                                        label: 'Tjen Gud',
                                        value: 'tjgu-podcast',
                                    },
                                ]"
                                class="w-full max-w-prose"
                            />
                        </div>
                        <div class="space-y-1">
                            <label class="text-body-3 text-text-muted block">
                                Languages
                            </label>
                            <DesignSelect
                                v-model="perms.bmm.languages"
                                multiple
                                :items="
                                    props.languages.map((v) => ({
                                        label: languageCodeToName(v),
                                        value: v,
                                    }))
                                "
                                class="w-full max-w-prose"
                            />
                        </div>
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.transcription"
                        title="Transcription"
                    >
                        <DesignSwitch
                            v-model="perms.transcription.admin"
                            label="Transcription Admin"
                            description="Can correct transcriptions (and preview) any asset in Mediabanken"
                        />
                        <DesignSwitch
                            v-model="perms.transcription.mediabanken"
                            label="Mediabanken"
                            description="Can correct transcriptions shared from Mediabanken"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.export"
                        title="Export"
                    >
                        <DesignSwitch
                            v-model="perms.export.admin"
                            label="Export Admin"
                            description="Can export to all destinations"
                        />
                        <div class="space-y-1">
                            <label class="text-body-3 text-text-muted block">
                                Destinations
                            </label>
                            <DesignSelect
                                v-model="perms.export.destinations"
                                multiple
                                :items="exportDestinations"
                                class="w-full max-w-prose"
                            />
                        </div>
                        <DesignSwitch
                            v-model="perms.export.timedMetadata"
                            label="Timed metadata export"
                            description="Can trigger the timed metadata (VOD) export"
                        />
                        <DesignSwitch
                            v-model="perms.export.bulkExport"
                            label="Bulk export"
                            description="Can paste a list of VX-ids and export them in bulk"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.vbExport"
                        title="VB Export"
                    >
                        <DesignSwitch
                            v-model="perms.vbExport.admin"
                            label="VB Export Admin"
                            description="Can export to all VB destinations"
                        />
                        <div class="space-y-1">
                            <label class="text-body-3 text-text-muted block">
                                Destinations
                            </label>
                            <DesignSelect
                                v-model="perms.vbExport.destinations"
                                multiple
                                :items="vbExportDestinations"
                                class="w-full max-w-prose"
                            />
                        </div>
                        <DesignSwitch
                            v-model="perms.vbExport.bulkExport"
                            label="Bulk export"
                            description="Can paste a list of VX-ids and VB-export them in bulk"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.cantemo"
                        title="Cantemo"
                    >
                        <DesignSwitch
                            v-model="perms.cantemo.preview"
                            label="Make preview"
                            description="Can trigger preview generation"
                        />
                        <DesignSwitch
                            v-model="perms.cantemo.transcribe"
                            label="Transcribe"
                            description="Can trigger transcription"
                        />
                        <DesignSwitch
                            v-model="perms.cantemo.subtitles"
                            label="Update subtitle from Subtrans"
                            description="Can import subtitles from Subtrans"
                        />
                        <DesignSwitch
                            v-model="perms.cantemo.relations"
                            label="Update asset relations"
                            description="Can trigger the asset relations update flow"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.vault"
                        title="Vault"
                    >
                        <DesignSwitch
                            v-model="perms.vault.enabled"
                            label="Vault search"
                            description="Can search Mediabanken and view item previews"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.shorts"
                        title="Shorts"
                    >
                        <DesignSwitch
                            v-model="perms.shorts.enabled"
                            label="Shorts generation"
                            description="Can open the shorts editor and submit shorts for generation"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.liveIngest"
                        title="Live ingest"
                    >
                        <DesignSwitch
                            v-model="perms.liveIngest.enabled"
                            label="Live ingest"
                            description="Can send the finish signal to a running live ingest"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.editorial"
                        title="Editorial"
                    >
                        <DesignSwitch
                            v-model="perms.editorial.enabled"
                            label="Editorial approval"
                            description="Can see all sessions and accept/reject markers for publishing"
                        />
                        <DesignSwitch
                            v-model="perms.editorial.admin"
                            label="Editorial editing"
                            description="Can add, remove and edit markers and sessions"
                        />
                    </AdminPermissionViewSection>
                </motion.div>
            </AnimatePresence>
        </LayoutGroup>
    </div>
</template>
