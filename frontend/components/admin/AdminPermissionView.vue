<script lang="ts" setup>
import { Permissions } from "~/src/gen/api/v1/api_pb";
import { motion } from "motion-v";

const props = defineProps<{
    email: string;
    permissions: Permissions;
    languages: string[];
}>();

defineEmits<{
    remove: [];
}>();

// Ensure all permission fields exist to avoid undefined errors
function withDefaultPermissions(p: Permissions): Permissions {
    return {
        admin: p.admin ?? false,
        bmm: p.bmm ?? {
            admin: false,
            integration: false,
            podcasts: [],
            languages: [],
        },
        transcription: p.transcription ?? { admin: false, mediabanken: false },
        email: p.email ?? "",
    };
}

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
        class="flex flex-col overflow-hidden rounded-xl border border-neutral-300 bg-white dark:border-neutral-700 dark:bg-neutral-900"
    >
        <LayoutGroup>
            <AnimatePresence>
                <motion.button
                    class="flex items-center justify-between bg-neutral-50 p-4 dark:bg-neutral-800"
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
                    class="grid max-w-full grid-cols-[1fr_3fr] divide-y divide-neutral-200 overflow-hidden border-t border-neutral-300 dark:divide-neutral-800 dark:border-neutral-700"
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
                </motion.div>
            </AnimatePresence>
        </LayoutGroup>
    </div>
</template>
