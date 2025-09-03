import { create } from '@bufbuild/protobuf';
import { useAPI } from "~/utils/api";
import type { Permissions } from "~~/src/gen/api/v1/api_pb";
import { BMMPermissionSchema, PermissionsSchema } from "~~/src/gen/api/v1/api_pb";

export type Me = {
    admin: boolean;
    email: string;
    bmm: {
        languages: string[];
        albums: string[];
    };
};

export function usePermissionsLoading() {
    return useState("me-loading", () => false);
}

export function useMe() {
    const me = useState<Permissions | null>("me", () => null);

    const loading = usePermissionsLoading();

    const loaded = useState("me-loaded", () => false);

    const load = async () => {
        loading.value = true;
        const api = useAPI();
        const p = await api.getPermissions({});
        if (p) {
            me.value = p;
        } else {
            me.value = create(PermissionsSchema);
            me.value!.bmm = create(BMMPermissionSchema);
        }

        loading.value = false;
        loaded.value = true;
    };

    if (!loaded.value && !loading.value) {
        load();
    }

    return { me, loading };
}
