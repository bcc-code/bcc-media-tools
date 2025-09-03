interface Tool {
	label: string;
	icon: string;
	description: string;
	to: string;
	enabled?: boolean;
}

export function useTools() {
	const { t } = useI18n();
	const { me } = useMe();
	const route = useRoute()

	const tools = computed<Tool[]>(() => [
		{
			label: t("tools.bmmUpload.title"),
			icon: "tabler:upload",
			description: t("tools.bmmUpload.description"),
			to: "/upload/bmm/",
		},
		{
			label: t("tools.transcription.title"),
			icon: "tabler:edit",
			description: t("tools.transcription.description"),
			to: "/transcription/",
		},
		{
			label: 'Export',
			icon: "tabler:file-export",
			description: 'Trigger export workflows',
			to: "/export/",
		},
		{
			label: 'Shorts generation',
			icon: "tabler:device-mobile",
			description: 'Generate shorts from existing videos',
			to: "/shorts/",
		},
		{
			label: t("tools.admin.title"),
			icon: "tabler:settings",
			description: t("tools.admin.description"),
			to: "/admin/",
			enabled: me.value?.admin,
		},
	]);


	const enabledTools = computed(() =>
		tools.value.filter((t) => t.enabled != false),
	);

	const currentTool = computed(() =>
		tools.value.find((t) => route.path.startsWith(t.to)),
	);

	return { tools, enabledTools, currentTool };
}
