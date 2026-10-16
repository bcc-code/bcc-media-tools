import { describe, expect, it } from "vitest";
import { bestOffset, describeOffset } from "~/utils/serverClock";

describe("bestOffset", () => {
    it("takes the midpoint of the quickest round trip", () => {
        expect(
            bestOffset([
                { sentAt: 1000, receivedAt: 1400, serverMs: 4300 }, // slow
                { sentAt: 2000, receivedAt: 2040, serverMs: 5020 }, // quick
                { sentAt: 3000, receivedAt: 3200, serverMs: 6150 },
            ]),
        ).toEqual({ offsetMs: 3000, uncertaintyMs: 20 });
    });

    it("is negative when this computer is ahead", () => {
        expect(
            bestOffset([
                { sentAt: 10_000, receivedAt: 10_010, serverMs: 6_005 },
            ])?.offsetMs,
        ).toBe(-4000);
    });

    it("has nothing to say without samples", () => {
        expect(bestOffset([])).toBeUndefined();
        // a sample that came back before it was sent is nonsense
        expect(
            bestOffset([{ sentAt: 5, receivedAt: 4, serverMs: 0 }]),
        ).toBeUndefined();
    });
});

describe("describeOffset", () => {
    it("says which way and by how much", () => {
        expect(describeOffset(-3210)).toEqual({ ahead: true, seconds: "3.2" });
        expect(describeOffset(1500)).toEqual({ ahead: false, seconds: "1.5" });
    });
});
