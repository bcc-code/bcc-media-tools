/**
 * The physics behind `DesignConfetti`: a tiny particle simulation with gravity,
 * air drag and flutter. It is deliberately free of canvas, DOM and timers — the
 * component owns those — so the motion can be stepped and asserted in a test.
 *
 * Distances are CSS pixels, time is seconds, angles radians.
 */

export type ConfettiOptions = {
    gravity: number;
    /** Fraction of velocity shed per frame at 60fps. */
    drag: number;
    /** Sideways acceleration from fluttering. */
    sway: number;
    /** Seconds a piece spends fading once it is spent. */
    fade: number;
};

export const defaultConfettiOptions: ConfettiOptions = {
    gravity: 1200,
    // Drag is what makes paper paper: it caps the fall at a floaty ~650px/s
    // (gravity / 60 / drag) instead of letting pieces accelerate like pebbles.
    drag: 0.03,
    sway: 55,
    fade: 0.6,
};

/**
 * The simulation runs at a fixed 120Hz internally, so a piece follows the same
 * path whatever the display refresh rate: a 30fps frame is simply four steps.
 */
const FIXED_STEP = 1 / 120;

export type Particle = {
    x: number;
    y: number;
    vx: number;
    vy: number;
    width: number;
    height: number;
    color: string;
    /** Rotation in the screen plane. */
    tilt: number;
    tiltSpeed: number;
    /** Phase of the flutter, which both sways and foreshortens the piece. */
    wobble: number;
    wobbleSpeed: number;
    /** Seconds this piece has been alive. */
    life: number;
    /** Seconds before it starts to fade. */
    maxLife: number;
};

export type Cannon = {
    x: number;
    y: number;
    /** 1 fires to the right, -1 to the left. */
    aim: 1 | -1;
    speed: [number, number];
    /** Degrees from horizontal. */
    angle: [number, number];
    /** Sideways scatter of the cannon mouth. */
    spread: number;
    colors: string[];
};

/** The barrel the component fires with, kept here so tests aim the same one. */
export const defaultCannonSpec = {
    speed: [2200, 3400],
    angle: [52, 78],
    spread: 140,
} satisfies Pick<Cannon, "speed" | "angle" | "spread">;

export function createCannon(
    x: number,
    y: number,
    aim: 1 | -1,
    colors: string[],
): Cannon {
    return { x, y, aim, colors, ...defaultCannonSpec };
}

/** Injectable so tests get a deterministic burst. */
export type Random = () => number;

const between = (random: Random, min: number, max: number) =>
    min + random() * (max - min);

export function createParticle(cannon: Cannon, random: Random): Particle {
    const angle =
        (between(random, cannon.angle[0], cannon.angle[1]) * Math.PI) / 180;
    const speed = between(random, cannon.speed[0], cannon.speed[1]);
    const width = between(random, 7, 12);

    return {
        x: cannon.x + cannon.aim * between(random, 0, cannon.spread),
        y: cannon.y + between(random, 0, 20),
        vx: Math.cos(angle) * speed * cannon.aim,
        vy: -Math.sin(angle) * speed,
        width,
        height: width * between(random, 0.5, 1.4),
        color: cannon.colors[
            Math.floor(between(random, 0, cannon.colors.length))
        ]!,
        tilt: between(random, 0, Math.PI * 2),
        tiltSpeed: between(random, -9, 9),
        wobble: between(random, 0, Math.PI * 2),
        wobbleSpeed: between(random, 5, 9),
        life: 0,
        // Long enough that the fade is a safety net for stragglers: most pieces
        // leave through the bottom of the viewport well before this.
        maxLife: between(random, 3.5, 5),
    };
}

export function createBurst(
    cannons: Cannon[],
    countPerCannon: number,
    random: Random,
): Particle[] {
    return cannons.flatMap((cannon) =>
        Array.from({ length: countPerCannon }, () =>
            createParticle(cannon, random),
        ),
    );
}

/** Advances one piece by `dt` seconds, in place. */
export function stepParticle(
    p: Particle,
    dt: number,
    options: ConfettiOptions = defaultConfettiOptions,
): Particle {
    let remaining = dt;
    while (remaining > 1e-9) {
        const h = Math.min(FIXED_STEP, remaining);
        remaining -= h;

        // Drag is expressed per 60fps frame, so it is raised to the number of
        // such frames this step covers.
        const drag = Math.pow(1 - options.drag, h * 60);

        p.wobble += p.wobbleSpeed * h;
        p.vx = (p.vx + Math.cos(p.wobble) * options.sway * h) * drag;
        p.vy = (p.vy + options.gravity * h) * drag;
        p.x += p.vx * h;
        p.y += p.vy * h;
        p.tilt += p.tiltSpeed * h;
        p.life += h;
    }

    return p;
}

/** How visible a piece is: it fades out over `fade` seconds once spent. */
export function particleOpacity(
    p: Particle,
    options: ConfettiOptions = defaultConfettiOptions,
): number {
    const spent = p.life - p.maxLife;
    if (spent <= 0) return 1;
    return Math.max(0, 1 - spent / options.fade);
}

/** A piece is done once it has fallen past the viewport or faded out. */
export function isSpent(
    p: Particle,
    viewportHeight: number,
    options: ConfettiOptions = defaultConfettiOptions,
): boolean {
    return p.y - p.height > viewportHeight || particleOpacity(p, options) === 0;
}

/**
 * Advances the whole burst, dropping the pieces that are done. Mutates and
 * returns `particles` so the render loop allocates nothing per frame.
 */
export function stepBurst(
    particles: Particle[],
    dt: number,
    viewportHeight: number,
    options: ConfettiOptions = defaultConfettiOptions,
): Particle[] {
    for (let i = particles.length - 1; i >= 0; i--) {
        const p = stepParticle(particles[i]!, dt, options);
        if (isSpent(p, viewportHeight, options)) {
            particles.splice(i, 1);
        }
    }
    return particles;
}
