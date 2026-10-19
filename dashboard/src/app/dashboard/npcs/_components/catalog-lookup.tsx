"use client"

import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useNpcResource } from "@/hooks/use-npcs"
import type { NpcLookupItem, NpcLookupResponse } from "@/types/npcs"

export function CatalogLookup({
    kind,
    label,
    value,
    disabled,
    allowClear = false,
    onChange,
    onSelect,
}: {
    kind: "shop" | "item" | "equipment" | "map"
    label: string
    value: number | null
    disabled?: boolean
    allowClear?: boolean
    onChange: (value: number | null) => void
    onSelect?: (item: NpcLookupItem) => void
}) {
    const [search, setSearch] = useState("")
    const [query, setQuery] = useState("")
    useEffect(() => {
        const timer = setTimeout(() => setQuery(search.trim()), 250)
        return () => clearTimeout(timer)
    }, [search])
    const params = new URLSearchParams({
        kind,
        search: query,
        page: "1",
        pageSize: "8",
    })
    const resource = useNpcResource<NpcLookupResponse>(
        query ? `/api/npcs/lookups?${params}` : null
    )
    return (
        <div className="grid gap-2 text-sm">
            <label className="grid gap-1">
                {label}
                <Input
                    type="number"
                    min={allowClear ? 0 : 1}
                    value={value ?? 0}
                    disabled={disabled}
                    onChange={event => {
                        const next = Number(event.target.value)
                        onChange(allowClear && next === 0 ? null : next)
                    }}
                />
            </label>
            <Input
                aria-label={`Tìm ${label}`}
                placeholder="Tìm theo tên hoặc ID…"
                value={search}
                disabled={disabled}
                onChange={event => setSearch(event.target.value)}
            />
            {query && resource.loading && (
                <span className="text-xs text-muted-foreground">Đang tìm…</span>
            )}
            {query && resource.error && (
                <span className="text-xs text-destructive">
                    {resource.error.message}
                </span>
            )}
            {query && resource.data && (
                <div className="flex max-h-28 flex-wrap gap-1 overflow-y-auto">
                    {resource.data.items.map(item => (
                        <Button
                            key={item.id}
                            type="button"
                            size="sm"
                            variant={item.id === value ? "default" : "outline"}
                            disabled={disabled}
                            title={item.name ?? undefined}
                            onClick={() => {
                                onChange(item.id)
                                onSelect?.(item)
                                setSearch("")
                                setQuery("")
                            }}
                        >
                            #{item.id} {item.name || "Không tên"}
                        </Button>
                    ))}
                    {!resource.data.items.length && (
                        <span className="text-xs text-muted-foreground">
                            Không tìm thấy.
                        </span>
                    )}
                </div>
            )}
        </div>
    )
}
