export default defineEventHandler(async (event) => {
    const id = getRouterParam(event, "id");

    const config = useRuntimeConfig().api.cantemo;

    const url = `${config.baseUrl}/API/v2/items/${id}/`;

    const result = await fetch(url, {
        method: "GET",
        headers: {
            "AUTH-TOKEN": config.authToken,
            Accept: "application/json",
        },
    });

    const data = await result.json();

    const formats = await (
        await fetch(`${config.baseUrl}/API/v2/items/${id}/formats/`, {
            method: "GET",
            headers: {
                "AUTH-TOKEN": config.authToken,
                Accept: "application/json",
            },
        })
    ).json();

    let transcription: any = null;
    for (const format of formats.formats) {
        if (format.name === "transcription_json") {
            transcription = await (
                await fetch(`${config.baseUrl}${format.download_uri}`, {
                    method: "GET",
                    headers: {
                        "AUTH-TOKEN": config.authToken,
                        Accept: "application/json",
                    },
                })
            ).json();
            break;
        }
    }

    //mediabanken.brunstad.tv/vs/item/download/VX-477493/?shape=VX-959065

    let video: any = null;

    for (const shape of data.previews.shapes) {
        if (shape.displayname === "lowres") {
            video = "https://mediabanken.brunstad.tv" + shape.uri;
            break;
        }
    }

    return {
        transcription,
        video,
    };
});
