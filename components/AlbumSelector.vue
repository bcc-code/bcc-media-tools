<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";

defineProps<{
    albums: readonly string[];
}>();

const value = defineModel<string>();
</script>

<template>
    <BccSelect v-model="value" :label="$t('album')">
        <option disabled value="">{{ $t("selectAnOption") }}</option>
        <option v-for="a in albums" :value="a">
            {{ a }}
        </option>
    </BccSelect>
</template>
