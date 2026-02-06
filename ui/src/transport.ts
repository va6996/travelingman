import { createConnectTransport } from "@connectrpc/connect-web";

const baseUrl = import.meta.env.DEV ? "http://localhost:8000" : "";

export const transport = createConnectTransport({
    baseUrl,
});
