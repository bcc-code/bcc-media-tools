import type { Track } from "~/utils/bmm";
import { Auth } from "./auth";

const config = useRuntimeConfig();
const auth = new Auth(
    `https://${config.api.auth0.domain}/oauth/token`,
    config.api.bmm.audience,
    config.api.auth0.clientId,
    config.api.auth0.clientSecret,
);

export const getFraKaareTracks = async () => {
    const token = await auth.getToken();
    console.log(token);
    const tracks = await $fetch(
        "https://bmm-api.brunstad.org/track?tags=fra-kaare&size=100",
        {
            method: "GET",
            headers: {
                Authorization: token,
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
