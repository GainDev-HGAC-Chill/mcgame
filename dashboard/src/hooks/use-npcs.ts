"use client"

import { useEffect, useState } from "react"

export class NpcApiError extends Error {
    constructor(
        public status: number,
        message: string,
        public code?: string,
        public issues?: unknown
    ) {
        super(message)
    }
}

export async function npcRequest<T>(
    url: string,
    init?: RequestInit
): Promise<T> {
    const response = await fetch(url, {
        credentials: "same-origin",
        cache: "no-store",
        ...init,
        headers: { "Content-Type": "application/json", ...init?.headers },
    })
    const body = await response.json().catch(() => ({}))
    if (!response.ok)
        throw new NpcApiError(
            response.status,
            body.error?.message ||
                body.message ||
                `Yêu cầu thất bại (${response.status})`,
            body.error?.code,
            body.error?.issues
        )
    return body.data ?? body
}

export function useNpcResource<T>(url: string | null) {
    const [data, setData] = useState<T | null>(null)
    const [error, setError] = useState<Error | null>(null)
    const [loading, setLoading] = useState(false)
    const [revision, setRevision] = useState(0)
    useEffect(() => {
        if (!url) return
        const controller = new AbortController()
        setLoading(true)
        setError(null)
        npcRequest<T>(url, { signal: controller.signal })
            .then(value => {
                if (!controller.signal.aborted) setData(value)
            })
            .catch(failure => {
                if (!controller.signal.aborted)
                    setError(
                        failure instanceof Error
                            ? failure
                            : new Error("Không tải được dữ liệu")
                    )
            })
            .finally(() => {
                if (!controller.signal.aborted) setLoading(false)
            })
        return () => controller.abort()
    }, [url, revision])
    return {
        data,
        error,
        loading,
        reload: () => setRevision(value => value + 1),
    }
}
