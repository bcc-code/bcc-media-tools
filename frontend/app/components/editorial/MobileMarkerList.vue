<script setup lang="ts">
// The phone layout of a review: one card per marker with its comment field
// right on it, and everything else behind a ··· drawer. No video: on a phone
// the job is commenting while the meeting runs. Fields use 16px text, below
// which iOS zooms the page on focus.

export interface MobileMarker {
    key: string;
    time: string;
    duration: string;
    type: string;
    typeLabel: string;
    name: string;
    contributors: string;
    bibleVerses: string;
    comment: string;
    publishBmm: boolean;
    publishBcc: boolean;
}

export interface MarkerDetails {
    name: string;
    contributors: string;
    type: string;
}

const props = defineProps<{
    markers: MobileMarker[];
    canEdit: boolean;
    typeItems: { label: string; value: string }[];
}>();

const emit = defineEmits<{
    comment: [key: string, comment: string];
    details: [key: string, details: MarkerDetails];
    publish: [key: string, target: "bmm" | "bcc", value: boolean];
}>();

const { t } = useI18n();

// Inserted into the comment text itself, so they stay Norwegian whatever the
// UI language: saved comments read the same for everyone.
const CANNED_COMMENTS = [
    "Sjekk navn",
    "Sjekk alder",
    "Ikke publiser",
    "Verifiser innhold",
];

// Comments being typed; saved when the field loses focus.
const drafts = reactive<Record<string, string>>({});

function draftOf(m: MobileMarker): string {
    return drafts[m.key] ?? m.comment;
}

function commit(m: MobileMarker) {
    const text = drafts[m.key];
    delete drafts[m.key];
    if (text !== undefined && text !== m.comment) emit("comment", m.key, text);
}

function addCanned(m: MobileMarker, phrase: string) {
    // Read the latest: a blur just before this click may have saved a draft.
    const latest = props.markers.find((x) => x.key === m.key) ?? m;
    const current = draftOf(latest).trim();
    delete drafts[m.key];
    emit("comment", m.key, current ? `${current} · ${phrase}` : phrase);
}

function headline(m: MobileMarker): string {
    return m.name || m.contributors || t("editorial.waitingForPlayout");
}

function subline(m: MobileMarker): string {
    return [m.name ? m.contributors : "", m.bibleVerses]
        .filter(Boolean)
        .join(" · ");
}

// ── Edit drawer ──────────────────────────────────────────
const editing = ref<MobileMarker>();
const editName = ref("");
const editContributors = ref("");
const editType = ref("");

const drawerOpen = computed({
    get: () => !!editing.value,
    set: (open: boolean) => {
        if (!open) closeDrawer();
    },
});

function openDrawer(m: MobileMarker) {
    editing.value = m;
    editName.value = m.name;
    editContributors.value = m.contributors;
    editType.value = m.type;
}

// Details save together when the drawer closes, however it was closed.
function closeDrawer() {
    const m = editing.value;
    editing.value = undefined;
    if (!m || !props.canEdit) return;
    if (
        editName.value !== m.name ||
        editContributors.value !== m.contributors ||
        editType.value !== m.type
    ) {
        emit("details", m.key, {
            name: editName.value,
            contributors: editContributors.value,
            type: editType.value,
        });
    }
}

// The drawer shows the marker as it is now, so publish switches reflect a
// toggle at once.
const current = computed(() =>
    props.markers.find((m) => m.key === editing.value?.key),
);
</script>

<template>
    <div class="flex flex-col gap-2.5">
        <article
            v-for="m in markers"
            :key="m.key"
            class="bg-surface-raise gradient-border shadow-resting flex flex-col gap-2.5 rounded-2xl p-3.5"
        >
            <div class="flex items-center gap-2.5">
                <span class="text-title-3 text-text-default tabular-nums">
                    {{ m.time }}
                </span>
                <span class="text-caption-1 text-text-hint tabular-nums">
                    {{ m.duration }}
                </span>
                <span class="grow" />
                <DesignBadge v-if="m.typeLabel">{{ m.typeLabel }}</DesignBadge>
                <button
                    type="button"
                    class="text-text-muted hover:bg-surface-indent ds-focus-ring -my-2 -mr-2 flex h-9 w-11 items-center justify-center rounded-2xl"
                    :aria-label="
                        t('editorial.moreOptionsFor', { name: headline(m) })
                    "
                    @click="openDrawer(m)"
                >
                    <Icon name="tabler:dots" class="size-5" />
                </button>
            </div>

            <div class="flex flex-col gap-0.5">
                <p class="text-title-2 text-text-default">{{ headline(m) }}</p>
                <p v-if="subline(m)" class="text-body-3 text-text-muted">
                    {{ subline(m) }}
                </p>
            </div>

            <textarea
                :value="draftOf(m)"
                rows="2"
                :placeholder="t('editorial.addComment')"
                :aria-label="t('editorial.commentOn', { name: headline(m) })"
                class="border-border-1 bg-surface-indent text-body-2 text-text-default placeholder:text-text-hint ds-focus-ring min-h-12 w-full resize-none rounded-xl border px-3 py-2.5"
                @input="
                    drafts[m.key] = ($event.target as HTMLTextAreaElement).value
                "
                @blur="commit(m)"
            />
            <div class="flex flex-wrap gap-1.5">
                <button
                    v-for="phrase in CANNED_COMMENTS"
                    :key="phrase"
                    type="button"
                    class="border-border-1 text-caption-1 text-text-muted hover:bg-surface-indent ds-focus-ring min-h-9 rounded-full border px-3"
                    @click="addCanned(m, phrase)"
                >
                    + {{ phrase }}
                </button>
            </div>
        </article>

        <DesignSheet
            v-model:open="drawerOpen"
            :title="editing ? headline(editing) : undefined"
            :description="
                editing
                    ? [editing.time, editing.duration].join(' · ')
                    : undefined
            "
        >
            <div v-if="current" class="flex flex-col gap-4">
                <template v-if="canEdit">
                    <label class="flex flex-col gap-1.5">
                        <span class="text-caption-1 text-text-muted">
                            {{ t("editorial.col.title") }}
                        </span>
                        <input
                            v-model="editName"
                            type="text"
                            class="border-border-1 bg-surface-indent text-body-2 text-text-default ds-focus-ring rounded-xl border px-3 py-2.5"
                        />
                    </label>
                    <label class="flex flex-col gap-1.5">
                        <span class="text-caption-1 text-text-muted">
                            {{ t("editorial.col.contributors") }}
                        </span>
                        <input
                            v-model="editContributors"
                            type="text"
                            class="border-border-1 bg-surface-indent text-body-2 text-text-default ds-focus-ring rounded-xl border px-3 py-2.5"
                        />
                    </label>
                    <div class="flex flex-col gap-1.5">
                        <span class="text-caption-1 text-text-muted">
                            {{ t("editorial.col.type") }}
                        </span>
                        <div
                            role="group"
                            :aria-label="t('editorial.col.type')"
                            class="flex flex-wrap gap-1.5"
                        >
                            <button
                                v-for="item in typeItems"
                                :key="item.value"
                                type="button"
                                :aria-pressed="editType === item.value"
                                class="text-body-3 ds-focus-ring min-h-10 rounded-full border px-3.5"
                                :class="
                                    editType === item.value
                                        ? 'bg-primary-default text-on-primary border-transparent'
                                        : 'border-border-1 text-text-default'
                                "
                                @click="editType = item.value"
                            >
                                {{ item.label }}
                            </button>
                        </div>
                    </div>
                </template>

                <div
                    class="border-border-1 divide-border-1 flex flex-col divide-y rounded-2xl border"
                >
                    <div class="flex min-h-13 items-center px-3.5">
                        <DesignSwitch
                            :model-value="current.publishBmm"
                            :label="t('editorial.publishTo', { target: 'BMM' })"
                            class="w-full"
                            @update:model-value="
                                emit('publish', current.key, 'bmm', $event)
                            "
                        />
                    </div>
                    <div class="flex min-h-13 items-center px-3.5">
                        <DesignSwitch
                            :model-value="current.publishBcc"
                            :label="
                                t('editorial.publishTo', {
                                    target: 'BCC Media',
                                })
                            "
                            class="w-full"
                            @update:model-value="
                                emit('publish', current.key, 'bcc', $event)
                            "
                        />
                    </div>
                </div>

                <DesignButton size="large" @click="closeDrawer">
                    {{ t("editorial.done") }}
                </DesignButton>
            </div>
        </DesignSheet>
    </div>
</template>
