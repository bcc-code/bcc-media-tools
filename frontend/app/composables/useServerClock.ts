// How far this computer's clock is from the server's (see utils/serverClock).
// Measured on mount, every 10 minutes and when the tab is shown again, since
// laptops drift and a sleeping one can jump. Only measured and shown for now:
// markers are still stamped with this computer's clock.

const SAMPLES = 5;
const REMEASURE_MS = 10 * 60_000;

function monotonicNow(): number {
    return performance.timeOrigin + performance.now();
}

export function useServerClock() {
    const api = useAPI();
    const offsetMs = ref<number>();
    const uncertaintyMs = ref<number>();
    let measuring = false;

    async function measure() {
        if (measuring) return;
        measuring = true;
        try {
            const samples: ClockSample[] = [];
            for (let i = 0; i < SAMPLES; i++) {
                const sentAt = monotonicNow();
                try {
                    const res = await api.getServerTime({});
                    const receivedAt = monotonicNow();
                    const server = timestampToDate(res.now);
                    if (!server) continue;
                    samples.push({
                        sentAt,
                        receivedAt,
                        serverMs: server.getTime(),
                    });
                } catch {
                    // A lost sample is skipped; the others still count.
                }
            }
            const best = bestOffset(samples);
            if (best) {
                offsetMs.value = best.offsetMs;
                uncertaintyMs.value = best.uncertaintyMs;
            }
        } finally {
            measuring = false;
        }
    }

    // The server's time now, or this computer's if not measured yet.
    function serverNow(): number {
        return offsetMs.value === undefined
            ? Date.now()
            : monotonicNow() + offsetMs.value;
    }

    let timer: ReturnType<typeof setInterval> | undefined;
    function onVisible() {
        if (document.visibilityState === "visible") void measure();
    }
    onMounted(() => {
        void measure();
        timer = setInterval(measure, REMEASURE_MS);
        document.addEventListener("visibilitychange", onVisible);
    });
    onBeforeUnmount(() => {
        clearInterval(timer);
        document.removeEventListener("visibilitychange", onVisible);
    });

    return { offsetMs, uncertaintyMs, measure, serverNow };
}
