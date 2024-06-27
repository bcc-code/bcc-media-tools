<script lang="ts" setup>
import { BccCheckbox } from "@bcc-code/design-library-vue";

defineProps<{
    available: readonly string[];
}>();

const value = defineModel<string[]>({ required: true });

const toggleCheckbox = (v: string) => {
    if (value.value.includes(v)) {
        value.value = value.value.filter((x) => x !== v);
    } else {
        value.value = [...value.value, v];
    }
};
</script>
<template>
    <div class="flex flex-wrap gap-2 gap-x-4">
        <div v-for="v in available">
            <BccCheckbox
                :label="v"
                :model-value="value.includes(v) === true"
                @update:model-value="toggleCheckbox(v)"
            />
        </div>
    </div>
</template>
