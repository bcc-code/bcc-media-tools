<script lang="ts" setup>
import { BmmEnvironment, Permissions } from "~/src/gen/api/v1/api_pb";
import { motion } from "motion-v";

const props = defineProps<{
    email: string;
    permissions: Permissions;
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

const availableLanguages = ref<string[]>([]);
api.getLanguages({ environment: BmmEnvironment.Production }).then(
    (result) =>
        (availableLanguages.value = result.Languages.map((f) => f.code)),
);

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
        class="flex flex-col rounded-xl border border-neutral-300 bg-white dark:border-neutral-700 dark:bg-neutral-900"
    >
        <LayoutGroup>
            <AnimatePresence>
                <motion.button
                    class="flex items-center justify-between p-4"
                    layout
                    @click="isOpen = !isOpen"
                >
                    <div class="flex items-center gap-2">
                        <h3 class="text-lg">{{ email }}</h3>
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
                    class="grid max-w-full grid-cols-[1fr_3fr] gap-4 overflow-hidden border-t border-neutral-300 p-4 dark:border-neutral-700"
                    :initial="{ height: 0 }"
                    :animate="{ height: 'auto' }"
                    :exit="{ height: 0 }"
                >
                    <div
                        class="grid-span-1 col-span-full grid grid-cols-subgrid items-baseline gap-4 rounded-lg border border-neutral-300 px-4 py-3 dark:border-neutral-700"
                    >
                        <h3>General</h3>
                        <div class="flex flex-wrap gap-4">
                            <USwitch
                                v-model="perms.admin"
                                label="Admin"
                                description="Can manage users and their roles"
                            />
                        </div>
                    </div>
                    <div
                        class="col-span-full grid grid-cols-subgrid grid-rows-1 items-baseline rounded-xl border border-neutral-300 px-4 py-3 dark:border-neutral-700"
                    >
                        <h3>BMM</h3>
                        <div v-if="perms.bmm" class="flex flex-wrap gap-4">
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
                            <div class="flex flex-col gap-1">
                                <label for="podcasts" class="text-sm">
                                    Podcasts
                                </label>
                                <MultiSelector
                                    v-model="perms.bmm.podcasts"
                                    id="podcasts"
                                    :available="['fra-kaare', 'tjgu-podcast']"
                                />
                            </div>
                            <div class="flex flex-col gap-1">
                                <label for="languages" class="text-sm">
                                    Languages
                                </label>
                                <MultiSelector
                                    v-model="perms.bmm.languages"
                                    id="languages"
                                    :available="availableLanguages"
                                    :label-transformer="
                                        (v) => languageCodeToName(v)
                                    "
                                />
                            </div>
                        </div>
                    </div>
                    <div
                        class="col-span-full grid grid-cols-subgrid grid-rows-1 items-baseline rounded-xl border border-neutral-300 px-4 py-3 dark:border-neutral-700"
                    >
                        <h3>Transcription</h3>
                        <div
                            v-if="perms.transcription"
                            class="flex flex-wrap gap-4"
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
                        </div>
                    </div>
                </motion.div>
            </AnimatePresence>
        </LayoutGroup>
    </div>
</template>
