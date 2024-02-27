export type Me = {
    admin: boolean;
    bmm: {
        languages: string[];
        albums: string[];
    };
};

export function useMe() {
    const me = useState<Me | null>("me", () => null);

    const loading = useState("me-loading", () => false);

    const loaded = useState("me-loaded", () => false);

    const load = async () => {
        loading.value = true;
        me.value = (await $fetch("/api/bmm/me", { method: "GET" })) ?? {
            admin: false,
            bmm: {
                languages: [],
                albums: [],
            },
        };
        loading.value = false;
        loaded.value = true;
    };

    if (!loaded.value && !loading.value) {
        load();
    }

    return { me, loading };
}
