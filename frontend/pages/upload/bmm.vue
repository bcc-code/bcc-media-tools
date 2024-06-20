<script lang="ts" setup>
const form = ref<BMMSingleForm>({
    title: "",
});

const metadataIsSet = ref(false);

const selectedFile = ref<File | null>(null);

const { me } = useMe();
const config = useRuntimeConfig();

const metadata = computed(() => {
    return {
        title: [form.value.title],
        language: [form.value.language],
        trackId: [form.value.trackId?.toString() ?? ""],
    } as { [key: string]: readonly string[] };
});

const uploaded = ref(false);


</script>

<template>
    <div
        class="mx-auto flex min-h-screen max-w-screen-md flex-col gap-4 rounded-lg bg-stone-300 p-4 text-black"
        v-if="me && me.bmm"
    >
      <template v-if="me.bmm && (me.bmm.podcasts.length > 0 || me.bmm.admin)">
          <template v-if="!uploaded">
              <BmmSingleMetadata
                  v-model="form"
                  @set="metadataIsSet = true"
                  :permissions="me.bmm"
              />
              <div
                  class="flex flex-col gap-4 p-4 transition"
                  :class="[
                      {
                          'pointer-events-none opacity-50': !metadataIsSet,
                      },
                  ]"
              >
                  <h3 class="text-lg font-bold">Upload File</h3>
                  <SelectFile v-model="selectedFile" />
                  <FileUploader
                      v-model="selectedFile"
                      :endpoint="config.public.grpcURL + '/upload'"
                      :metadata="metadata"
                      @uploaded="uploaded = true"
                  />
              </div>
          </template>
          <template v-else>
              <div class="rounded-lg bg-green-500 p-4">{{ $t("uploaded") }}</div>
          </template>
      </template>
      <template v-else>
          You don't have enough permissions
      </template>
    </div>
</template>
