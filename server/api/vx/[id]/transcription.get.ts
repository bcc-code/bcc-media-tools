export default defineEventHandler(async (event) => {
    const id = getRouterParam(event, "id");

    const config = useRuntimeConfig().api.cantemo;

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

    return {
        transcription,
    };
});
