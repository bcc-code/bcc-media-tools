import {createPromiseClient} from "@connectrpc/connect";
import {APIService} from "~/src/gen/api/v1/api_connect";
import { createConnectTransport } from "@connectrpc/connect-web";

const runtimeConfig = useRuntimeConfig();

// This transport is going to be used throughout the app
const transport = createConnectTransport({
    baseUrl:  runtimeConfig.public.grpcURL,
});
const client = createPromiseClient(APIService, transport);

export function useAPI() {
    return client;
}