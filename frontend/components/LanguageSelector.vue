<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";
import type {BmmEnvironment} from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    languages: readonly string[];
    env: BmmEnvironment;
}>();

const value = defineModel<string>();
const bmmLanguages = ref<string[]>([]);

const api = useAPI();

watch(() => props.env, async(env)=> {
  bmmLanguages.value = (await api.getLanguages({environment: env})).Languages;
}, {immediate: true});


const languageDisplay = (l: string) => {
    if (typeof Intl.DisplayNames !== "undefined") {
        const dn = new Intl.DisplayNames(["en"], { type: "language" });
        return dn.of(l);
    }
};
</script>

<template>
    <BccSelect v-model="value" :label="$t('language')">
        <option disabled value="">{{ $t("selectAnOption") }}</option>
        <option v-for="l in bmmLanguages" :value="l">
            {{ languageDisplay(l) }}
        </option>
    </BccSelect>
</template>
