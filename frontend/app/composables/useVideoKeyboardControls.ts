

interface Options {
	togglePlay: () => void;
	forward: () => void;
	backward: () => void;
}

export function useVideoKeyboardControls(options: Options) {
	onKeyStroke(' ', options.togglePlay)
	onKeyStroke('ArrowRight', options.forward)
	onKeyStroke('ArrowLeft', options.backward)
}