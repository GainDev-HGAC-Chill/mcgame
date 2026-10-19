"use client"

import Link from "next/link"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import type { DraftShopSlot } from "@/lib/npcs/editor"
import type { NpcDetail, NpcRecord } from "@/types/npcs"
import { CatalogLookup } from "./catalog-lookup"
import { ShopSlotEditor } from "./shop-slot-editor"

function QuestLinks({
    title,
    rows,
    dialogue,
}: {
    title: string
    rows: NpcDetail["offeredQuests"]
    dialogue: "start_text" | "complete_text"
}) {
    return (
        <section className="space-y-2">
            <h3 className="font-semibold">
                {title} ({rows.length})
            </h3>
            {rows.length ? (
                <div className="overflow-x-auto rounded-md border">
                    <table className="w-full text-sm">
                        <thead>
                            <tr className="border-b text-left">
                                <th className="p-2">ID</th>
                                <th>Tên</th>
                                <th>Loại</th>
                                <th>Cấp</th>
                                <th>Đối thoại</th>
                            </tr>
                        </thead>
                        <tbody>
                            {rows.map(row => (
                                <tr className="border-b align-top" key={row.id}>
                                    <td className="p-2">
                                        <Link
                                            className="text-primary underline"
                                            href={`/dashboard/quests/${row.id}`}
                                        >
                                            {row.id}
                                        </Link>
                                    </td>
                                    <td>{row.name || "Không tên"}</td>
                                    <td>{row.type}</td>
                                    <td>
                                        {row.min_level}–{row.max_level}
                                    </td>
                                    <td className="max-w-xl whitespace-pre-wrap py-2">
                                        {row[dialogue] || "—"}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            ) : (
                <p className="text-sm text-muted-foreground">Không có.</p>
            )}
        </section>
    )
}

export function NpcTabs({
    detail,
    npc,
    setNpc,
    slots,
    setSlots,
    disabled,
}: {
    detail: NpcDetail
    npc: NpcRecord
    setNpc: (npc: NpcRecord) => void
    slots: DraftShopSlot[]
    setSlots: (slots: DraftShopSlot[]) => void
    disabled: boolean
}) {
    return (
        <Tabs defaultValue="npc" className="space-y-4">
            <TabsList>
                <TabsTrigger value="npc">NPC & đối thoại</TabsTrigger>
                <TabsTrigger value="quests">Nhiệm vụ liên kết</TabsTrigger>
                <TabsTrigger value="shop">Cửa hàng</TabsTrigger>
            </TabsList>
            <TabsContent
                value="npc"
                className="space-y-4 rounded-lg border p-4"
            >
                <div className="grid gap-4 md:grid-cols-2">
                    <label className="grid gap-2 text-sm">
                        Tên NPC
                        <Input
                            value={npc.name ?? ""}
                            disabled={disabled}
                            onChange={event =>
                                setNpc({ ...npc, name: event.target.value })
                            }
                        />
                    </label>
                    <CatalogLookup
                        kind="shop"
                        label="Shop chuẩn"
                        value={npc.shop_id}
                        disabled={disabled}
                        allowClear
                        onChange={value => setNpc({ ...npc, shop_id: value })}
                    />
                </div>
                <label className="grid gap-2 text-sm">
                    Đối thoại ambient (`onServiceText`)
                    <Textarea
                        rows={7}
                        value={npc.on_service_text ?? ""}
                        disabled={disabled}
                        onChange={event =>
                            setNpc({
                                ...npc,
                                on_service_text: event.target.value || null,
                            })
                        }
                    />
                </label>
                <div className="grid gap-2 rounded-md bg-muted p-3 text-sm md:grid-cols-4">
                    <span>ID: {npc.id}</span>
                    <span>Map: {npc.pos_map_id ?? "—"}</span>
                    <span>
                        Vị trí: {npc.pos_x}, {npc.pos_y}
                    </span>
                    <span>Loại: {npc.type}</span>
                </div>
                {npc.shop_id !== detail.npc.shop_id && (
                    <p className="text-sm text-amber-700">
                        Hãy lưu shop assignment trước; tab cửa hàng chỉ nạp slot
                        của shop trong revision hiện tại.
                    </p>
                )}
            </TabsContent>
            <TabsContent
                value="quests"
                className="space-y-5 rounded-lg border p-4"
            >
                <p className="rounded-md border border-amber-500 p-3 text-sm">
                    Quan hệ nhiệm vụ chỉ đọc. Chọn ID để sửa bằng Quest Admin,
                    nơi giữ revision/audit riêng. Dashboard không tự sửa tiến độ
                    người chơi.
                </p>
                <QuestLinks
                    title="Nhiệm vụ bắt đầu"
                    rows={detail.offeredQuests}
                    dialogue="start_text"
                />
                <QuestLinks
                    title="Nhiệm vụ trả"
                    rows={detail.turnInQuests}
                    dialogue="complete_text"
                />
                <section className="space-y-2">
                    <h3 className="font-semibold">
                        Vòng lặp ({detail.loops.length})
                    </h3>
                    {detail.loops.map(loop => (
                        <p className="text-sm" key={loop.id}>
                            #{loop.id} · {loop.name || "Không tên"} · cấp{" "}
                            {loop.min_level}–{loop.max_level}
                        </p>
                    ))}
                </section>
                <p className="text-sm">
                    Legacy `qid`: {detail.legacyQid.raw || "trống"}
                    {detail.legacyQid.missingIds.length > 0 &&
                        ` · thiếu quest: ${detail.legacyQid.missingIds.join(", ")}`}
                </p>
            </TabsContent>
            <TabsContent
                value="shop"
                className="space-y-4 rounded-lg border p-4"
            >
                {detail.shop ? (
                    <ShopSlotEditor
                        shop={detail.shop}
                        slots={slots}
                        setSlots={setSlots}
                        disabled={disabled}
                    />
                ) : (
                    <p>
                        NPC chưa gắn shop chuẩn đã tồn tại. Gắn ID shop ở tab
                        NPC rồi lưu trước khi sửa slot.
                    </p>
                )}
            </TabsContent>
        </Tabs>
    )
}
