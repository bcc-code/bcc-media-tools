interface Options {
    togglePlay: () => void;
    forward: () => void;
    backward: () => void;
    setStartPoint?: () => void;
    setEndPoint?: () => void;
    addClip?: () => void;
}

/**
 * Whether the keystroke was aimed at somewhere text is being typed.
 *
 * Every shortcut here is a bare letter or space, so without this a space in the
 * transcript search box plays the video and an "o" sets the out point.
 */
function isTypingTarget(event: KeyboardEvent): boolean {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return false;
    if (target.isContentEditable) return true;
    return ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName);
}

function onVideoKey(
    keys: string | string[],
    handler: (() => void) | undefined,
) {
    if (!handler) return;
    onKeyStroke(keys, (event) => {
        if (isTypingTarget(event)) return;
        handler();
    });
}

export function useVideoKeyboardControls(options: Options) {
    onVideoKey(" ", options.togglePlay);
    onVideoKey("ArrowRight", options.forward);
    onVideoKey("ArrowLeft", options.backward);
    onVideoKey(["i", "I"], options.setStartPoint);
    onVideoKey(["o", "O"], options.setEndPoint);
    onVideoKey(["a", "A"], options.addClip);
}
