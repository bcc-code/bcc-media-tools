import { describe, expect, it } from "vitest";
import { cropGuideBox } from "~/utils/shortsCrop";

/** The crop keeps `height * 9/16` of the width, so for 16:9 that is 81/256. */
const SIXTEEN_NINE_FRACTION = (9 / 16) * (9 / 16);

describe("cropGuideBox", () => {
    it("keeps a 9:16 column centred in a 16:9 source", () => {
        const guide = cropGuideBox(1600, 900, 1920, 1080);

        expect(guide).toBeDefined();
        expect(guide!.crop.width).toBeCloseTo(1600 * SIXTEEN_NINE_FRACTION, 5);
        expect(guide!.crop.height).toBeCloseTo(900, 5);
        expect(guide!.crop.top).toBeCloseTo(0, 5);

        // Centred: the discarded bands either side are equal.
        const right = 1600 - (guide!.crop.left + guide!.crop.width);
        expect(guide!.crop.left).toBeCloseTo(right, 5);
    });

    it("measures against the picture, not the element, when letterboxed", () => {
        // A 4:3 source inside a 16:9 box leaves pillarbox bars either side.
        const guide = cropGuideBox(1600, 900, 1440, 1080);

        expect(guide).toBeDefined();
        // Picture is 900 * 4/3 = 1200 wide, centred in 1600.
        expect(guide!.picture.width).toBeCloseTo(1200, 5);
        expect(guide!.picture.left).toBeCloseTo(200, 5);
        expect(guide!.picture.height).toBeCloseTo(900, 5);

        // Crop is 1080 * 9/16 = 607.5 of 1440 source px, scaled to the picture.
        expect(guide!.crop.width).toBeCloseTo(1200 * (607.5 / 1440), 5);
        // And it sits inside the picture, not the element.
        expect(guide!.crop.left).toBeGreaterThanOrEqual(guide!.picture.left);
    });

    it("letterboxes top and bottom for a source wider than the box", () => {
        // 2.39:1 scope inside a 16:9 box.
        const guide = cropGuideBox(1600, 900, 2390, 1000);

        expect(guide).toBeDefined();
        expect(guide!.picture.width).toBeCloseTo(1600, 5);
        expect(guide!.picture.height).toBeCloseTo(1600 / 2.39, 5);
        expect(guide!.picture.top).toBeGreaterThan(0);
        expect(guide!.crop.top).toBeCloseTo(guide!.picture.top, 5);
        expect(guide!.crop.height).toBeCloseTo(guide!.picture.height, 5);
    });

    it("keeps a source already narrower than 9:16 whole", () => {
        const guide = cropGuideBox(1600, 900, 720, 1920);

        expect(guide).toBeDefined();
        expect(guide!.crop.width).toBeCloseTo(guide!.picture.width, 5);
        expect(guide!.crop.left).toBeCloseTo(guide!.picture.left, 5);
    });

    it("keeps an exactly 9:16 source whole", () => {
        const guide = cropGuideBox(1600, 900, 1080, 1920);

        expect(guide!.crop.width).toBeCloseTo(guide!.picture.width, 5);
    });

    it("never reaches outside the picture", () => {
        const guide = cropGuideBox(1600, 900, 1920, 1080);

        expect(guide!.crop.left).toBeGreaterThanOrEqual(guide!.picture.left);
        expect(guide!.crop.left + guide!.crop.width).toBeLessThanOrEqual(
            guide!.picture.left + guide!.picture.width + 1e-9,
        );
    });

    it("is undefined before anything has been measured", () => {
        expect(cropGuideBox(0, 0, 0, 0)).toBeUndefined();
        expect(cropGuideBox(1600, 900, 0, 0)).toBeUndefined();
        expect(cropGuideBox(0, 0, 1920, 1080)).toBeUndefined();
    });
});
