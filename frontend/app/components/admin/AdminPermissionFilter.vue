<script setup lang="ts">
import { BmmEnvironment } from "~~/src/gen/api/v1/api_pb";
import type { Language, Permissions } from "~~/src/gen/api/v1/api_pb";

const props = defineProps<{
    permissions: Record<string, Permissions>;
}>();

const emit = defineEmits<{
    "update:filteredPermissions": [permissions: Record<string, Permissions>];
}>();

const searchQuery = defineModel("search", { default: "" });

const api = useAPI();

const roles = ref([
    {
        label: "Admin",
        value: "admin",
    },
    {
        label: "BMM Admin",
        value: "bmm.admin",
    },
    {
        label: "Transcription Admin",
        value: "transcription.admin",
    },
]);
const selectedRoles = defineModel<string[]>("roles", { default: [] });

const languages = ref<Language[]>();
const selectedLanguages = defineModel<string[]>("languages", { default: [] });
const languageItems = computed(() => {
    return (
        languages.value?.map((l) => ({
            label: languageCodeToName(l.code),
            value: l.code,
        })) ?? []
    );
});
onMounted(async () => {
    languages.value = (
        await api.getLanguages({
            environment: BmmEnvironment.Production,
        })
    ).Languages;
});

const getDeepValue = (object: any, path: string) => {
    const keys = path.split(".");
    let value = object;
    for (let i = 0; i < keys.length; i++) {
        const key = keys[i];
        if (!key) return false;
        value = value[key];
        if (value === undefined) {
            return false;
        }
    }
    return value;
};
const filteredPermissions = computed(() => {
    if (!props.permissions) return {};

    let filtered = props.permissions;

    if (selectedLanguages.value.length > 0) {
        filtered = Object.fromEntries(
            Object.entries(filtered).filter(([email, perm]) =>
                selectedLanguages.value.some((l) =>
                    perm.bmm?.languages.includes(l),
                ),
            ),
        );
    }
    if (selectedRoles.value.length > 0) {
        filtered = Object.fromEntries(
            Object.entries(filtered).filter(([email, perm]) =>
                selectedRoles.value.some((r) => getDeepValue(perm, r)),
            ),
        );
    }
    if (searchQuery.value.trim()) {
        filtered = Object.fromEntries(
            Object.entries(filtered).filter(([email]) =>
                email
                    .toLowerCase()
                    .includes(searchQuery.value.toLowerCase().trim()),
            ),
        );
    }

    return filtered;
});

watch(
    filteredPermissions,
    (filtered) => {
        emit("update:filteredPermissions", filtered);
    },
    { immediate: true },
);

const isShowingClearButton = computed(() => {
    return selectedLanguages.value.length > 0 || selectedRoles.value.length > 0;
});

function clear() {
    selectedLanguages.value = [];
    selectedRoles.value = [];
}
</script>

<template>
    <div class="relative flex w-full items-center gap-2">
        <USelect
            v-model="selectedLanguages"
            :items="languageItems"
            placeholder="Languages"
            multiple
            class="w-32"
        />
        <USelect
            placeholder="Roles"
            :items="roles"
            v-model="selectedRoles"
            multiple
            class="w-24"
        />
        <UButton v-if="isShowingClearButton" variant="ghost" @click="clear">
            Clear
        </UButton>
        <UInput
            v-model="searchQuery"
            clearable
            placeholder="Search email..."
            leading-icon="heroicons:magnifying-glass"
            class="ml-auto"
        >
            <template #trailing>
                <UButton
                    v-if="searchQuery"
                    size="xs"
                    color="neutral"
                    variant="outline"
                    @click="searchQuery = ''"
                >
                    Clear
                </UButton>
            </template>
        </UInput>
    </div>
</template>
