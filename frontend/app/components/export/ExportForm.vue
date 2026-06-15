<script setup lang="ts">
import type { GetExportConfigResponse } from "~~/src/gen/api/v1/api_pb";
import type { ExportSelection } from "~/components/export/types";

const props = defineProps<{
    config: GetExportConfigResponse;
    submitting?: boolean;
}>();

const emit = defineEmits<{
    (e: "start-export", payload: ExportSelection): void;
    (e: "export-timed-metadata"): void;
}>();

const { t } = useI18n();

/* ------------------------------------------------------------------ state --- */

const destChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(props.config.destinations.map((d) => [d, false])),
);

const audioSource = ref(
    props.config.audioSources.includes(props.config.selectedAudioSource)
        ? props.config.selectedAudioSource
        : (props.config.audioSources[0] ?? ""),
);

const langChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(
        props.config.languages.map((l) => [
            l.code,
            props.config.selectedLanguages.includes(l.code),
        ]),
    ),
);

const resolutions = reactive(
    props.config.resolutions.map((r) => ({
        width: r.width,
        height: r.height,
        enabled: true,
        downloadable: false,
    })),
);

const overlay = ref("None");
const subclipChecked = reactive<Record<string, boolean>>(
    Object.fromEntries(props.config.subclips.map((s) => [s.title, false])),
);

const withChapters = ref(false);
const ignoreSilence = ref(false);
const exportAiSubs = ref(false);

/* --------------------------------------------------------------- computed --- */

const selectedLangCount = computed(
    () => props.config.languages.filter((l) => langChecked[l.code]).length,
);

const selectedDestCount = computed(
    () => props.config.destinations.filter((d) => destChecked[d]).length,
);

// Aspect ratio of the first resolution, e.g. "16:9".
const aspectRatio = computed(() => {
    const first = resolutions[0];
    if (!first) return "";
    const gcd = (a: number, b: number): number => (b === 0 ? a : gcd(b, a % b));
    const d = gcd(first.width, first.height);
    return d === 0 ? "" : `${first.width / d}:${first.height / d}`;
});

/* ---------------------------------------------------------------- actions --- */

function setLangs(codes: string[]) {
    props.config.languages.forEach(
        (l) => (langChecked[l.code] = codes.includes(l.code)),
    );
}
const selectAllLangs = () =>
    setLangs(props.config.languages.map((l) => l.code));
const clearLangs = () => setLangs([]);
const selectMU1 = () =>
    setLangs(props.config.languages.filter((l) => l.mu1).map((l) => l.code));
const selectMU2 = () =>
    setLangs(props.config.languages.filter((l) => l.mu2).map((l) => l.code));

function startExport() {
    emit("start-export", {
        destinations: props.config.destinations.filter((d) => destChecked[d]),
        audioSource: audioSource.value,
        languages: props.config.languages
            .filter((l) => langChecked[l.code])
            .map((l) => l.code),
        resolutions: resolutions
            .filter((r) => r.enabled)
            .map((r) => ({
                width: r.width,
                height: r.height,
                downloadable: r.downloadable,
            })),
        overlay: overlay.value,
        withChapters: withChapters.value,
        ignoreSilence: ignoreSilence.value,
        exportAiSubs: exportAiSubs.value,
        subclips: props.config.subclips
            .filter((s) => subclipChecked[s.title])
            .map((s) => s.title),
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
            <!-- Alternative actions -->
            <UCard variant="subtle" :ui="{ body: 'space-y-2' }">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.alternativeActions") }}
                </h3>
                <UButton
                    color="neutral"
                    variant="outline"
                    size="sm"
                    icon="tabler:file-export"
                    :disabled="submitting"
                    @click="emit('export-timed-metadata')"
                >
                    {{ $t("export.exportTimedMetadata") }}
                </UButton>
                <p class="text-muted text-xs">
                    {{ $t("export.exportTimedMetadataHint") }}
                </p>
            </UCard>

            <!-- Destinations -->
            <section class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.destinations") }}
                </h3>
                <div class="flex flex-col gap-2">
                    <UCheckbox
                        v-for="d in config.destinations"
                        :key="d"
                        v-model="destChecked[d]"
                        color="neutral"
                    >
                        <template #label>
                            <span class="font-mono text-sm">{{ d }}</span>
                        </template>
                    </UCheckbox>
                    <p
                        v-if="config.destinations.length === 0"
                        class="text-muted text-xs"
                    >
                        {{ $t("export.noDestinations") }}
                    </p>
                </div>
            </section>

            <!-- Audio source -->
            <UFormField :label="$t('export.audioSource')">
                <USelect
                    v-model="audioSource"
                    :items="config.audioSources"
                    class="w-full"
                />
            </UFormField>

            <!-- Subclips -->
            <section class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.subclips") }}
                </h3>
                <p class="text-muted text-xs">{{ $t("export.subclipsHint") }}</p>
                <div
                    v-if="config.subclips.length > 0"
                    class="flex flex-col gap-2"
                >
                    <UCheckbox
                        v-for="s in config.subclips"
                        :key="s.title"
                        v-model="subclipChecked[s.title]"
                        color="neutral"
                        :label="s.title"
                    />
                </div>
            </section>

            <!-- Language exports -->
            <section class="space-y-3">
                <div class="flex flex-wrap items-center justify-between gap-2">
                    <h3 class="text-highlighted text-sm font-semibold">
                        {{ $t("export.languageExports") }}
                    </h3>
                    <div class="flex items-center gap-2">
                        <span class="text-muted text-xs">
                            {{ $t("export.nSelected", { n: selectedLangCount }) }}
                        </span>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="selectAllLangs"
                        >
                            {{ $t("export.all") }}
                        </UButton>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="clearLangs"
                        >
                            {{ $t("export.none") }}
                        </UButton>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="selectMU1"
                        >
                            MU1
                        </UButton>
                        <UButton
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            @click="selectMU2"
                        >
                            MU2
                        </UButton>
                    </div>
                </div>
                <div class="grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2">
                    <UCheckbox
                        v-for="l in config.languages"
                        :key="l.code"
                        v-model="langChecked[l.code]"
                        color="neutral"
                    >
                        <template #label>
                            <span class="font-mono text-sm">{{ l.code }}</span>
                            <span class="text-muted"> · {{ l.name }}</span>
                        </template>
                    </UCheckbox>
                </div>
            </section>

            <!-- Resolutions -->
            <section class="space-y-2">
                <h3 class="text-highlighted text-sm font-semibold">
                    {{ $t("export.resolutions")
                    }}<span v-if="aspectRatio"> — {{ aspectRatio }}</span>
                </h3>
                <div class="flex flex-col gap-2">
                    <div
                        v-for="r in resolutions"
                        :key="`${r.width}x${r.height}`"
                        class="flex items-center gap-6"
                    >
                        <UCheckbox
                            v-model="r.enabled"
                            color="neutral"
                            :ui="{ root: 'w-28' }"
                        >
                            <template #label>
                                <span class="font-mono text-sm">
                                    {{ r.width }}x{{ r.height }}
                                </span>
                            </template>
                        </UCheckbox>
                        <UCheckbox
                            v-model="r.downloadable"
                            color="neutral"
                            :label="$t('export.downloadable')"
                            :disabled="!r.enabled"
                        />
                    </div>
                </div>
            </section>

            <!-- Overlay -->
            <UFormField :label="$t('export.overlay')">
                <USelect
                    v-model="overlay"
                    :items="config.overlays"
                    class="w-full"
                />
            </UFormField>

            <!-- Toggles -->
            <div class="space-y-3">
                <UCheckbox
                    v-model="withChapters"
                    color="neutral"
                    :label="$t('export.withChapters')"
                />
                <UCheckbox
                    v-model="ignoreSilence"
                    color="neutral"
                    :label="$t('export.ignoreSilence')"
                />
                <UCheckbox
                    v-model="exportAiSubs"
                    color="neutral"
                    :label="$t('export.exportAiSubs')"
                />
            </div>

            <!-- Submit -->
            <UButton
                block
                size="lg"
                icon="tabler:file-export"
                :loading="submitting"
                :disabled="selectedDestCount === 0"
                @click="startExport"
            >
                {{ $t("export.startExport") }}
            </UButton>
        </div>
    </div>
</template>
