<script setup lang="ts">
import { en, nb_no } from "@nuxt/ui/locale";

const { locale: l } = useI18n();
const locale = computed(() => {
    return getLocale(l.value);
});

function getLocale(l: string) {
    if (l == "nb") return nb_no;
    return en;
}
</script>

<template>
    <UApp :locale="locale">
        <NuxtLayout>
            <NuxtRouteAnnouncer />
            <NuxtPage />
            <DevOnly>
                <DevTools />
            </DevOnly>
        </NuxtLayout>
    </UApp>
</template>
