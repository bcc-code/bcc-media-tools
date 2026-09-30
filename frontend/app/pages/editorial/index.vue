<script setup lang="ts">
import type { EditorialSession } from "~~/src/gen/api/v1/api_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { timestampDate } from "@bufbuild/protobuf/wkt";

const { t } = useI18n();
const api = useAPI();
const perms = usePermissions();
const toaster = useToast();

useHead({ title: t("tools.editorial.title") });

const analytics = useAnalytics();
onMounted(() => analytics.page({ id: "editorial", title: "editorial" }));

const sessions = ref<EditorialSession[]>([]);
const loading = ref(true);

async function load() {
    loading.value = true;
    try {
        const res = await api.listEditorialSessions({});
        sessions.value = res.sessions;
    } finally {
        loading.value = false;
    }
}
onMounted(load);

const dateTimeFormatter = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Europe/Oslo",
    dateStyle: "medium",
    timeStyle: "short",
});
function formatDateTime(ts?: Timestamp): string {
    if (!ts) return "";
    return dateTimeFormatter.format(timestampDate(ts));
}

const createOpen = ref(false);
const newVxid = ref("");
const newTitle = ref("");
const creating = ref(false);

async function create() {
    const vxid = newVxid.value.trim();
    if (!vxid) return;
    creating.value = true;
    try {
        const sess = await api.createEditorialSession({
            VXID: vxid,
            title: newTitle.value.trim(),
        });
        createOpen.value = false;
        newVxid.value = "";
        newTitle.value = "";
        await navigateTo(`/editorial/${sess.id}`);
    } catch {
        toaster.create({ title: t("editorial.saveFailed"), type: "error" });
    } finally {
        creating.value = false;
    }
}
</script>

<template>
    <div v-if="!perms.canUseEditorial.value" class="py-16 text-center">
        <p class="text-body-2 text-text-muted">{{ t("noPermissions") }}</p>
    </div>

    <div v-else class="mx-auto w-full max-w-3xl px-4 py-8">
        <div class="mb-6 flex items-center justify-between">
            <h1 class="text-heading-2 text-text-default">
                {{ t("tools.editorial.title") }}
            </h1>
            <DesignButton
                v-if="perms.canEditEditorial.value"
                icon="tabler:plus"
                @click="createOpen = true"
            >
                {{ t("editorial.newSession") }}
            </DesignButton>
        </div>

        <div v-if="loading" class="flex flex-col gap-2">
            <DesignSkeleton v-for="i in 4" :key="i" class="h-16 rounded-2xl" />
        </div>

        <p
            v-else-if="sessions.length === 0"
            class="text-body-3 text-text-hint py-16 text-center"
        >
            {{ t("editorial.noSessions") }}
        </p>

        <ul v-else class="flex flex-col gap-2">
            <li v-for="s in sessions" :key="s.id">
                <NuxtLink
                    :to="`/editorial/${s.id}`"
                    class="bg-surface-raise gradient-border shadow-resting ds-focus-ring hover:bg-surface-indent flex items-center justify-between gap-4 rounded-2xl px-4 py-3 transition-colors"
                >
                    <div class="min-w-0">
                        <p class="text-title-2 text-text-default truncate">
                            {{ s.title || s.VXID }}
                        </p>
                        <p class="text-caption-1 text-text-hint truncate">
                            {{ s.VXID }} · {{ s.createdBy }}
                            <template v-if="s.updatedAt">
                                ·
                                {{
                                    t("editorial.lastUpdated", {
                                        date: formatDateTime(s.updatedAt),
                                    })
                                }}
                            </template>
                        </p>
                    </div>
                </NuxtLink>
            </li>
        </ul>
    </div>

    <DesignDialog v-model:open="createOpen" :title="t('editorial.createTitle')">
        <template #default="{ initialFocus }">
            <form class="flex flex-col gap-4" @submit.prevent="create">
                <DesignInput
                    :ref="initialFocus"
                    v-model="newVxid"
                    :label="t('editorial.vxidLabel')"
                    placeholder="VX-123456"
                />
                <DesignInput
                    v-model="newTitle"
                    :label="t('editorial.titleLabel')"
                />
                <div class="flex justify-end gap-2">
                    <DesignButton
                        variant="secondary"
                        type="button"
                        @click="createOpen = false"
                    >
                        {{ t("editorial.cancel") }}
                    </DesignButton>
                    <DesignButton
                        type="submit"
                        :loading="creating"
                        :disabled="!newVxid.trim()"
                    >
                        {{ t("editorial.create") }}
                    </DesignButton>
                </div>
            </form>
        </template>
    </DesignDialog>
</template>
