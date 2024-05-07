import {useAPI} from "~/utils/api";
import {BMMPermission, Permissions} from "~/src/gen/api/v1/api_pb";

export type Me = {
    admin: boolean;
    bmm: {
        languages: string[];
        albums: string[];
    };
};


export function useMe() {
    const me = useState<Permissions | null>("me", () => null);

    const loading = useState("me-loading", () => false);

    const loaded = useState("me-loaded", () => false);

    const load = async () => {
        loading.value = true;
        const api = useAPI()
        const p = await api.getPermissions({})
        if (p) {
            me.value = p;
        } else {
            me.value = new Permissions()
            me.value!.bmm = new BMMPermission()
        }

        loading.value = false;
        loaded.value = true;
    };

    if (!loaded.value && !loading.value) {
        load();
    }

    return { me, loading };
}
