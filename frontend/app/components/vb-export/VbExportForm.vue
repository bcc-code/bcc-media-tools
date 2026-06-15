<script setup lang="ts">
import type { GetVBExportConfigResponse } from "~~/src/gen/api/v1/api_pb";
import type { VBExportSelection } from "~/components/vb-export/types";

const props = defineProps<{
    config: GetVBExportConfigResponse;
    submitting?: boolean;
}>();

const emit = defineEmits<{
    (e: "start-export", payload: VBExportSelection): void;
}>();

/* ------------------------------------------------------------------ state --- */

const destChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(props.config.destinations.map((d) => [d, false])),
);

const subtitleShape = ref(props.config.subtitleShapes[0] ?? "None");
const subtitleStyle = ref(props.config.subtitleStyles[0] ?? "");

/* --------------------------------------------------------------- computed --- */

const selectedDestCount = computed(
    () => props.config.destinations.filter((d) => destChecked[d]).length,
);

/* ---------------------------------------------------------------- actions --- */

function startExport() {
    emit("start-export", {
        destinations: props.config.destinations.filter((d) => destChecked[d]),
        subtitleShape: subtitleShape.value,
        subtitleStyle: subtitleStyle.value,
    });
}
</script>

<template>
    <div class="mx-auto w-full max-w-3xl px-6 py-8">
        <!-- Title -->
        <h1 class="text-highlighted mb-6 text-center text-2xl font-semibold tracking-tight">
            <span class="font-mono">{{ config.VXID }}</span>
            <span v-if="config.title" class="text-muted">
                {{ config.title }}</span
            >
        </h1>

        <div class="space-y-6">
            <!-- Destinations -->
            <section class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("vbExport.destinations") }}
                </h3>
                <div class="flex flex-col gap-2">
                    <UCheckbox
                        v-for="d in config.destinations"
                        :key="d"
                        v-model="destChecked[d]"
                        color="neutral"
                    >
                        <template #label>
                            <span class="text-sm">{{ destinationName(d) }}</span>
                            <span class="text-muted font-mono text-xs ml-2">
                                {{ d }}</span
                            >
                        </template>
                    </UCheckbox>
                    <p
                        v-if="config.destinations.length === 0"
                        class="text-muted text-xs"
                    >
                        {{ $t("vbExport.noDestinations") }}
                    </p>
                </div>
            </section>

            <!-- Subtitles (burn-in) -->
            <UFormField :label="$t('vbExport.subtitlesBurnIn')">
                <USelect
                    v-model="subtitleShape"
                    :items="config.subtitleShapes"
                    class="w-full"
                />
            </UFormField>

            <!-- Subtitles burn in style -->
            <UFormField :label="$t('vbExport.subtitlesBurnInStyle')">
                <USelect
                    v-model="subtitleStyle"
                    :items="config.subtitleStyles"
                    class="w-full"
                />
            </UFormField>

            <!-- Submit -->
            <UButton
                block
                size="lg"
                icon="tabler:file-export"
                :loading="submitting"
                :disabled="selectedDestCount === 0"
                @click="startExport"
            >
                {{ $t("vbExport.startExport") }}
            </UButton>
        </div>
    </div>
</template>
