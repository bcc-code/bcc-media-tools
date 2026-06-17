import { ref, watch, nextTick, type Ref } from "vue";
import type { Router } from "vue-router";

type Primitive = string | number | boolean;
type QueryValue = Primitive | Primitive[];

// Per-router pending-update map so simultaneous changes to multiple
// useQueryRef instances flush as one router.replace, instead of each
// watcher reading stale route.query and clobbering the others.
const pendingFlushes = new WeakMap<Router, Map<string, string | null>>();

/**
 * Bidirectional sync between a Ref and a URL query param.
 *
 * The param drops out of the URL when value === defaultValue, so unset
 * filters don't pollute it. Type-driven serialization: arrays join on
 * comma, numbers/booleans round-trip through String(), strings pass through.
 *
 * Multi-ref updates within the same tick are batched into a single
 * router.replace(); back/forward navigation and deep links re-populate the
 * refs from the URL.
 */
export function useQueryRef<T extends QueryValue>(
    name: string,
    defaultValue: T,
): Ref<T> {
    const route = useRoute();
    const router = useRouter();

    const state = ref(
        parseQueryValue<T>(route.query[name], defaultValue),
    ) as Ref<T>;

    let suppressNext = false;

    // ref → URL (batched via nextTick across all instances)
    watch(
        state,
        (val) => {
            if (suppressNext) return;
            const serialized = equals(val, defaultValue)
                ? null
                : serialize(val);
            scheduleQueryUpdate(router, name, serialized);
        },
        { deep: true },
    );

    // URL → ref (back/forward, manual edits)
    watch(
        () => route.query[name],
        (raw) => {
            const next = parseQueryValue<T>(raw, defaultValue);
            if (!equals(next, state.value)) {
                suppressNext = true;
                state.value = next;
                nextTick(() => {
                    suppressNext = false;
                });
            }
        },
    );

    return state;
}

function scheduleQueryUpdate(
    router: Router,
    name: string,
    value: string | null,
) {
    let queue = pendingFlushes.get(router);
    const fresh = !queue;
    if (!queue) {
        queue = new Map();
        pendingFlushes.set(router, queue);
    }
    queue.set(name, value);

    if (!fresh) return;

    nextTick(() => {
        const flushQueue = pendingFlushes.get(router);
        pendingFlushes.delete(router);
        if (!flushQueue) return;

        const merged: Record<string, string | undefined> = {
            ...(router.currentRoute.value.query as Record<string, string>),
        };
        for (const [n, v] of flushQueue) {
            if (v === null) delete merged[n];
            else merged[n] = v;
        }
        router.replace({ query: merged });
    });
}

function parseQueryValue<T>(raw: unknown, defaultValue: T): T {
    const value = Array.isArray(raw) ? raw[0] : raw;
    if (value == null || value === "") return defaultValue;
    if (Array.isArray(defaultValue)) {
        return String(value).split(",").filter(Boolean) as unknown as T;
    }
    if (typeof defaultValue === "number") {
        const n = Number(value);
        return (Number.isFinite(n) ? n : defaultValue) as unknown as T;
    }
    if (typeof defaultValue === "boolean") {
        return (value === "1" || value === "true") as unknown as T;
    }
    return String(value) as unknown as T;
}

function serialize(val: QueryValue): string {
    if (Array.isArray(val)) return val.join(",");
    if (typeof val === "boolean") return val ? "1" : "0";
    return String(val);
}

function equals<T extends QueryValue>(a: T, b: T): boolean {
    if (Array.isArray(a) && Array.isArray(b)) {
        if (a.length !== b.length) return false;
        return a.every((v, i) => v === b[i]);
    }
    return a === b;
}
