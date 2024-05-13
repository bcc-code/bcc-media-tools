import { createPromiseClient, type PromiseClient } from "@connectrpc/connect";
import { APIService } from "~/src/gen/api/v1/api_connect";
import { createConnectTransport } from "@connectrpc/connect-web";

let client : PromiseClient<typeof APIService>;

export function useAPI() {
    if (client) {
        return client;
    }

    const runtimeConfig = useRuntimeConfig();
    const transport = createConnectTransport({
        baseUrl:  runtimeConfig.public.grpcURL,
    });

   client = createPromiseClient(APIService, transport);
   return client;
}
