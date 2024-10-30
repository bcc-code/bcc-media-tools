<script lang="ts" setup>
import { BmmEnvironment, Permissions } from "~/src/gen/api/v1/api_pb";
import {
    BccButton,
    BccFormLabel,
    BccToggle,
} from "@bcc-code/design-library-vue";

const props = defineProps<{
    email: string;
    permissions: Permissions;
}>();

defineEmits<{
    remove: [];
}>();

const perms = reactive(props.permissions);
const api = useAPI();

const availableLanguages = ref<string[]>([]);
api.getLanguages({ environment: BmmEnvironment.Production }).then(
    (result) =>
        (availableLanguages.value = result.Languages.map((f) => f.code)),
);

watch(perms, () => {
    api.updatePermissions({
        email: props.email,
        permissions: perms,
    });
});
</script>

<template>
    <div class="flex flex-col rounded-lg border bg-white p-4">
        <div class="flex justify-between">
            <h3 class="text-lg">{{ email }}</h3>
            <BccButton @click="$emit('remove')" size="sm">Remove</BccButton>
        </div>
        <div class="flex max-w-full gap-4 overflow-hidden">
            <div class="flex flex-col rounded border px-4 py-2">
                <h3>General</h3>
                <div class="flex gap-4">
                    <div>
                        <BccFormLabel>Admin</BccFormLabel>
                        <BccToggle v-model="perms.admin" />
                    </div>
                </div>
            </div>
            <div class="flex flex-col rounded border px-4 py-2">
                <h3>BMM</h3>
                <div class="flex gap-4">
                    <div>
                        <div>
                            <BccFormLabel>BMM Admin</BccFormLabel>
                            <BccToggle v-model="perms!.bmm!.admin" />
                        </div>
                        <div>
                            <BccFormLabel>Integration environment</BccFormLabel>
                            <BccToggle v-model="perms!.bmm!.integration" />
                        </div>
                    </div>
                    <div>
                        <BccFormLabel>Podcasts</BccFormLabel>
                        <MultiSelector
                            :available="['fra-kaare', 'gibraltar-podcast']"
                            v-model="perms.bmm!.podcasts"
                        />
                    </div>
                </div>
                <div>
                    <div>
                        <BccFormLabel>Languages</BccFormLabel>
                        <MultiSelector
                            :available="availableLanguages"
                            v-model="perms.bmm!.languages"
                            :label-transformer="(v) => languageCodeToName(v)"
                        />
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
