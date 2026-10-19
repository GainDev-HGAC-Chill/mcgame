"use client"

import Link from "next/link"
import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useNpcResource } from "@/hooks/use-npcs"
import type { NpcListResponse } from "@/types/npcs"

export default function NpcListPage() {
    const [search, setSearch] = useState("")
    const [query, setQuery] = useState("")
    const [mapId, setMapId] = useState("")
    const [type, setType] = useState("")
    const [hasDialogue, setHasDialogue] = useState("")
    const [hasShop, setHasShop] = useState("")
    const [page, setPage] = useState(1)
    useEffect(() => {
        const timer = setTimeout(() => {
            setQuery(search)
            setPage(1)
        }, 300)
        return () => clearTimeout(timer)
    }, [search])
    const params = new URLSearchParams({
        search: query,
        page: String(page),
        pageSize: "25",
    })
    if (mapId) params.set("mapId", mapId)
    if (type) params.set("type", type)
    if (hasDialogue) params.set("hasDialogue", hasDialogue)
    if (hasShop) params.set("hasShop", hasShop)
    const resource = useNpcResource<NpcListResponse>(`/api/npcs?${params}`)
    return (
        <div className="space-y-6 p-6">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold">Quản lý NPC</h1>
                    <p className="text-muted-foreground">
                        Đối thoại, nhiệm vụ liên kết và cửa hàng chuẩn
                    </p>
                </div>
                <Button variant="outline" onClick={resource.reload}>
                    Tải lại
                </Button>
            </div>
            <p className="rounded-lg border border-amber-500 bg-amber-500/10 p-4 text-sm">
                Lưu không tự khởi động lại máy chủ. Liên kết nhiệm vụ chỉ đọc
                tại đây và vẫn chịu giới hạn danh mục Flash.
            </p>
            <div className="grid gap-3 md:grid-cols-3 xl:grid-cols-5">
                <Input
                    aria-label="Tìm NPC"
                    placeholder="Tên hoặc ID…"
                    value={search}
                    onChange={event => setSearch(event.target.value)}
                />
                <Input
                    aria-label="Map ID"
                    inputMode="numeric"
                    placeholder="Map ID…"
                    value={mapId}
                    onChange={event => {
                        setMapId(event.target.value)
                        setPage(1)
                    }}
                />
                <Input
                    aria-label="Loại NPC"
                    inputMode="numeric"
                    placeholder="Loại NPC…"
                    value={type}
                    onChange={event => {
                        setType(event.target.value)
                        setPage(1)
                    }}
                />
                <select
                    className="rounded-md border bg-background px-3 py-2 text-sm"
                    value={hasDialogue}
                    onChange={event => {
                        setHasDialogue(event.target.value)
                        setPage(1)
                    }}
                >
                    <option value="">Mọi đối thoại</option>
                    <option value="true">Có đối thoại</option>
                    <option value="false">Không đối thoại</option>
                </select>
                <select
                    className="rounded-md border bg-background px-3 py-2 text-sm"
                    value={hasShop}
                    onChange={event => {
                        setHasShop(event.target.value)
                        setPage(1)
                    }}
                >
                    <option value="">Mọi cửa hàng</option>
                    <option value="true">Có cửa hàng</option>
                    <option value="false">Không cửa hàng</option>
                </select>
            </div>
            {resource.error && (
                <p role="alert" className="text-destructive">
                    {resource.error.message}
                </p>
            )}
            {resource.loading && <p role="status">Đang tải NPC…</p>}
            {resource.data && (
                <>
                    <p className="text-sm">
                        {resource.data.total} NPC
                        {resource.data.canWrite === false && " · Chỉ đọc"}
                    </p>
                    <div className="overflow-x-auto rounded-lg border">
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="border-b text-left">
                                    <th className="p-3">ID</th>
                                    <th>Tên</th>
                                    <th>Map</th>
                                    <th>Loại</th>
                                    <th>Đối thoại</th>
                                    <th>Nhiệm vụ</th>
                                    <th>Shop</th>
                                </tr>
                            </thead>
                            <tbody>
                                {resource.data.items.map(npc => (
                                    <tr className="border-b" key={npc.id}>
                                        <td className="p-3">{npc.id}</td>
                                        <td>
                                            <Link
                                                className="font-medium text-primary underline"
                                                href={`/dashboard/npcs/${npc.id}`}
                                            >
                                                {npc.name || "Không tên"}
                                            </Link>
                                        </td>
                                        <td>{npc.pos_map_id ?? "—"}</td>
                                        <td>{npc.type}</td>
                                        <td>
                                            {npc.on_service_text?.trim()
                                                ? "Có"
                                                : "—"}
                                        </td>
                                        <td>
                                            {(npc.offered_count ?? 0) +
                                                (npc.turn_in_count ?? 0) +
                                                (npc.loop_count ?? 0)}
                                        </td>
                                        <td>{npc.shop_id || "—"}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                    {!resource.data.items.length && <p>Không tìm thấy NPC.</p>}
                    <div className="flex items-center gap-4">
                        <Button
                            variant="outline"
                            disabled={page <= 1 || resource.loading}
                            onClick={() => setPage(page - 1)}
                        >
                            Trước
                        </Button>
                        <span>
                            Trang {page} /{" "}
                            {Math.max(1, Math.ceil(resource.data.total / 25))}
                        </span>
                        <Button
                            variant="outline"
                            disabled={
                                page * 25 >= resource.data.total ||
                                resource.loading
                            }
                            onClick={() => setPage(page + 1)}
                        >
                            Sau
                        </Button>
                    </div>
                </>
            )}
        </div>
    )
}
