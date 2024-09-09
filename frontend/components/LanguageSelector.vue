<script lang="ts" setup>
import { BccSelect } from "@bcc-code/design-library-vue";
import type { BmmEnvironment } from "~/src/gen/api/v1/api_pb";

const props = defineProps<{
    languages: string[];
    env: BmmEnvironment;
}>();

const model = defineModel<string>();
const bmmLanguages = ref<string[]>([]);

const api = useAPI();

watch(
    [() => props.env, () => props.languages],
    async ([newEnv, newLanguages]) => {
        if (newLanguages.length > 0) {
            bmmLanguages.value = newLanguages;
        } else {
            bmmLanguages.value = (
                await api.getLanguages({ environment: newEnv })
            ).Languages;
        }

        if (bmmLanguages.value.length == 1) {
            model.value = bmmLanguages.value[0];
        }
    },
    { immediate: true },
);

const languageDisplay = (l: string) => {
    if (typeof Intl.DisplayNames !== "undefined") {
        const dn = new Intl.DisplayNames(["en"], { type: "language" });
		let name = dn.of(l);

		// Chrome doesn't support "kha"
		if (name == "kha") {
			return "Khasi";
		} else if (name == "zxx") {
			return "Instrumental";
		}

        return dn.of(l);
    }
};
</script>

<template>
    <BccSelect required v-model="model" :label="$t('language')">
        <option v-if="bmmLanguages.length > 1" disabled value="">{{ $t("selectAnOption") }}</option>
        <option v-for="l in bmmLanguages" :value="l">
            {{ languageDisplay(l) }}
        </option>
    </BccSelect>
</template>
