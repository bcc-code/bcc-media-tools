<script lang="tsx" setup>
import type { Permissions } from "#imports";
import { BccButton, BccInput } from "@bcc-code/design-library-vue";

const { me } = useMe();

const permissions = ref<{
    [key: string]: Permissions;
}>();

onMounted(async () => {
    permissions.value = await $fetch("/api/permissions/list");
});

const newEmail = ref<string>();

const addEmail = async () => {
    if (newEmail.value) {
        await $fetch("/api/permissions/set", {
            method: "PUT",
            body: {
                email: newEmail.value,
                permissions: {
                    admin: false,
                    bmm: {
                        albums: [],
                        languages: [],
                    },
                },
            },
        });
        permissions.value = await $fetch("/api/permissions/list");
        newEmail.value = "";
    }
};

const removeEmail = async (email: string) => {
    await $fetch("/api/permissions/set", {
        method: "PUT",
        body: {
            email,
            permissions: null,
        },
    });
    permissions.value = await $fetch("/api/permissions/list");
};
</script>

<template>
    <div class="flex h-screen w-screen" v-if="me?.admin">
        <div
            class="mx-auto w-full max-w-screen-md rounded-lg border-2 border-slate-950 bg-zinc-100 p-8 text-black"
        >
            <h3 class="text-lg">Admin</h3>
            <div class="flex flex-col gap-4" v-if="permissions">
                <PermissionView
                    v-for="[email, perms] in Object.entries(permissions)"
                    :email="email"
                    :permissions="perms"
                    @remove="removeEmail(email)"
                />
                <div class="flex">
                    <BccInput v-model="newEmail" type="email"></BccInput>
                    <BccButton @click="addEmail">Add</BccButton>
                </div>
            </div>
        </div>
    </div>
</template>
