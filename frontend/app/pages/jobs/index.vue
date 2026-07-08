<script lang="ts" setup>
import { formatTimeAgo } from "@vueuse/core";
import type { Job } from "~~/src/gen/api/v1/api_pb";

const api = useAPI();
const { t, locale } = useI18n();
const { me } = useMe();
// Only total admins may see who started a job; everyone else gets the column
// hidden. Non-admins still receive their own email in the payload (the backend
// redacts others'), so the "Only my jobs" filter keeps working.
const { admin } = usePermissions();
const toaster = useDesignToaster();

function copyText(text: string) {
    navigator.clipboard?.writeText(text);
    toaster.create({ title: t("jobs.copied"), type: "success" });
}

useHead({ title: "Jobs" });

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({ id: "jobs", title: "jobs" });
});

const STATUSES = [
    "running",
    "completed",
    "failed",
    "canceled",
    "terminated",
    "timed_out",
] as const;

const TOOLS = [
    "export",
    "vb_export",
    "timed_metadata",
    "shorts",
    "bmm_upload",
    "subtitles",
    "transcribe",
] as const;

const STATUS_VARIANT: Record<string, string> = {
    running: "info",
    completed: "success",
    failed: "error",
    canceled: "warning",
    terminated: "warning",
    timed_out: "warning",
    continued_as_new: "neutral",
};

const PAGE_SIZE = 25;

const selectedStatuses = useQueryRef<string[]>("status", []);
const selectedTools = useQueryRef<string[]>("tool", []);
const reference = useQueryRef("ref", "");
const mineOnly = useQueryRef("mine", false);

const jobs = ref<Job[]>([]);
const nextToken = ref<Uint8Array>(new Uint8Array());
const loading = ref(false);
const loaded = ref(false);
// Auto-refresh reloads the latest page every few seconds; loading further pages
// pauses it so appended rows aren't discarded on the next tick.
const autoRefresh = ref(true);

const hasMore = computed(() => nextToken.value.length > 0);

async function load(reset: boolean) {
    loading.value = true;
    try {
        const res = await api.listJobs({
            statuses: selectedStatuses.value,
            tools: selectedTools.value,
            pageSize: PAGE_SIZE,
            pageToken: reset ? new Uint8Array() : nextToken.value,
        });
        jobs.value = reset ? res.jobs : [...jobs.value, ...res.jobs];
        nextToken.value = res.nextPageToken ?? new Uint8Array();
    } finally {
        loading.value = false;
        loaded.value = true;
    }
}

function refresh() {
    autoRefresh.value = true;
    load(true);
}
function loadMore() {
    autoRefresh.value = false;
    load(false);
}

watch([selectedStatuses, selectedTools], () => refresh());

const detailOpen = ref(false);

// Poll only while the tab is visible and auto-refresh is on, so a backgrounded
// tab doesn't keep hitting Temporal. Refresh once immediately on return.
function pollTick() {
    if (autoRefresh.value && !detailOpen.value && !document.hidden) load(true);
}
function onVisibility() {
    if (!document.hidden) pollTick();
}

let timer: ReturnType<typeof setInterval> | undefined;
onMounted(() => {
    load(true);
    timer = setInterval(pollTick, 5000);
    document.addEventListener("visibilitychange", onVisibility);
});
onBeforeUnmount(() => {
    if (timer) clearInterval(timer);
    document.removeEventListener("visibilitychange", onVisibility);
});

// Reference and "mine only" filter client-side: both live in the workflow Memo,
// which Temporal can't filter on server-side.
const visibleJobs = computed(() => {
    let list = jobs.value;
    const ref = reference.value.trim().toLowerCase();
    if (ref) {
        list = list.filter((j) => j.reference.toLowerCase().includes(ref));
    }
    if (mineOnly.value) {
        const email = me.value?.email;
        if (email) list = list.filter((j) => j.startedBy === email);
    }
    return list;
});

const hasFilter = computed(
    () =>
        selectedStatuses.value.length > 0 ||
        selectedTools.value.length > 0 ||
        reference.value !== "" ||
        mineOnly.value,
);
function clearFilters() {
    selectedStatuses.value = [];
    selectedTools.value = [];
    reference.value = "";
    mineOnly.value = false;
}

function toggleStatus(value: string, checked: boolean) {
    selectedStatuses.value = checked
        ? [...selectedStatuses.value, value]
        : selectedStatuses.value.filter((v) => v !== value);
}
function toggleTool(value: string, checked: boolean) {
    selectedTools.value = checked
        ? [...selectedTools.value, value]
        : selectedTools.value.filter((v) => v !== value);
}

const dateFmt = computed(
    () =>
        new Intl.DateTimeFormat(locale.value, {
            dateStyle: "medium",
            timeStyle: "short",
        }),
);
function formatStarted(job: Job): string {
    const d = timestampToDate(job.startedAt);
    return d ? dateFmt.value.format(d) : "—";
}

function formatRelative(job: Job): string {
    const d = timestampToDate(job.startedAt);
    return d ? formatTimeAgo(d) : "—";
}

function formatDuration(job: Job): string {
    const start = timestampToDate(job.startedAt);
    if (!start) return "—";
    const end = timestampToDate(job.closedAt) ?? new Date();
    const sec = Math.max(0, (end.getTime() - start.getTime()) / 1000);
    if (sec < 60) return `${Math.round(sec)}s`;
    const m = Math.floor(sec / 60);
    const s = Math.round(sec % 60);
    if (m < 60) return s ? `${m}m ${s}s` : `${m}m`;
    const h = Math.floor(m / 60);
    return m % 60 ? `${h}h ${m % 60}m` : `${h}h`;
}

const detail = reactive<{
    job: Job | null;
    error: string;
    temporalUrl: string;
    loading: boolean;
}>({
    job: null,
    error: "",
    temporalUrl: "",
    loading: false,
});

async function openDetail(job: Job) {
    detail.job = job;
    detail.error = "";
    detail.temporalUrl = "";
    detail.loading = true;
    detailOpen.value = true;
    try {
        const res = await api.getJob({
            workflowId: job.workflowId,
            runId: job.runId,
        });
        if (res.job) detail.job = res.job;
        detail.error = res.errorMessage;
        detail.temporalUrl = res.temporalUrl;
    } finally {
        detail.loading = false;
    }
}
</script>

<template>
    <div class="flex h-full flex-col">
        <div
            class="border-border-1 flex items-center justify-between gap-4 border-b px-6 py-5"
        >
            <div class="flex items-center gap-3">
                <h1 class="text-title-1 text-text-default font-semibold">
                    {{ t("jobs.title") }}
                </h1>
                <Icon
                    v-if="loading"
                    name="svg-spinners:ring-resize"
                    class="text-text-hint size-4"
                />
            </div>

            <div class="flex items-center gap-3">
                <span
                    class="text-text-hint flex items-center gap-1.5 text-xs whitespace-nowrap"
                >
                    <span
                        class="size-1.5 rounded-full"
                        :class="
                            autoRefresh ? 'bg-semantic-success' : 'bg-text-hint'
                        "
                    />
                    {{ autoRefresh ? t("jobs.autoRefresh") : t("jobs.paused") }}
                </span>
                <DesignInput
                    v-model="reference"
                    leading-icon="tabler:search"
                    :placeholder="t('jobs.referencePlaceholder')"
                    class="w-56"
                />
                <DesignButton
                    variant="secondary"
                    icon="tabler:refresh"
                    @click="refresh"
                >
                    {{ t("jobs.refresh") }}
                </DesignButton>
            </div>
        </div>

        <div class="flex grow items-start">
            <aside
                class="border-border-1 w-64 shrink-0 self-stretch border-r p-6"
            >
                <div class="mb-5 flex items-center justify-between">
                    <h2 class="text-title-1 text-text-default font-semibold">
                        {{ t("jobs.filters") }}
                    </h2>
                    <button
                        v-if="hasFilter"
                        class="text-text-muted hover:text-text-default text-xs"
                        @click="clearFilters"
                    >
                        {{ t("jobs.clear") }}
                    </button>
                </div>

                <DesignSwitch v-model="mineOnly" :label="t('jobs.mineOnly')" />

                <fieldset class="mt-6">
                    <legend
                        class="text-text-muted mb-3 text-[11px] font-semibold tracking-wide uppercase"
                    >
                        {{ t("jobs.status") }}
                    </legend>
                    <div class="flex flex-col gap-y-2">
                        <DesignCheckbox
                            v-for="s in STATUSES"
                            :key="s"
                            :model-value="selectedStatuses.includes(s)"
                            :label="t(`jobs.statusLabel.${s}`)"
                            @update:model-value="toggleStatus(s, $event)"
                        />
                    </div>
                </fieldset>

                <fieldset class="mt-6">
                    <legend
                        class="text-text-muted mb-3 text-[11px] font-semibold tracking-wide uppercase"
                    >
                        {{ t("jobs.tool") }}
                    </legend>
                    <div class="flex flex-col gap-y-2">
                        <DesignCheckbox
                            v-for="tool in TOOLS"
                            :key="tool"
                            :model-value="selectedTools.includes(tool)"
                            :label="t(`jobs.toolLabel.${tool}`)"
                            @update:model-value="toggleTool(tool, $event)"
                        />
                    </div>
                </fieldset>
            </aside>

            <main class="min-w-0 flex-1 p-6">
                <div v-if="loading && !jobs.length" class="flex flex-col gap-2">
                    <DesignSkeleton
                        v-for="n in 8"
                        :key="n"
                        class="h-12 w-full"
                    />
                </div>

                <div v-else-if="visibleJobs.length" class="overflow-x-auto">
                    <table class="w-full text-left text-sm">
                        <thead
                            class="text-text-muted border-border-1 border-b text-xs"
                        >
                            <tr>
                                <th class="py-2 pr-4 font-semibold">
                                    {{ t("jobs.tool") }}
                                </th>
                                <th class="py-2 pr-4 font-semibold">
                                    {{ t("jobs.reference") }}
                                </th>
                                <th
                                    v-if="admin"
                                    class="py-2 pr-4 font-semibold"
                                >
                                    {{ t("jobs.startedBy") }}
                                </th>
                                <th class="py-2 pr-4 font-semibold">
                                    {{ t("jobs.status") }}
                                </th>
                                <th class="py-2 pr-4 font-semibold">
                                    {{ t("jobs.started") }}
                                </th>
                                <th class="py-2 font-semibold">
                                    {{ t("jobs.duration") }}
                                </th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr
                                v-for="job in visibleJobs"
                                :key="job.workflowId + job.runId"
                                class="border-border-1 hover:bg-surface-indent cursor-pointer border-b"
                                @click="openDetail(job)"
                            >
                                <td class="text-text-default py-2.5 pr-4">
                                    {{ t(`jobs.toolLabel.${job.tool}`) }}
                                </td>
                                <td
                                    class="text-text-muted py-2.5 pr-4 font-mono text-xs"
                                >
                                    {{ job.reference || "—" }}
                                </td>
                                <td
                                    v-if="admin"
                                    class="text-text-muted py-2.5 pr-4"
                                >
                                    {{ job.startedBy || t("jobs.unknownUser") }}
                                </td>
                                <td class="py-2.5 pr-4">
                                    <DesignBadge
                                        :variant="
                                            (STATUS_VARIANT[
                                                job.status
                                            ] as any) ?? 'neutral'
                                        "
                                    >
                                        <Icon
                                            v-if="job.status === 'running'"
                                            name="svg-spinners:bars-rotate-fade"
                                            class="mr-1 size-3"
                                        />
                                        {{
                                            t(`jobs.statusLabel.${job.status}`)
                                        }}
                                    </DesignBadge>
                                </td>
                                <td
                                    class="text-text-muted py-2.5 pr-4 whitespace-nowrap"
                                    :title="formatStarted(job)"
                                >
                                    {{ formatRelative(job) }}
                                </td>
                                <td
                                    class="text-text-muted py-2.5 whitespace-nowrap"
                                >
                                    {{ formatDuration(job) }}
                                </td>
                            </tr>
                        </tbody>
                    </table>

                    <div v-if="hasMore" class="mt-6 flex justify-center">
                        <DesignButton
                            variant="secondary"
                            :disabled="loading"
                            @click="loadMore"
                        >
                            {{ t("jobs.loadMore") }}
                        </DesignButton>
                    </div>
                </div>

                <div
                    v-else-if="loaded"
                    class="text-text-muted py-20 text-center"
                >
                    <Icon
                        name="tabler:clipboard-list"
                        class="size-10 opacity-50"
                    />
                    <p class="mt-3 text-sm">{{ t("jobs.noResults") }}</p>
                </div>
            </main>
        </div>

        <DesignDialog
            v-model:open="detailOpen"
            :title="detail.job ? t(`jobs.toolLabel.${detail.job.tool}`) : ''"
            size="lg"
        >
            <div v-if="detail.job" class="flex flex-col gap-5">
                <div class="flex flex-wrap items-center gap-2.5">
                    <DesignBadge
                        :variant="
                            (STATUS_VARIANT[detail.job.status] as any) ??
                            'neutral'
                        "
                    >
                        <Icon
                            v-if="detail.job.status === 'running'"
                            name="svg-spinners:bars-rotate-fade"
                            class="mr-1 size-3"
                        />
                        {{ t(`jobs.statusLabel.${detail.job.status}`) }}
                    </DesignBadge>
                    <span
                        v-if="detail.job.reference"
                        class="bg-surface-indent text-text-default rounded-md px-2 py-0.5 font-mono text-sm"
                    >
                        {{ detail.job.reference }}
                    </span>
                    <Icon
                        v-if="detail.loading"
                        name="svg-spinners:bars-rotate-fade"
                        class="text-text-hint size-4"
                    />
                </div>

                <dl
                    class="border-border-1 divide-border-1 divide-y rounded-xl border text-sm"
                >
                    <div
                        v-if="admin"
                        class="flex items-center justify-between gap-4 px-4 py-3"
                    >
                        <dt class="text-text-muted">
                            {{ t("jobs.startedBy") }}
                        </dt>
                        <dd class="text-text-default">
                            {{ detail.job.startedBy || t("jobs.unknownUser") }}
                        </dd>
                    </div>
                    <div
                        class="flex items-center justify-between gap-4 px-4 py-3"
                    >
                        <dt class="text-text-muted">{{ t("jobs.started") }}</dt>
                        <dd class="text-text-default">
                            {{ formatStarted(detail.job) }}
                        </dd>
                    </div>
                    <div
                        class="flex items-center justify-between gap-4 px-4 py-3"
                    >
                        <dt class="text-text-muted">
                            {{ t("jobs.duration") }}
                        </dt>
                        <dd class="text-text-default tabular-nums">
                            {{ formatDuration(detail.job) }}
                        </dd>
                    </div>
                    <div
                        class="flex items-center justify-between gap-4 px-4 py-3"
                    >
                        <dt class="text-text-muted">{{ t("jobs.type") }}</dt>
                        <dd class="text-text-default font-mono text-xs">
                            {{ detail.job.workflowType }}
                        </dd>
                    </div>
                    <div
                        class="flex items-center justify-between gap-4 px-4 py-3"
                    >
                        <dt class="text-text-muted shrink-0">
                            {{ t("jobs.workflowId") }}
                        </dt>
                        <dd class="flex min-w-0 items-center gap-2">
                            <span
                                class="text-text-default truncate font-mono text-xs"
                            >
                                {{ detail.job.workflowId }}
                            </span>
                            <button
                                class="text-text-hint hover:text-text-default ds-focus-ring shrink-0 cursor-pointer rounded p-1"
                                :aria-label="t('jobs.copy')"
                                @click="copyText(detail.job.workflowId)"
                            >
                                <Icon name="tabler:copy" class="size-4" />
                            </button>
                        </dd>
                    </div>
                </dl>

                <div
                    v-if="detail.error"
                    class="border-semantic-error/30 bg-semantic-error/5 rounded-xl border p-3"
                >
                    <div
                        class="text-semantic-error mb-1.5 flex items-center gap-1.5 text-sm font-semibold"
                    >
                        <Icon name="tabler:alert-triangle" class="size-4" />
                        {{ t("jobs.error") }}
                    </div>
                    <pre
                        class="text-text-default max-h-64 overflow-auto text-xs whitespace-pre-wrap"
                        >{{ detail.error }}</pre
                    >
                </div>

                <div
                    v-if="detail.temporalUrl"
                    class="border-border-1 -mx-6 -mb-6 border-t px-6 py-4"
                >
                    <a
                        :href="detail.temporalUrl"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-text-muted hover:text-text-default ds-focus-ring inline-flex items-center gap-1.5 rounded text-sm font-medium"
                    >
                        {{ t("jobs.seeInTemporal") }}
                        <Icon name="tabler:external-link" class="size-4" />
                    </a>
                </div>
            </div>
        </DesignDialog>
    </div>
</template>
