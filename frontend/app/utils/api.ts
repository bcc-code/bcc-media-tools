import { createClient, type Client } from "@connectrpc/connect";
import { APIService } from "~~/src/gen/api/v1/api_connect";
import { createConnectTransport } from "@connectrpc/connect-web";

let client: Client<typeof APIService>;

export function useAPI() {
    if (client) {
        return client;
    }

    const runtimeConfig = useRuntimeConfig();
    const transport = createConnectTransport({
        baseUrl: runtimeConfig.public.grpcUrl,
    });

    client = createClient(APIService, transport);
    return client;
}
