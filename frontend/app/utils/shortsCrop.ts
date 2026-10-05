export type Box = {
    left: number;
    top: number;
    width: number;
    height: number;
};

export type CropGuide = {
    /** The picture itself, which is letterboxed inside the video element. */
    picture: Box;
    /** The 9:16 column the generated short would keep. */
    crop: Box;
};

/** Portrait shorts are 9:16. */
const CROP_RATIO = 9 / 16;

/**
 * Where the generated short's crop falls on the preview, in element pixels.
 *
 * Mirrors the default crop in bcc-media-flows (`buildCropFilter` with no
 * keyframes): a column `height * 9/16` wide, the full height of the frame,
 * centred horizontally. It is only the *default* — when the shorts service
 * returns keyframes the real crop tracks the speaker and pans, which no preview
 * can know in advance, so this is a guide and not a promise.
 *
 * A `<video>` letterboxes its picture to preserve aspect, so the guide is
 * measured against the picture rather than against the element box. Returns
 * undefined until both have been measured.
 */
export function cropGuideBox(
    elementWidth: number,
    elementHeight: number,
    videoWidth: number,
    videoHeight: number,
): CropGuide | undefined {
    if (elementWidth <= 0 || elementHeight <= 0) return undefined;
    if (videoWidth <= 0 || videoHeight <= 0) return undefined;

    const ratio = videoWidth / videoHeight;
    const pictureWidth = Math.min(elementWidth, elementHeight * ratio);
    const pictureHeight = Math.min(elementHeight, elementWidth / ratio);

    const picture: Box = {
        left: (elementWidth - pictureWidth) / 2,
        top: (elementHeight - pictureHeight) / 2,
        width: pictureWidth,
        height: pictureHeight,
    };

    // A source already as narrow as 9:16, or narrower, is kept whole.
    const cropFraction = Math.min(1, (videoHeight * CROP_RATIO) / videoWidth);
    const cropWidth = pictureWidth * cropFraction;

    return {
        picture,
        crop: {
            left: picture.left + (pictureWidth - cropWidth) / 2,
            top: picture.top,
            width: cropWidth,
            height: pictureHeight,
        },
    };
}
