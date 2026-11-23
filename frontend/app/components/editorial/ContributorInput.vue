<script setup lang="ts">
// Contributors as chips: typing suggests people from BCC Members, those
// already linked first, and a pick keeps their member uid. An empty field offers
// the frequent contributors. Without the members permission there are no
// suggestions. Enter or a comma (or leaving the field) adds what was typed as
// a name of its own, linked to no one.
import { Combobox, createListCollection, useCombobox } from "@ark-ui/vue";
import type { Member } from "~~/src/gen/api/v1/api_pb";

export interface Contributor {
    name: string;
    // BCC Members uid (a UUID); empty when typed in by hand
    memberUid: string;
    // as the server tells, for linked people; age and church only with the
    // members permission, age 0 when unknown
    u26?: boolean;
    age?: number;
    church?: string;
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
const { canLookUpMembers } = usePermissions();

const input = useTemplateRef<ComponentPublicInstance>("input");
const query = ref("");
const results = ref<Member[]>([]);
const focused = ref(false);

const collection = computed(() =>
    createListCollection({
        items: results.value,
        itemToString: (m) => m.displayName,
        itemToValue: (m) => m.memberUid,
    }),
);

// Answers can arrive out of order; only the latest counts.
let searchSeq = 0;
async function search(q: string) {
    if (!canLookUpMembers.value) return;
    const seq = ++searchSeq;
    try {
        // an empty query lists the frequent contributors
        const res = await api.searchMembers({
            query: q.trim(),
            limit: q.trim() ? 8 : 20,
        });
        if (seq !== searchSeq) return;
        const linked = new Set(model.value.map((c) => c.memberUid));
        results.value = res.members.filter((m) => !linked.has(m.memberUid));
    } catch {
        if (seq === searchSeq) results.value = [];
    }
}
watchDebounced(query, search, { debounce: 250 });

const typing = computed(() => query.value.trim() !== "");

function onFocus() {
    focused.value = true;
    if (!typing.value) void search("");
}

// The list opens and closes on its own; open isn't bound, as closing a bound
// one puts the focus back in the input. Only suggestions arriving while the
// field has focus open it from here.
const combobox = useCombobox(
    computed(() => ({
        collection: collection.value,
        allowCustomValue: true,
        inputBehavior: "none" as const,
        selectionBehavior: "clear" as const,
        openOnClick: false,
        positioning: { placement: "bottom-start" as const, gutter: 4 },
        value: [],
        inputValue: query.value,
        onInputValueChange: (d: { inputValue: string }) => {
            query.value = d.inputValue;
        },
        onValueChange: onSelect,
    })),
);
watch(results, (found) => {
    if (!focused.value) return;
    if (found.length && !combobox.value.open) combobox.value.setOpen(true);
    else if (!found.length && combobox.value.open)
        combobox.value.setOpen(false);
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
    update([
        ...model.value,
        {
            name: m.displayName,
            memberUid: m.memberUid,
            u26: isU26(m.age),
            age: m.age,
            church: m.church,
        },
    ]);
    // Leave the field; a click opens the list again. A frame later, as the
    // combobox first puts focus back in the input.
    requestAnimationFrame(() => input.value?.$el?.blur());
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
            ...names.map((name) => ({ name, memberUid: "" })),
        ]);
}

function remove(i: number) {
    update(model.value.filter((_, j) => j !== i));
}

function onKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" && combobox.value.highlightedValue == null) {
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
    <Combobox.RootProvider :value="combobox">
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
                    v-if="c.memberUid"
                    :content="
                        [t('editorial.contributor.linked'), memberDetails(c)]
                            .filter(Boolean)
                            .join(' · ')
                    "
                >
                    <Icon
                        name="tabler:user-check"
                        class="text-text-muted size-3.5 shrink-0"
                        :aria-label="t('editorial.contributor.linked')"
                    />
                </DesignTooltip>
                <!-- the name and U26 share a baseline; together they're
                     centred like the icons -->
                <span class="flex min-w-0 items-baseline gap-1">
                    <span class="truncate">{{ c.name }}</span>
                    <EditorialU26Badge v-if="c.u26" class="ml-0.5" />
                </span>
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
                    'text-text-default placeholder:text-text-hint w-0 min-w-6 flex-1 bg-transparent px-1 outline-none focus:min-w-24',
                    large ? 'text-body-2' : 'text-body-3',
                ]"
                ref="input"
                @focus="onFocus"
                @blur="onBlur"
                @keydown="onKeydown"
            />
        </Combobox.Control>

        <!-- without the members permission it's a plain text field -->
        <Teleport v-if="canLookUpMembers" to="#teleports">
            <Combobox.Positioner class="z-60">
                <Combobox.Content
                    class="gradient-border bg-surface-raise shadow-floating w-max max-w-[min(24rem,calc(100vw-2rem))] min-w-(--reference-width) rounded-xl p-1"
                >
                    <div class="max-h-80 overflow-y-auto">
                        <Combobox.Item
                            v-for="m in results"
                            :key="m.memberUid"
                            :item="m"
                            class="text-body-3 text-text-default data-highlighted:bg-surface-indent flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2"
                        >
                            <Icon
                                name="tabler:user"
                                class="text-text-muted size-4 shrink-0"
                            />
                            <Combobox.ItemText class="self-baseline truncate">{{
                                m.displayName
                            }}</Combobox.ItemText>
                            <!-- church and age tell people of one name apart -->
                            <span
                                class="text-caption-1 text-text-muted ml-auto self-baseline pl-2 whitespace-nowrap"
                                >{{ memberDetails(m) }}</span
                            >
                            <EditorialU26Badge
                                v-if="isU26(m.age)"
                                class="self-baseline"
                            />
                        </Combobox.Item>
                    </div>
                    <!-- one line in one place, so typing doesn't shift the list -->
                    <p
                        class="text-caption-1 text-text-hint px-3 pt-1 pb-1.5 contain-inline-size"
                    >
                        {{
                            t(
                                typing
                                    ? "editorial.contributor.enterToAdd"
                                    : "editorial.contributor.frequent",
                            )
                        }}
                    </p>
                </Combobox.Content>
            </Combobox.Positioner>
        </Teleport>
    </Combobox.RootProvider>
</template>
