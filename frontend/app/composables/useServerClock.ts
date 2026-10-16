// How far this computer's clock is from the server's (see utils/serverClock).
// Measured on mount, every 10 minutes and when the tab is shown again, since
// laptops drift and a sleeping one can jump. Only measured and shown for now:
// markers are still stamped with this computer's clock.

const SAMPLES = 5;
const REMEASURE_MS = 10 * 60_000;

// The wall clock, not performance.timeOrigin + performance.now(): on macOS
// that monotonic clock stops while the computer sleeps, so a tab left open
// overnight would read hours behind. A measurement lasts milliseconds, so a
// system clock change in the middle of one doesn't matter.
function wallNow(): number {
    return Date.now();
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
                const sentAt = wallNow();
                try {
                    const res = await api.getServerTime({});
                    const receivedAt = wallNow();
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
            : wallNow() + offsetMs.value;
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
