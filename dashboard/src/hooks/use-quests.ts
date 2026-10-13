"use client"

import { useEffect, useState } from "react"

export class QuestApiError extends Error {
    constructor(
        public status: number,
        message: string,
        public code?: string,
        public issues?: unknown
    ) {
        super(message)
    }
}
export async function questRequest<T>(
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
        throw new QuestApiError(
            response.status,
            typeof body.error === "string"
                ? body.error
                : body.error?.message ||
                      body.message ||
                      `Yêu cầu thất bại (${response.status})`,
            body.error?.code,
            body.error?.issues
        )
    return body.data ?? body
}
export function useQuestResource<T>(url: string | null) {
    const [data, setData] = useState<T | null>(null)
    const [error, setError] = useState<Error | null>(null)
    const [loading, setLoading] = useState(false)
    const [revision, setRevision] = useState(0)
    useEffect(() => {
        if (!url) {
            setData(null)
            setError(null)
            setLoading(false)
            return
        }
        const controller = new AbortController()
        setLoading(true)
        setError(null)
        setData(null)
        questRequest<T>(url, { signal: controller.signal })
            .then(value => {
                if (!controller.signal.aborted) setData(value)
            })
            .catch(error => {
                if (!controller.signal.aborted) setError(error)
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
