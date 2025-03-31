<script lang="ts" setup>
import { BmmEnvironment, Permissions } from "~/src/gen/api/v1/api_pb";
import {
    BccButton,
    BccFormLabel,
    BccToggle,
} from "@bcc-code/design-library-vue";
import { motion } from "motion-v";

const props = defineProps<{
    email: string;
    permissions: Permissions;
}>();

defineEmits<{
    remove: [];
}>();

const perms = reactive(props.permissions);
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
    <div class="flex flex-col rounded-2xl border bg-white">
        <LayoutGroup>
            <AnimatePresence multiple>
                <motion.button
                    class="flex items-center justify-between p-4"
                    layout
                    @click="(isOpen = !isOpen)"
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
                    <BccButton
                        size="sm"
                        variant="tertiary"
                        context="danger"
                        @click.stop="$emit('remove')"
                    >
                        Remove
                    </BccButton>
                </motion.button>
                <motion.div
                    v-if="isOpen"
                    layout
                    class="grid max-w-full grid-cols-[1fr_3fr] gap-4 overflow-hidden border-t p-4"
                    :initial="{ height: 0 }"
                    :animate="{ height: 'auto' }"
                    :exit="{ height: 0 }"
                >
                    <div
                        class="grid-span-1 col-span-full grid grid-cols-subgrid items-baseline gap-4 rounded-lg border px-4 py-3"
                    >
                        <h3>General</h3>
                        <div class="flex gap-4">
                            <div>
                                <BccFormLabel>Admin</BccFormLabel>
                                <BccToggle v-model="perms.admin" />
                            </div>
                        </div>
                    </div>
                    <div
                        class="col-span-full grid grid-cols-subgrid grid-rows-1 items-baseline rounded-xl border px-4 py-3"
                    >
                        <h3>BMM</h3>
                        <div class="flex flex-wrap gap-4">
                            <div>
                                <div>
                                    <BccFormLabel>BMM Admin</BccFormLabel>
                                    <BccToggle v-model="perms!.bmm!.admin" />
                                </div>
                                <div>
                                    <BccFormLabel>
                                        Integration environment
                                    </BccFormLabel>
                                    <BccToggle
                                        v-model="perms!.bmm!.integration"
                                    />
                                </div>
                            </div>
                            <div>
                                <BccFormLabel>Podcasts</BccFormLabel>
                                <MultiSelector
                                    :available="['fra-kaare', 'hvhe-podcast']"
                                    v-model="perms.bmm!.podcasts"
                                />
                            </div>
                            <div>
                                <BccFormLabel>Languages</BccFormLabel>
                                <MultiSelector
                                    :available="availableLanguages"
                                    v-model="perms.bmm!.languages"
                                    :label-transformer="
                                        (v) => languageCodeToName(v)
                                    "
                                />
                            </div>
                        </div>
                    </div>
                </motion.div>
            </AnimatePresence>
        </LayoutGroup>
    </div>
</template>
