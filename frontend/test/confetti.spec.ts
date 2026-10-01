import { describe, expect, it } from "vitest";
import type { Cannon, Particle } from "~/utils/confetti";
import {
    createBurst,
    createCannon,
    createParticle,
    defaultConfettiOptions,
    isSpent,
    particleOpacity,
    stepBurst,
    stepParticle,
} from "~/utils/confetti";

const cannon = (overrides: Partial<Cannon> = {}): Cannon => ({
    ...createCannon(0, 1000, 1, ["#ff0000", "#00ff00"]),
    ...overrides,
});

/** A fixed sequence, so a burst is reproducible. */
function sequence(values: number[]) {
    let i = 0;
    return () => values[i++ % values.length]!;
}

function particle(overrides: Partial<Particle> = {}): Particle {
    return {
        x: 0,
        y: 0,
        vx: 0,
        vy: 0,
        width: 10,
        height: 10,
        color: "#fff",
        tilt: 0,
        tiltSpeed: 0,
        wobble: 0,
        wobbleSpeed: 0,
        life: 0,
        maxLife: 3,
        ...overrides,
    };
}

/** Steps a piece at 60fps and reports how high above its start it reached. */
function peakRise(p: Particle, seconds = 4): number {
    const startY = p.y;
    let highest = p.y;
    for (let t = 0; t < seconds * 60; t++) {
        stepParticle(p, 1 / 60);
        highest = Math.min(highest, p.y);
    }
    return startY - highest;
}

describe("createParticle", () => {
    it("fires up and in the direction of the cannon's aim", () => {
        const right = createParticle(cannon({ aim: 1 }), () => 0.5);
        expect(right.vx).toBeGreaterThan(0);
        expect(right.vy).toBeLessThan(0);

        const left = createParticle(cannon({ x: 1920, aim: -1 }), () => 0.5);
        expect(left.vx).toBeLessThan(0);
        expect(left.vy).toBeLessThan(0);
    });

    it("scatters pieces across the mouth of the cannon, never behind it", () => {
        const low = createParticle(cannon(), () => 0);
        const high = createParticle(cannon(), () => 0.999);

        expect(low.x).toBeCloseTo(0, 5);
        expect(high.x).toBeGreaterThan(0);
        expect(high.x).toBeLessThanOrEqual(140);
    });

    it("is deterministic for a given random source", () => {
        const values = [0.1, 0.9, 0.3, 0.7, 0.2, 0.8, 0.4, 0.6, 0.5, 0.05];
        const a = createParticle(cannon(), sequence(values));
        const b = createParticle(cannon(), sequence(values));
        expect(a).toEqual(b);
    });
});

describe("createBurst", () => {
    it("spawns the requested count from every cannon", () => {
        const burst = createBurst(
            [cannon({ aim: 1 }), cannon({ x: 1920, aim: -1 })],
            25,
            Math.random,
        );

        expect(burst).toHaveLength(50);
        expect(burst.filter((p) => p.vx > 0)).toHaveLength(25);
        expect(burst.filter((p) => p.vx < 0)).toHaveLength(25);
    });
});

describe("stepParticle", () => {
    it("pulls a resting piece downwards", () => {
        const p = stepParticle(particle(), 1 / 60);

        expect(p.vy).toBeGreaterThan(0);
        expect(p.y).toBeGreaterThan(0);
    });

    it("sheds horizontal speed to drag", () => {
        const p = stepParticle(particle({ vx: 1000 }), 1 / 60);

        expect(p.vx).toBeLessThan(1000);
        expect(p.vx).toBeGreaterThan(0);
    });

    it("reaches terminal velocity rather than accelerating forever", () => {
        const p = particle();
        for (let i = 0; i < 600; i++) stepParticle(p, 1 / 60);
        const terminal = p.vy;

        for (let i = 0; i < 600; i++) stepParticle(p, 1 / 60);
        expect(p.vy).toBeCloseTo(terminal, 1);
    });

    it("is frame-rate independent: 30fps and 120fps land in the same place", () => {
        const slow = particle({ vx: 400, vy: -1200, wobbleSpeed: 7 });
        const fast = particle({ vx: 400, vy: -1200, wobbleSpeed: 7 });

        for (let i = 0; i < 2 * 30; i++) stepParticle(slow, 1 / 30);
        for (let i = 0; i < 2 * 120; i++) stepParticle(fast, 1 / 120);

        // The fixed internal step makes these the same path, not merely a
        // similar one: a slow frame is just more sub-steps.
        expect(slow.y).toBeCloseTo(fast.y, 6);
        expect(slow.x).toBeCloseTo(fast.x, 6);
    });

    it("integrates a long frame as several short ones, not one big jump", () => {
        const stepped = particle({ vy: -1200 });
        const jumped = particle({ vy: -1200 });

        stepParticle(stepped, 1 / 30);
        // Half the drag a single 1/30 step would apply, if it were applied once.
        stepParticle(jumped, 1 / 60);
        stepParticle(jumped, 1 / 60);

        expect(stepped.y).toBeCloseTo(jumped.y, 6);
    });

    it("flutters sideways even when launched straight up", () => {
        const p = particle({ vy: -1500, wobbleSpeed: 7 });
        for (let i = 0; i < 60; i++) stepParticle(p, 1 / 60);

        expect(Math.abs(p.x)).toBeGreaterThan(0);
    });

    it("carries the burst well up the screen before it falls back", () => {
        // A 1080px-tall viewport is the case to clear: the strongest pieces
        // should reach the top of it and the weakest still clear a dialog,
        // or the confetti is never really seen.
        const strongest = peakRise(createParticle(cannon(), () => 0.999));
        const weakest = peakRise(createParticle(cannon(), () => 0));

        expect(strongest).toBeGreaterThan(1000);
        expect(weakest).toBeGreaterThan(400);
    });

    it("settles into a floaty terminal velocity, not a stone's", () => {
        const p = particle();
        for (let i = 0; i < 600; i++) stepParticle(p, 1 / 60);

        expect(p.vy).toBeGreaterThan(400);
        expect(p.vy).toBeLessThan(800);
    });
});

describe("particleOpacity", () => {
    it("is solid while the piece is alive and fades once it is spent", () => {
        expect(particleOpacity(particle({ life: 1, maxLife: 3 }))).toBe(1);
        expect(particleOpacity(particle({ life: 3, maxLife: 3 }))).toBe(1);
        expect(
            particleOpacity(particle({ life: 3.3, maxLife: 3 })),
        ).toBeCloseTo(0.5, 1);
        expect(particleOpacity(particle({ life: 9, maxLife: 3 }))).toBe(0);
    });
});

describe("isSpent", () => {
    it("drops pieces that have fallen past the viewport", () => {
        expect(isSpent(particle({ y: 500 }), 1000)).toBe(false);
        expect(isSpent(particle({ y: 1020 }), 1000)).toBe(true);
    });

    it("drops pieces that have faded out on screen", () => {
        const faded = particle({
            y: 100,
            life: 3 + defaultConfettiOptions.fade,
            maxLife: 3,
        });

        expect(isSpent(faded, 1000)).toBe(true);
    });
});

describe("stepBurst", () => {
    it("empties itself, so the render loop always terminates", () => {
        const particles = createBurst([cannon()], 50, Math.random);

        for (let t = 0; t < 15 * 60 && particles.length; t++) {
            stepBurst(particles, 1 / 60, 1000);
        }

        expect(particles).toHaveLength(0);
    });

    it("keeps pieces that are still in flight", () => {
        const particles = [particle({ vy: -1500 }), particle({ y: 2000 })];

        stepBurst(particles, 1 / 60, 1000);

        expect(particles).toHaveLength(1);
        expect(particles[0]!.vy).toBeLessThan(0);
    });
});
