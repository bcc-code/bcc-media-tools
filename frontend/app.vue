<template>
    <UApp :toaster="{ position: 'bottom-center', expand: false }">
        <NuxtLayout>
            <NuxtRouteAnnouncer />
            <NuxtPage />
            <DevOnly>
                <DevTools />
            </DevOnly>
        </NuxtLayout>
    </UApp>
</template>
