import type { Clip } from "~/utils/shortsClips";

/**
 * The set of clips being cut from one asset, and which of them is being edited.
 *
 * There is no separate "draft" range: the active clip *is* the range the
 * timeline and transcript edit. That removes the failure where a user trims a
 * range, submits, and the last clip is missing because they never pressed Add.
 *
 * Clips are kept in localStorage per asset. Losing one selection to a reload is
 * annoying; losing five is a setback.
 */
export function useShortsClips(
    vxId: MaybeRefOrGetter<string | undefined>,
    duration: MaybeRefOrGetter<number | undefined>,
) {
    const storageKey = computed(() => `shortsClips:${toValue(vxId) ?? "none"}`);
    const stored = useLocalStorage<Clip[]>(storageKey, []);

    /*
     * Creation order, deliberately not time order. Sorting by start time means
     * trimming a clip earlier renumbers the list under the user's hands, so the
     * "clip 3" they were just looking at becomes clip 2. The timecode on each
     * row says where it sits; the number says which one it is.
     */
    const clips = computed(() => stored.value);
    const activeId = ref<string>();

    const activeClip = computed(() =>
        stored.value.find((c) => c.id === activeId.value),
    );

    function newId() {
        return generateRandomId();
    }

    function patchActive(patch: Partial<Pick<Clip, "start" | "end">>) {
        stored.value = stored.value.map((clip) =>
            clip.id === activeId.value ? { ...clip, ...patch } : clip,
        );
    }

    /*
     * The timeline and transcript were written against a single start/end pair,
     * so the active clip is exposed the same way rather than rewriting them to
     * understand a list.
     */
    const activeStart = computed<number | undefined>({
        get: () => activeClip.value?.start,
        set: (value) => {
            if (value != undefined) patchActive({ start: value });
        },
    });

    const activeEnd = computed<number | undefined>({
        get: () => activeClip.value?.end,
        set: (value) => {
            if (value != undefined) patchActive({ end: value });
        },
    });

    function activate(id: string) {
        if (stored.value.some((c) => c.id === id)) activeId.value = id;
    }

    function add(playhead: number): Clip | undefined {
        const range = nextClipRange(playhead, toValue(duration) ?? 0);
        if (!range) return undefined;

        const clip: Clip = { id: newId(), ...range };
        stored.value = [...stored.value, clip];
        activeId.value = clip.id;
        return clip;
    }

    function remove(id: string) {
        const remaining = stored.value.filter((c) => c.id !== id);
        stored.value = remaining;

        if (activeId.value !== id) return;
        activeId.value = remaining[0]?.id;
    }

    /**
     * Seeds the list once the media duration is known: restores what was stored
     * for this asset, or starts with one clip spanning the whole video, which is
     * how the editor behaved before it could hold more than one.
     */
    function initialise() {
        const total = toValue(duration);
        if (!total) return;

        const restored = sanitiseClips(stored.value, total);
        stored.value = restored.length
            ? restored
            : [{ id: newId(), start: 0, end: total }];
        activeId.value = stored.value[0]?.id;
    }

    /** Called once a batch is away, so a reload does not re-offer sent clips. */
    function clear() {
        stored.value = [];
        activeId.value = undefined;
    }

    const overlapping = computed(() => overlappingClipIds(clips.value));
    const submittable = computed(() => clips.value.filter(isValidClip));

    return {
        clips,
        activeId,
        activeClip,
        activeStart,
        activeEnd,
        overlapping,
        submittable,
        activate,
        add,
        remove,
        clear,
        initialise,
    };
}
