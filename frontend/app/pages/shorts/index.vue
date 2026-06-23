<script setup lang="ts">
import { z } from "zod";

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "shorts_index",
        title: "shorts",
    });
});

useHead({
    title: "Shorts generation",
});

const schema = z.object({
    vxId: z.string().regex(/^VX-[a-zA-Z0-9]+$/, "Must look like VX-123456"),
});

type Schema = z.output<typeof schema>;

const state = reactive<Partial<Schema>>({
    vxId: undefined,
});

const error = ref<string>();

function onSubmit() {
    const result = schema.safeParse(state);
    if (!result.success) {
        error.value = result.error.issues[0]?.message ?? "Invalid";
        return;
    }
    error.value = undefined;
    navigateTo({
        name: "shorts-generate",
        query: {
            id: result.data.vxId,
        },
    });
}
</script>

<template>
    <div class="flex justify-center py-8">
        <form
            class="flex w-full max-w-xs flex-col gap-2"
            @submit.prevent="onSubmit"
        >
            <DesignInput
                v-model="state.vxId"
                label="VX ID"
                placeholder="VX-123456"
                :invalid="!!error"
                :error-text="error"
            />
            <DesignButton type="submit" class="w-full">
                {{ $t("shorts.generation.loadFromMediabanken") }}
            </DesignButton>
        </form>
    </div>
</template>
