<script lang="ts" setup>
import { BccButton } from "@bcc-code/design-library-vue";
import type { FileAndLanguage } from "~/utils/bmm";
import { analytics } from "~/utils/analytics";

const props = defineProps<{
    endpoint: string;
    metadata: { [key: string]: readonly string[] };
}>();

const emit = defineEmits<{
    uploaded: [];
}>();

const selectedFiles = defineModel<FileAndLanguage[]>({ required: true });
const uploadPercentageFiles = ref<{ [key: string]: number }>({});
const uploading = ref(false);
const uploadPercentage = ref(0);

watch(
    uploadPercentageFiles,
    () => {
        uploadPercentage.value =
            Object.values(uploadPercentageFiles.value).reduce(
                (a, b) => a + b,
                0,
            ) / Object.keys(uploadPercentageFiles.value).length;
    },
    { deep: true },
);

watch(selectedFiles, () => {
    uploadPercentage.value = 0;
    uploadPercentageFiles.value = {};
    uploading.value = false;
});

const abort = ref<() => void>();

const showProgress = ref(false);

const uploadFile = () => {
    for (const selectedFile of selectedFiles.value || []) {
        const start = Date.now();

        analytics.track("upload_started", {
            language: selectedFile.language,
            trackId:props.metadata.trackId[0],
        });

        if (!selectedFile.file) return;
        uploading.value = true;

        const formData = new FormData();
        formData.append("file", selectedFile.file);
        formData.append("file_language", selectedFile.language);
        if (props.metadata) {
            for (const [key, values] of Object.entries(props.metadata)) {
                for (const value of values) {
                    formData.append(key, value);
                }
            }
        }

        const xhr = new XMLHttpRequest();
        let startedAt: number;
        xhr.open("post", props.endpoint, true);
        xhr.onloadstart = function () {
            startedAt = Date.now();
        };
        xhr.upload.onprogress = function (ev) {
            // Upload progress here
            uploadPercentageFiles.value[selectedFile.file.name] =
                Math.floor((ev.loaded / ev.total) * 1000) / 10;

            // Only show progress indicator in UI after 200ms
            if (Date.now() - startedAt >= 200) {
                showProgress.value = true;
            }
        };

        let errHandler = (e:ProgressEvent) => {
            uploading.value = false;
            console.log(e);

            const t = e.target as XMLHttpRequest;

            analytics.track("upload_finished", {
                language: selectedFile.language,
                trackId:props.metadata.trackId[0],
                success: false,
                error: t.statusText,
                duration: Date.now() - start,
            });

            if (confirm("Upload failed, try again?")) {
                uploadFile();
            }
        }

        xhr.onerror = errHandler;
        xhr.onabort = errHandler;

        xhr.onload = function (e:ProgressEvent) {
            const t = e.target as XMLHttpRequest;

            if (t.status != 202) {
                errHandler(e);
                return
            }

            analytics.track("upload_finished", {
                language: selectedFile.language,
                trackId:props.metadata.trackId[0],
                success: true,
                duration: Date.now() - start,
                size: selectedFile.file.size,
            });

            emit("uploaded");
            showProgress.value = false;
        };
        xhr.send(formData);
        abort.value = () => {
            xhr.abort();
            showProgress.value = false;
            uploading.value = false;
        };
    }
};
</script>

<template>
    <div class="flex flex-col gap-2">
        <BccButton
            @click="uploadFile"
            v-if="!uploading"
            :disabled="selectedFiles.length < 1"
        >
            Upload
        </BccButton>
        <BccButton
            @click="abort"
            v-else
            variant="secondary"
            :disabled="uploadPercentage >= 100"
        >
            Cancel
        </BccButton>
        <Transition
            enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0 -translate-y-2 scale-95"
            enter-to-class="opacity-100 translate-y-0 scale-100"
            leave-active-class="transition duration-300 ease-out absolute"
            leave-from-class="opacity-100 translate-y-0 scale-100"
            leave-to-class="opacity-0 -translate-y-2 scale-95"
        >
            <div
                v-if="showProgress"
                class="relative overflow-clip rounded-lg border border-neutral-200 bg-neutral-100 p-1"
            >
                <div
                    class="flex h-6 rounded bg-green-600"
                    :style="{ width: `${uploadPercentage}%` }"
                />
                <div
                    class="absolute right-3 top-1/2 -translate-y-1/2 tabular-nums"
                >
                    {{
                        uploadPercentage !== 100
                            ? uploadPercentage.toFixed(1)
                            : 100
                    }}%
                </div>
            </div>
        </Transition>
    </div>
</template>
