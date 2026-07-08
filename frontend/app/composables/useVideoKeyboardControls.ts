interface Options {
    togglePlay: () => void;
    forward: () => void;
    backward: () => void;
    setStartPoint?: () => void;
    setEndPoint?: () => void;
}

export function useVideoKeyboardControls(options: Options) {
    onKeyStroke(" ", options.togglePlay);
    onKeyStroke("ArrowRight", options.forward);
    onKeyStroke("ArrowLeft", options.backward);
    if (options.setStartPoint) {
        onKeyStroke(["i", "I"], options.setStartPoint);
    }
    if (options.setEndPoint) {
        onKeyStroke(["o", "O"], options.setEndPoint);
    }
}
