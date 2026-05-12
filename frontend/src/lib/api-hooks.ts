"use client";

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
  type UseMutationOptions,
} from "@tanstack/react-query";
import { apiFetch, ApiError } from "@/lib/api";

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    total: number;
    limit: number;
    offset: number;
  };
}

export interface SingleResponse<T> {
  data: T;
}

export function useApiQuery<T>(
  key: string[],
  path: string,
  options?: Omit<UseQueryOptions<T, ApiError>, "queryKey" | "queryFn">,
) {
  return useQuery<T, ApiError>({
    queryKey: key,
    queryFn: () => apiFetch<T>(path),
    ...options,
  });
}

export function useApiMutation<TData, TVariables>(
  method: "POST" | "PUT" | "PATCH" | "DELETE",
  pathFn: (variables: TVariables) => string,
  options?: Omit<
    UseMutationOptions<TData, ApiError, TVariables>,
    "mutationFn"
  > & {
    invalidateKeys?: string[][];
    bodyFn?: (variables: TVariables) => unknown;
  },
) {
  const queryClient = useQueryClient();
  const { invalidateKeys, bodyFn, ...mutationOptions } = options || {};

  return useMutation<TData, ApiError, TVariables>({
    mutationFn: (variables: TVariables) =>
      apiFetch<TData>(pathFn(variables), {
        method,
        body:
          method === "DELETE"
            ? undefined
            : bodyFn
              ? bodyFn(variables)
              : variables,
      }),
    onSuccess: (...args) => {
      if (invalidateKeys) {
        invalidateKeys.forEach((key) =>
          queryClient.invalidateQueries({ queryKey: key }),
        );
      }
      mutationOptions.onSuccess?.(...args);
    },
    ...mutationOptions,
  });
}

export { useQueryClient };
