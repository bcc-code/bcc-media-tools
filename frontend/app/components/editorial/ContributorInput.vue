<script setup lang="ts">
// Contributors as chips: typing suggests people from BCC Members, and a pick
// keeps their personID. Enter or a comma (or leaving the field) adds what was
// typed as a name of its own, linked to no one.
import { Combobox, createListCollection } from "@ark-ui/vue";
import type { Member } from "~~/src/gen/api/v1/api_pb";

export interface Contributor {
    name: string;
    // BCC Members personID; empty when typed in by hand
    personId: string;
}

const props = defineProps<{
    placeholder?: string;
    // 16px text, below which iOS zooms the page on focus
    large?: boolean;
}>();

const model = defineModel<Contributor[]>({ required: true });
// After each add or remove, for callers that save at once.
const emit = defineEmits<{ change: [] }>();

const { t } = useI18n();
const api = useAPI();

const query = ref("");
const results = ref<Member[]>([]);
const open = ref(false);
const highlighted = ref<string>();
const focused = ref(false);

const collection = computed(() =>
    createListCollection({
        items: results.value,
        itemToString: (m) => m.displayName,
        itemToValue: (m) => m.personId,
    }),
);

// Answers can arrive out of order; only the latest counts.
let searchSeq = 0;
watchDebounced(
    query,
    async (q) => {
        const seq = ++searchSeq;
        if (q.trim().length < 2) {
            results.value = [];
            return;
        }
        try {
            const res = await api.searchMembers({ query: q.trim(), limit: 8 });
            if (seq !== searchSeq) return;
            const linked = new Set(model.value.map((c) => c.personId));
            results.value = res.members.filter((m) => !linked.has(m.personId));
        } catch {
            if (seq === searchSeq) results.value = [];
        }
    },
    { debounce: 250 },
);

watch([results, focused], () => {
    open.value = focused.value && results.value.length > 0;
});

function update(next: Contributor[]) {
    model.value = next;
    emit("change");
}

function onSelect(details: { items: Member[] }) {
    const m = details.items[0];
    if (!m) return;
    query.value = "";
    results.value = [];
    update([...model.value, { name: m.displayName, personId: m.personId }]);
}

// Adds what was typed; a pasted "A, B" adds both.
function addTyped() {
    const have = new Set(model.value.map((c) => c.name.toLowerCase()));
    const names = query.value
        .split(",")
        .map((n) => n.trim())
        .filter((n) => n && !have.has(n.toLowerCase()));
    query.value = "";
    results.value = [];
    if (names.length)
        update([
            ...model.value,
            ...names.map((name) => ({ name, personId: "" })),
        ]);
}

function remove(i: number) {
    update(model.value.filter((_, j) => j !== i));
}

function onKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && highlighted.value == null) {
        e.preventDefault();
        addTyped();
    } else if (e.key === ",") {
        e.preventDefault();
        addTyped();
    } else if (
        e.key === "Backspace" &&
        query.value === "" &&
        model.value.length
    ) {
        remove(model.value.length - 1);
    }
}

function onBlur() {
    focused.value = false;
    addTyped();
}
</script>

<template>
    <Combobox.Root
        v-model:input-value="query"
        v-model:open="open"
        v-model:highlighted-value="highlighted"
        :collection="collection"
        :model-value="[]"
        allow-custom-value
        input-behavior="none"
        selection-behavior="clear"
        :open-on-click="false"
        :positioning="{ placement: 'bottom-start', gutter: 4 }"
        @value-change="onSelect"
    >
        <Combobox.Control
            :class="[
                'border-border-1 flex w-full flex-wrap items-center gap-1 rounded-xl border px-2 focus-within:outline-3 focus-within:outline-offset-2 focus-within:outline-(--color-focus-ring)',
                large ? 'bg-surface-indent py-2' : 'py-1.5',
            ]"
        >
            <span
                v-for="(c, i) in model"
                :key="`${i}-${c.name}`"
                class="bg-surface-indent text-text-default inline-flex max-w-full items-center gap-1 rounded-lg py-0.5 pr-0.5 pl-2"
                :class="large ? 'text-body-2' : 'text-body-3'"
            >
                <DesignTooltip
                    v-if="c.personId"
                    :content="
                        t('editorial.contributor.linked', { id: c.personId })
                    "
                >
                    <Icon
                        name="tabler:user-check"
                        class="text-text-muted size-3.5 shrink-0"
                        :aria-label="
                            t('editorial.contributor.linked', {
                                id: c.personId,
                            })
                        "
                    />
                </DesignTooltip>
                <span class="truncate">{{ c.name }}</span>
                <button
                    type="button"
                    class="text-text-hint hover:text-text-default ds-focus-ring rounded p-0.5"
                    :aria-label="
                        t('editorial.contributor.remove', { name: c.name })
                    "
                    @click="remove(i)"
                >
                    <Icon name="tabler:x" class="size-3" />
                </button>
            </span>
            <Combobox.Input
                :placeholder="model.length ? '' : props.placeholder"
                :class="[
                    'text-text-default placeholder:text-text-hint min-w-24 flex-1 bg-transparent px-1 outline-none',
                    large ? 'text-body-2' : 'text-body-3',
                ]"
                @focus="focused = true"
                @blur="onBlur"
                @keydown="onKeydown"
            />
        </Combobox.Control>

        <Teleport to="#teleports">
            <Combobox.Positioner class="z-60">
                <Combobox.Content
                    class="gradient-border bg-surface-raise shadow-floating min-w-(--reference-width) rounded-xl p-1"
                >
                    <Combobox.Item
                        v-for="m in results"
                        :key="m.personId"
                        :item="m"
                        class="text-body-3 text-text-default data-highlighted:bg-surface-indent flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2"
                    >
                        <Icon
                            name="tabler:user"
                            class="text-text-muted size-4 shrink-0"
                        />
                        <Combobox.ItemText>{{
                            m.displayName
                        }}</Combobox.ItemText>
                    </Combobox.Item>
                    <p class="text-caption-1 text-text-hint px-3 pt-1 pb-1.5">
                        {{ t("editorial.contributor.enterToAdd") }}
                    </p>
                </Combobox.Content>
            </Combobox.Positioner>
        </Teleport>
    </Combobox.Root>
</template>
