<script lang="ts" setup>
import type { Permissions } from "#imports";
import {
    BccButton,
    BccFormLabel,
    BccToggle,
} from "@bcc-code/design-library-vue";

const props = defineProps<{
    email: string;
    permissions: Permissions;
}>();

defineEmits<{
    remove: [];
}>();

const perms = ref<Permissions>(props.permissions);

watch(perms, () => {
    $fetch("/api/permissions/set", {
        method: "PUT",
        body: {
            email: props.email,
            permissions: perms.value,
        },
    });
});

const admin = computed({
    get: () => perms.value.admin === true,
    set: (value: boolean) => {
        perms.value = {
            ...perms.value,
            admin: value,
        };
    },
});

const bmmAlbums = computed({
    get() {
        return perms.value.bmm.albums;
    },
    set(v) {
        perms.value = {
            ...perms.value,
            bmm: {
                ...perms.value.bmm,
                albums: v,
            },
        };
    },
});

const bmmAvailableLanguages = computed({
    get() {
        return perms.value.bmm.languages;
    },
    set(v) {
        perms.value = {
            ...perms.value,
            bmm: {
                ...perms.value.bmm,
                languages: v,
            },
        };
    },
});
</script>

<template>
    <div class="flex flex-col rounded-lg border bg-white p-4">
        <div class="flex justify-between">
            <h3 class="text-lg">{{ email }}</h3>
            <BccButton @click="$emit('remove')" size="sm">Remove</BccButton>
        </div>
        <div class="flex gap-4">
            <div class="flex flex-col rounded border px-4 py-2">
                <h3>General</h3>
                <div class="flex gap-4">
                    <div>
                        <BccFormLabel>Admin</BccFormLabel>
                        <BccToggle v-model="admin" />
                    </div>
                </div>
            </div>
            <div class="flex flex-col rounded border px-4 py-2">
                <h3>BMM</h3>
                <div class="flex gap-4">
                    <div>
                        <BccFormLabel>Albums</BccFormLabel>
                        <MultiSelector
                            :available="['fra-kaare', 'romans']"
                            v-model="bmmAlbums"
                        />
                    </div>
                    <div>
                        <BccFormLabel>Languages</BccFormLabel>
                        <MultiSelector
                            :available="bmmLanguages"
                            v-model="bmmAvailableLanguages"
                        />
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
