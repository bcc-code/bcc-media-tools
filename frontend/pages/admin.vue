<script lang="tsx" setup>
import { BccButton, BccInput } from "@bcc-code/design-library-vue";
import { Permissions } from "~/src/gen/api/v1/api_pb";

const { me } = useMe();
const api = useAPI();

const permissions = ref<{
    [key: string]: Permissions;
}>();

onMounted(async () => {
    permissions.value = (await api.listPermissions({})).permissions;
});

const newEmail = ref<string>();

const addEmail = async () => {
    if (newEmail.value) {
        await api.updatePermissions({
            email: newEmail.value,
            permissions: {
                admin: false,
                bmm: {
                    albums: [],
                    languages: [],
                    admin: false,
                },
            },
        });

        permissions.value = (await api.listPermissions({})).permissions;
        newEmail.value = "";
    }
};

const removeEmail = async (email: string) => {
    await api.deletePermissions({ email });
    permissions.value = (await api.listPermissions({})).permissions;
};
</script>

<template>
    <div class="flex h-screen w-screen" v-if="me?.admin">
        <div
            class="mx-auto w-full max-w-screen-md rounded-lg border-2 border-neutral-950 bg-zinc-100 p-8 text-black"
        >
            <h3 class="text-lg">Admin</h3>
            <div class="flex flex-col gap-4" v-if="permissions">
                <PermissionView
                    v-for="[email, perms] in Object.entries(permissions)"
                    :email="email"
                    :permissions="perms"
                    :key="email"
                    @remove="removeEmail(email)"
                />
                <div class="flex">
                    <BccInput v-model="newEmail" type="email"></BccInput>
                    <BccButton @click="addEmail">Add</BccButton>
                </div>
            </div>
        </div>
    </div>
    <div v-else>
        <h1>You are not an admin</h1>
    </div>
</template>
