export default defineEventHandler(async (event) => {
    const id = getRouterParam(event, "id");

    const config = useRuntimeConfig().api.cantemo;

    const metadata = await (
        await fetch(`${config.baseUrl}/API/v2/items/${id}/`, {
            method: "GET",
            headers: {
                "AUTH-TOKEN": config.authToken,
                Accept: "application/json",
            },
        })
    ).json();

    let video: any = null;

    for (const shape of metadata.previews.shapes) {
        if (shape.displayname === "lowres") {
            video = "https://mediabanken.brunstad.tv" + shape.uri;
            break;
        }
    }

    return {
        video,
        filename: metadata.metadata_summary.filename,
    };
});
