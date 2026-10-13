"use client"

import Link from "next/link"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useQuestResource } from "@/hooks/use-quests"
import { ApiError, EnumField } from "./_components/fields"
import { questTypes, text, type RecordRow } from "./_lib/editor"

export default function QuestListPage() {
    const [search, setSearch] = useState("")
    const [query, setQuery] = useState("")
    const [type, setType] = useState("")
    const [level, setLevel] = useState("")
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
    if (type) params.set("type", type)
    if (level) {
        params.set("minLevel", level)
        params.set("maxLevel", level)
    }
    const { data, error, loading, reload } = useQuestResource<{
        items: RecordRow[]
        total: number
        canWrite?: boolean
    }>(`/api/quests?${params}`)
    return (
        <div className="space-y-6 p-6">
            <div className="flex justify-between">
                <div>
                    <h1 className="text-2xl font-bold">Quản lý nhiệm vụ</h1>
                    <p className="text-muted-foreground">
                        Danh mục nhiệm vụ · không sửa tiến độ người chơi
                    </p>
                </div>
                <Button variant="outline" onClick={reload}>
                    Tải lại
                </Button>
            </div>
            <p className="rounded-lg border border-amber-500 bg-amber-500/10 p-4 text-sm">
                Lưu chỉ thay đổi cơ sở dữ liệu, chưa kích hoạt đồng bộ trong
                game. Backend cần nạp lại; xuất bản danh mục Flash đang bị chặn
                do chưa có đường nạp được xác minh. Đăng nhập lại không đảm bảo
                cập nhật danh mục nhúng.
            </p>
            <div className="grid items-end gap-4 md:grid-cols-3">
                <Input
                    aria-label="Tìm nhiệm vụ"
                    placeholder="Tên nhiệm vụ hoặc ID…"
                    value={search}
                    onChange={event => setSearch(event.target.value)}
                />
                <EnumField
                    label="Loại nhiệm vụ"
                    value={type}
                    options={[["", "Tất cả"], ...questTypes]}
                    onChange={value => {
                        setType(value)
                        setPage(1)
                    }}
                />
                <Input
                    aria-label="Cấp nhân vật (khoảng cấp nhiệm vụ)"
                    inputMode="numeric"
                    placeholder="Lọc khoảng cấp nhiệm vụ…"
                    value={level}
                    onChange={event => {
                        setLevel(event.target.value)
                        setPage(1)
                    }}
                />
            </div>
            {error && <ApiError error={error} />}
            {loading && <p role="status">Đang tải nhiệm vụ…</p>}
            {data && (
                <>
                    <p className="text-sm">
                        {data.total} nhiệm vụ
                        {data.canWrite === false && " · Chỉ đọc"}
                    </p>
                    <div className="overflow-x-auto rounded-lg border">
                        <table className="w-full text-sm">
                            <thead>
                                <tr className="border-b text-left">
                                    <th className="p-3">ID</th>
                                    <th>Tên</th>
                                    <th>Loại</th>
                                    <th>Cấp min–max</th>
                                </tr>
                            </thead>
                            <tbody>
                                {data.items.map(quest => (
                                    <tr key={quest.id} className="border-b">
                                        <td className="p-3">{quest.id}</td>
                                        <td>
                                            <Link
                                                className="font-medium text-primary underline"
                                                href={`/dashboard/quests/${quest.id}`}
                                            >
                                                {text(quest.name) ||
                                                    "Không tên"}
                                            </Link>
                                        </td>
                                        <td>
                                            {questTypes.find(
                                                ([id]) =>
                                                    String(id) ===
                                                    text(quest.type)
                                            )?.[1] ||
                                                `Loại cũ ${text(quest.type)}`}
                                        </td>
                                        <td>
                                            {text(quest.min_level)}–
                                            {text(quest.max_level)}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                    {!data.items.length && <p>Không tìm thấy nhiệm vụ.</p>}
                    <div className="flex items-center gap-4">
                        <Button
                            variant="outline"
                            disabled={page <= 1 || loading}
                            onClick={() => setPage(page - 1)}
                        >
                            Trước
                        </Button>
                        <span>
                            Trang {page} /{" "}
                            {Math.max(1, Math.ceil(data.total / 25))}
                        </span>
                        <Button
                            variant="outline"
                            disabled={page * 25 >= data.total || loading}
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
