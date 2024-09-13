<script setup lang="ts">
import { BccItemTile, BccLinkItem } from "@bcc-code/design-library-vue";
</script>

<template>
    <div class="grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-4 p-8">
        <NuxtLink to="/upload/bmm/">
            <BccItemTile
                title="BMM Upload"
                class="border-on-secondary bg-white"
            />
        </NuxtLink>
        <NuxtLink to="/transcription">
            <BccItemTile
                title="Transcription"
                class="border-on-secondary bg-white"
            />
        </NuxtLink>
    </div>
</template>
