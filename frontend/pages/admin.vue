<script lang="tsx" setup>
import { BccButton, BccInput } from "@bcc-code/design-library-vue";
import { Permissions } from "~/src/gen/api/v1/api_pb";

useHead({
    title: "Admin",
});

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
        showNewEmailForm.value = false;
    }
};

const removeEmail = async (email: string) => {
    if (!confirm(`Are you sure you want to remove ${email}?`)) return;
    await api.deletePermissions({ email });
    permissions.value = (await api.listPermissions({})).permissions;
};

const searchQuery = ref("");
const filteredPermissions = computed(() => {
    if (!searchQuery.value) return permissions.value;
    return Object.fromEntries(
        Object.entries(permissions.value ?? {}).filter(([email]) =>
            email.toLowerCase().includes(searchQuery.value.toLowerCase()),
        ),
    );
});

const showNewEmailForm = ref(false);
</script>

<template>
    <div class="flex h-screen w-screen" v-if="me?.admin">
        <div class="mx-auto w-full max-w-screen-md p-8 text-black">
            <div class="mb-8 flex items-center justify-between gap-2">
                <h2 class="text-2xl font-bold">Admin</h2>
                <BccInput
                    v-model="searchQuery"
                    clearable
                    placeholder="Search email..."
                    class="ml-auto"
                />
                <BccButton @click="(showNewEmailForm = true)">
                    Add new email
                </BccButton>
            </div>
            <TransitionGroup
                v-if="filteredPermissions"
                tag="div"
                class="flex flex-col gap-4"
                enter-active-class="transition duration-300 ease-out"
                enter-from-class="opacity-0 scale-95"
                enter-to-class="opacity-100 scale-100"
                leave-active-class="transition duration-300 ease-out absolute"
                leave-from-class="opacity-100 scale-100"
                leave-to-class="opacity-0 scale-95"
                move-class="transition duration-300 ease-out"
            >
                <form
                    v-if="showNewEmailForm"
                    class="flex items-center gap-2 rounded-2xl border-2 border-dashed border-slate-200 p-4"
                    @submit.prevent="addEmail"
                >
                    <BccInput v-model="newEmail" type="email" clearable />
                    <BccButton type="submit" variant="secondary">
                        Add
                    </BccButton>
                </form>
                <PermissionView
                    v-for="[email, perms] in Object.entries(
                        filteredPermissions,
                    )"
                    :email="email"
                    :permissions="perms"
                    :key="email"
                    @remove="removeEmail(email)"
                />
            </TransitionGroup>
        </div>
    </div>
    <div
        v-else
        class="flex h-screen w-screen flex-col items-center justify-center gap-4"
    >
        <h1 class="text-2xl font-bold">You are not an admin</h1>
        <NuxtLink to="/">
            <BccButton variant="secondary">Go home</BccButton>
        </NuxtLink>
    </div>
</template>
