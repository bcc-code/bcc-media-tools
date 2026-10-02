<script setup lang="ts">
import {
    createBurst,
    createCannon,
    particleOpacity,
    stepBurst,
} from "~/utils/confetti";

/**
 * A short burst of confetti, for the rare moment worth celebrating.
 *
 * Mount it (`v-if`) to fire two cannons from the bottom corners; the burst ends
 * on its own once every piece is spent, so nothing keeps animating behind a
 * closed dialog. It renders nothing at all for users who have asked for reduced
 * motion.
 *
 * The pieces are simulated rather than tweened — see `~/utils/confetti` — which
 * is what makes paper read as paper. Everything is drawn to one canvas, so a few
 * hundred pieces cost one element and no layout.
 */
interface Props {
    /** Pieces per cannon. */
    count?: number;
}

const props = withDefaults(defineProps<Props>(), {
    count: 150,
});

// Tokens would follow the theme, which is the opposite of what confetti wants:
// these stay the same festive colours in light and dark.
const COLORS = [
    "#4dd27d",
    "#5c95f1",
    "#e5bf4e",
    "#ec6565",
    "#b07de8",
    "#59d7d2",
];

const canvas = ref<HTMLCanvasElement>();
const reducedMotion = usePreferredReducedMotion();

onMounted(() => {
    const el = canvas.value;
    if (!el || reducedMotion.value === "reduce") return;

    const ctx = el.getContext("2d");
    if (!ctx) return;

    let width = 0;
    let height = 0;
    const resize = () => {
        const dpr = window.devicePixelRatio || 1;
        width = window.innerWidth;
        height = window.innerHeight;
        el.width = Math.floor(width * dpr);
        el.height = Math.floor(height * dpr);
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    };
    resize();
    window.addEventListener("resize", resize);

    // Two cannons, firing up and inwards from the bottom corners, so the middle
    // of the screen — where the dialog is — stays readable.
    const particles = createBurst(
        [
            createCannon(0, height, 1, COLORS),
            createCannon(width, height, -1, COLORS),
        ],
        props.count,
        Math.random,
    );

    let frame = 0;
    let previous = performance.now();

    const step = (now: number) => {
        // Clamped so a backgrounded tab does not teleport everything off-screen.
        const dt = Math.min((now - previous) / 1000, 1 / 30);
        previous = now;

        stepBurst(particles, dt, height);

        ctx.clearRect(0, 0, width, height);
        for (const p of particles) {
            ctx.save();
            ctx.globalAlpha = particleOpacity(p);
            ctx.translate(p.x, p.y);
            ctx.rotate(p.tilt);
            // Foreshortening as the piece flutters edge-on to the viewer.
            ctx.scale(1, Math.cos(p.wobble));
            ctx.fillStyle = p.color;
            ctx.fillRect(-p.width / 2, -p.height / 2, p.width, p.height);
            ctx.restore();
        }

        if (particles.length) frame = requestAnimationFrame(step);
    };

    frame = requestAnimationFrame(step);

    onBeforeUnmount(() => {
        cancelAnimationFrame(frame);
        window.removeEventListener("resize", resize);
    });
});
</script>

<template>
    <Teleport to="#teleports">
        <canvas
            ref="canvas"
            class="pointer-events-none fixed inset-0 z-[60] h-full w-full"
            aria-hidden="true"
        />
    </Teleport>
</template>
