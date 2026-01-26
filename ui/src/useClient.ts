import { useMemo } from "react";
import { ServiceType } from "@bufbuild/protobuf";
import { PromiseClient, createPromiseClient } from "@connectrpc/connect";
import { transport } from "./transport";

export function useClient<T extends ServiceType>(service: T): PromiseClient<T> {
    return useMemo(() => createPromiseClient(service, transport), [service]);
}
