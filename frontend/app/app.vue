<template>
    <NuxtLayout>
        <NuxtRouteAnnouncer />
        <NuxtPage />
        <DevOnly>
            <DevTools />
        </DevOnly>
    </NuxtLayout>
    <DesignToastProvider />
</template>
