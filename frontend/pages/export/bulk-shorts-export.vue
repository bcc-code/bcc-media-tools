<script setup lang="ts">
import { z } from "zod";
import type { FormSubmitEvent } from "@nuxt/ui";

const schema = z.object({
    vxId: z
        .string({ error: "VXID is required" })
        .regex(/VX-[0-9]{6}/, { error: "Needs to be a valid VXID" }),
});

type Schema = z.infer<typeof schema>;

const form = reactive<Partial<Schema>>({
    vxId: undefined,
});

const toast = useToast();
async function onSubmit(event: FormSubmitEvent<Schema>) {
    toast.add({
        icon: "heroicons:check",
        title: "Export triggered",
        description: `Triggered bulk shorts export for ${event.data.vxId}`,
        color: "success",
    });
    navigateTo(`/export/`);
}
</script>

<template>
    <div class="mx-auto w-full max-w-2xl p-4">
        <h1 class="my-8 text-2xl font-bold">Bulk Shorts Export</h1>
        <UForm
            :state="form"
            :schema="schema"
            class="bg-default border-default space-y-2 rounded-2xl border p-4"
            @submit="onSubmit"
        >
            <UFormField label="Collection VXID" name="vxId">
                <UInput v-model="form.vxId" placeholder="VX-123456" />
            </UFormField>
            <UButton type="submit">Trigger Export</UButton>
        </UForm>
    </div>
</template>
