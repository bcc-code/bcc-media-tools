<script lang="tsx" setup>
import { BmmEnvironment, Permissions } from "~~/src/gen/api/v1/api_pb";

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

const { data: availableLanguages } = useAsyncData(
    () => `languages`,
    () => api.getLanguages({ environment: BmmEnvironment.Production }),
    {
        default: () => [],
        transform: (data) => data.Languages.map((l) => l.code),
    },
);
</script>

<template>
    <div class="flex h-screen w-screen" v-if="me?.admin">
        <div class="mx-auto w-full max-w-screen-md p-8">
            <div class="mb-8 flex items-center justify-between gap-2">
                <h2 class="text-2xl font-bold">Admin</h2>
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
                <UButton @click="showNewEmailForm = true">
                    Add new email
                </UButton>
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
                    class="border-default flex items-center gap-2 rounded-2xl border-2 border-dashed p-4"
                    @submit.prevent="addEmail"
                >
                    <UInput
                        v-model="newEmail"
                        type="email"
                        placeholder="john@doe.com"
                    />
                    <UButton type="submit" variant="soft">Add</UButton>
                </form>
                <AdminPermissionView
                    v-for="[email, perms] in Object.entries(
                        filteredPermissions,
                    )"
                    :key="email"
                    :email="email"
                    :permissions="perms"
                    :languages="availableLanguages"
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
            <UButton variant="soft" block>Go home</UButton>
        </NuxtLink>
    </div>
</template>
