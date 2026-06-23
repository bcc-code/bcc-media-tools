<script setup lang="ts">
import { Pagination } from "@ark-ui/vue";

interface Props {
    total: number;
    pageSize?: number;
    siblingCount?: number;
    showEdges?: boolean;
    disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
    pageSize: 50,
    siblingCount: 1,
    showEdges: false,
});

const page = defineModel<number>("page", { default: 1 });

const triggerClass =
    "ds-focus-ring text-text-default hover:bg-surface-indent flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-lg disabled:cursor-not-allowed disabled:opacity-40";
</script>

<template>
    <Pagination.Root
        v-model:page="page"
        :count="total"
        :page-size="pageSize"
        :sibling-count="siblingCount"
        class="flex items-center gap-1"
    >
        <Pagination.FirstTrigger
            v-if="showEdges"
            :disabled="disabled"
            :class="triggerClass"
        >
            <Icon name="tabler:chevrons-left" class="size-4" />
        </Pagination.FirstTrigger>
        <Pagination.PrevTrigger :disabled="disabled" :class="triggerClass">
            <Icon name="tabler:chevron-left" class="size-4" />
        </Pagination.PrevTrigger>

        <Pagination.Context v-slot="{ pages }">
            <template v-for="(p, index) in pages">
                <Pagination.Item
                    v-if="p.type === 'page'"
                    :key="`p${p.value}`"
                    :value="p.value"
                    :type="p.type"
                    :disabled="disabled"
                    class="ds-focus-ring text-text-default hover:bg-surface-indent data-selected:bg-primary-default data-selected:text-on-primary text-title-3 flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-lg tabular-nums"
                >
                    {{ p.value }}
                </Pagination.Item>
                <Pagination.Ellipsis
                    v-else
                    :key="`e${index}`"
                    :index="index"
                    class="text-text-hint flex size-9 items-center justify-center"
                >
                    …
                </Pagination.Ellipsis>
            </template>
        </Pagination.Context>

        <Pagination.NextTrigger :disabled="disabled" :class="triggerClass">
            <Icon name="tabler:chevron-right" class="size-4" />
        </Pagination.NextTrigger>
        <Pagination.LastTrigger
            v-if="showEdges"
            :disabled="disabled"
            :class="triggerClass"
        >
            <Icon name="tabler:chevrons-right" class="size-4" />
        </Pagination.LastTrigger>
    </Pagination.Root>
</template>
