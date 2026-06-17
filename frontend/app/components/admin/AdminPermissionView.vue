<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import {
    BMMPermissionSchema,
    CantemoPermissionSchema,
    ExportPermissionSchema,
    PermissionsSchema,
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
        class="border-accented bg-default flex flex-col overflow-hidden rounded-xl border"
    >
        <LayoutGroup>
            <AnimatePresence>
                <motion.button
                    class="bg-muted flex items-center justify-between p-4"
                    layout
                    @click="isOpen = !isOpen"
                >
                    <div class="flex items-center gap-2">
                        <h3>{{ email }}</h3>
                        <Icon
                            name="heroicons:chevron-down"
                            :class="[
                                'transition duration-200',
                                { '-rotate-180': isOpen },
                            ]"
                        />
                    </div>
                    <UButton
                        size="sm"
                        variant="ghost"
                        color="error"
                        icon="heroicons:trash"
                        @click.stop="$emit('remove')"
                    >
                        Remove
                    </UButton>
                </motion.button>
                <motion.div
                    v-if="isOpen"
                    layout
                    class="divide-default border-accented grid max-w-full grid-cols-[1fr_3fr] divide-y overflow-hidden border-t"
                    :initial="{ height: 0 }"
                    :animate="{ height: 'auto' }"
                    :exit="{ height: 0 }"
                >
                    <AdminPermissionViewSection title="General">
                        <USwitch
                            v-model="perms.admin"
                            label="Admin"
                            description="Can manage users and their roles"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection v-if="perms.bmm" title="BMM">
                        <USwitch
                            v-model="perms.bmm.admin"
                            label="BMM Admin"
                            description="Has full access to BMM upload tools"
                        />
                        <USwitch
                            v-model="perms.bmm.integration"
                            label="Integration environment"
                            description="Can upload to the integration environment"
                        />
                        <UFormField label="Podcasts">
                            <USelect
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
                        </UFormField>
                        <UFormField label="Languages">
                            <USelect
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
                        </UFormField>
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.transcription"
                        title="Transcription"
                    >
                        <USwitch
                            v-model="perms.transcription.admin"
                            label="Transcription Admin"
                            description="Can correct transcriptions (and preview) any asset in Mediabanken"
                        />
                        <USwitch
                            v-model="perms.transcription.mediabanken"
                            label="Mediabanken"
                            description="Can correct transcriptions shared from Mediabanken"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.export"
                        title="Export"
                    >
                        <USwitch
                            v-model="perms.export.admin"
                            label="Export Admin"
                            description="Can export to all destinations"
                        />
                        <UFormField label="Destinations">
                            <USelect
                                v-model="perms.export.destinations"
                                multiple
                                :items="exportDestinations"
                                class="w-full max-w-prose"
                            />
                        </UFormField>
                        <USwitch
                            v-model="perms.export.timedMetadata"
                            label="Timed metadata export"
                            description="Can trigger the timed metadata (VOD) export"
                        />
                        <USwitch
                            v-model="perms.export.bulkExport"
                            label="Bulk export"
                            description="Can paste a list of VX-ids and export them in bulk"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.vbExport"
                        title="VB Export"
                    >
                        <USwitch
                            v-model="perms.vbExport.admin"
                            label="VB Export Admin"
                            description="Can export to all VB destinations"
                        />
                        <UFormField label="Destinations">
                            <USelect
                                v-model="perms.vbExport.destinations"
                                multiple
                                :items="vbExportDestinations"
                                class="w-full max-w-prose"
                            />
                        </UFormField>
                        <USwitch
                            v-model="perms.vbExport.bulkExport"
                            label="Bulk export"
                            description="Can paste a list of VX-ids and VB-export them in bulk"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.cantemo"
                        title="Cantemo"
                    >
                        <USwitch
                            v-model="perms.cantemo.preview"
                            label="Make preview"
                            description="Can trigger preview generation"
                        />
                        <USwitch
                            v-model="perms.cantemo.transcribe"
                            label="Transcribe"
                            description="Can trigger transcription"
                        />
                        <USwitch
                            v-model="perms.cantemo.subtitles"
                            label="Update subtitle from Subtrans"
                            description="Can import subtitles from Subtrans"
                        />
                        <USwitch
                            v-model="perms.cantemo.relations"
                            label="Update asset relations"
                            description="Can trigger the asset relations update flow"
                        />
                    </AdminPermissionViewSection>
                    <AdminPermissionViewSection
                        v-if="perms.vault"
                        title="Vault"
                    >
                        <USwitch
                            v-model="perms.vault.enabled"
                            label="Vault search"
                            description="Can search Mediabanken and view item previews"
                        />
                    </AdminPermissionViewSection>
                </motion.div>
            </AnimatePresence>
        </LayoutGroup>
    </div>
</template>
