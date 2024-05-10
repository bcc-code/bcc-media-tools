import {createPromiseClient} from "@connectrpc/connect";
import {APIService} from "~/src/gen/api/v1/api_connect";
import { createConnectTransport } from "@connectrpc/connect-web";

// This transport is going to be used throughout the app
const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
});
const client = createPromiseClient(APIService, transport);

export function useAPI() {
    return client;
}