<script setup lang="ts">
import { z } from "zod";
import type { FormSubmitEvent } from "@nuxt/ui";

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
    vxId: z.string().regex(/^VX-[a-zA-Z0-9]+$/),
});

type Schema = z.output<typeof schema>;

const state = reactive<Partial<Schema>>({
    vxId: undefined,
});

function onSubmit(event: FormSubmitEvent<Schema>) {
    navigateTo({
        name: "shorts-generate",
        query: {
            id: event.data.vxId,
        },
    });
}
</script>

<template>
    <div class="flex justify-center py-8">
        <UForm
            :state
            :schema
            @submit="onSubmit"
            class="flex w-full max-w-xs flex-col gap-2"
        >
            <UFormField label="VX ID" name="vxId" size="lg">
                <UInput
                    v-model="state.vxId"
                    class="w-full"
                    placeholder="VX-123456"
                />
            </UFormField>
            <UButton type="submit" block>Submit</UButton>
        </UForm>
    </div>
</template>
