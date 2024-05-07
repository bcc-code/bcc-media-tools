<script lang="ts" setup>
import { useAPI} from "#imports";
import { Permissions } from "~/src/gen/api/v1/api_pb";
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

watch(perms, () => {
  api.updatePermissions({
    email: props.email,
    permissions: perms
  });
});

</script>

<template>
    <div class="flex flex-col rounded-lg border bg-white p-4">
        <div class="flex justify-between">
            <h3 class="text-lg">{{ email }}</h3>
            <BccButton @click="$emit('remove')" size="sm">Remove</BccButton>
        </div>
        <div class="flex gap-4">
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
                        <BccFormLabel>Albums</BccFormLabel>
                        <MultiSelector
                            :available="['fra-kaare', 'romans']"
                            v-model="perms.bmm!.albums"
                        />
                    </div>
                    <div>
                        <BccFormLabel>Languages</BccFormLabel>
                        <MultiSelector
                            :available="bmmLanguages"
                            v-model="perms.bmm!.languages"
                        />
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
