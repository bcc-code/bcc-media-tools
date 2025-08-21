<script lang="ts" setup>
import { useMe } from "~/utils/me";

useHead({
    title: "Transcription",
});

const analytics = useAnalytics();
onMounted(() => {
    analytics.page({
        id: "transcription",
        title: "transcription",
    });
});

const transcription = ref<TranscriptionResult>();

const fileName = ref<string>();

const tKey = ref<string>();

const handleFile = (event: Event) => {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (file) {
        fileName.value = file.name;
        transcription.value = undefined;
        segments.value = [];
        const reader = new FileReader();
        reader.onload = (e) => {
            const result = e.target?.result;
            if (result) {
                transcription.value = JSON.parse(result.toString());
                transcription.value!.segments.forEach((s) => {
                    s.text = s.text.trim();
                    s.words = s.words.map((w) => ({
                        ...w,
                        text: w.text.trim(),
                    }));
                });

                segments.value = transcription.value!.segments ?? [];

                tKey.value = generateRandomId();
            }
        };
        reader.readAsText(file);
    }
};

const segments = ref<Segment[]>([]);

const vxId = ref("");

const { deleteMode } = useDeleteMode();

const { me } = useMe();

const truncatedFileName = computed(() => {
    if (!fileName.value) return;
    if (fileName.value.length < 30) return fileName.value;

    const first = fileName.value.slice(0, 10);
    const last = fileName.value.slice(-10);
    return [first, last].join("...");
});

function setSegments(s: Segment[]) {
    segments.value = s;
    if (!transcription.value) return;
    transcription.value.segments = s;
}
</script>

<template>
    <div
        :class="[
            'mx-auto flex h-screen max-w-7xl p-8',
            {
                'border-8 border-red-700': deleteMode,
            },
        ]"
    >
        <div class="flex flex-grow flex-col">
            <div class="flex max-w-80 items-center gap-4">
                <div class="shrink-0">
                    <label for="file-input" class="cursor-pointer">
                        <UButton class="pointer-events-none">
                            {{
                                fileName && truncatedFileName
                                    ? truncatedFileName
                                    : $t("transcription.selectFile")
                            }}
                        </UButton>
                    </label>
                    <input
                        id="file-input"
                        hidden
                        type="file"
                        placeholder="File here"
                        accept="application/json"
                        @input="handleFile"
                    />
                </div>
                <template
                    v-if="
                        !fileName && me?.transcription && me.transcription.admin
                    "
                >
                    <span class="text-muted text-sm">
                        {{ $t("transcription.or") }}
                    </span>
                    <form
                        class="bg-default flex gap-1 rounded-xl p-2"
                        @submit.prevent="navigateTo(`/transcription/${vxId}`)"
                    >
                        <UInput
                            v-model="vxId"
                            placeholder="Vidispine-ID"
                            class="min-w-32"
                        />
                        <UButton variant="ghost" type="submit">
                            {{ $t("transcription.load") }}
                        </UButton>
                    </form>
                </template>
                <TranscriptionDownloader
                    v-if="fileName"
                    :segments="segments"
                    :filename="fileName"
                />
            </div>
            <TranscriptionEditor
                :key="tKey"
                :transcription="transcription"
                :file-name="fileName"
                v-model="segments"
                @update-segments="(s) => setSegments(s)"
            />
        </div>
    </div>
</template>
