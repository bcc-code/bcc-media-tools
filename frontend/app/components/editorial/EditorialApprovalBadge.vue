<script setup lang="ts">
import type { EditorialApproval } from "~~/src/gen/api/v1/api_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

// One sign-off on a review session: green once approved, grey until then. The
// tooltip says who approved it and when.
const props = defineProps<{
    approval?: EditorialApproval;
    label: string;
}>();

const { t } = useI18n();

const dateTimeFormatter = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Europe/Oslo",
    dateStyle: "medium",
    timeStyle: "short",
});

const approved = computed(() => !!props.approval?.approved);
const detail = computed(() => {
    const a = props.approval;
    if (!a?.approved) return t("editorial.notApproved");
    return t("editorial.approvedBy", {
        name: a.by,
        date: a.at ? dateTimeFormatter.format(timestampDate(a.at)) : "",
    });
});
</script>

<template>
    <DesignTooltip :content="detail">
        <DesignBadge
            :variant="approved ? 'success' : 'neutral'"
            size="large"
            class="shrink-0 gap-1.5"
            :aria-label="`${label}: ${detail}`"
        >
            <Icon
                :name="approved ? 'tabler:check' : 'tabler:clock'"
                class="size-4"
            />
            {{ label }}
        </DesignBadge>
    </DesignTooltip>
</template>
