import type { Track } from "~/utils/bmm";

export const getFraKaareTracks = async () => {
    const tracks = await $fetch(
        "https://bmm-api.brunstad.org/track?tags=fra-kaare&size=100",
        {
            method: "GET",
            headers: {
                Authorization: "Bearer " + useRuntimeConfig().api.bmm.token,
                "Accept-Language": "nb",
            },
        },
    );

    return (tracks as Track[]).map(
        (i) =>
            ({
                id: i.id,
                title: i.title,
                type: "track",
                published_at: i.published_at,
            }) as Track,
    );
};
