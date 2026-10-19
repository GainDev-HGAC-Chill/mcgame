"use client"

import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { newShopSlot, type DraftShopSlot } from "@/lib/npcs/editor"
import type { NpcLookupItem, NpcShopDetail, ShopSlotInsert } from "@/types/npcs"
import { ShopSlotCard } from "./shop-slot-card"
import { ShopSlotDetail } from "./shop-slot-detail"

export function ShopSlotEditor({
    shop,
    slots,
    setSlots,
    disabled,
}: {
    shop: NpcShopDetail
    slots: DraftShopSlot[]
    setSlots: (slots: DraftShopSlot[]) => void
    disabled: boolean
}) {
    const [selectedKey, setSelectedKey] = useState<string | null>(null)
    const selected = slots.find(row => row._key === selectedKey) ?? null
    useEffect(() => {
        if (selectedKey && !slots.some(row => row._key === selectedKey))
            setSelectedKey(null)
    }, [selectedKey, slots])

    function editSlot(
        key: string,
        field: keyof ShopSlotInsert,
        value: number | null
    ) {
        setSlots(
            slots.map(row =>
                row._key === key ? { ...row, [field]: value } : row
            )
        )
    }
    function selectTemplate(key: string, item: NpcLookupItem) {
        setSlots(
            slots.map(row =>
                row._key === key
                    ? {
                          ...row,
                          item_id: item.id,
                          displayName: item.name,
                          iconCode: item.iconCode,
                      }
                    : row
            )
        )
    }
    function changeTemplateID(key: string, itemID: number) {
        setSlots(
            slots.map(row =>
                row._key === key
                    ? {
                          ...row,
                          item_id: itemID,
                          displayName: null,
                          iconCode: 0,
                      }
                    : row
            )
        )
    }
    function changeType(key: string, slotType: 19 | 29) {
        setSlots(
            slots.map(row =>
                row._key === key
                    ? {
                          ...row,
                          type: slotType,
                          item_id: 1,
                          displayName: null,
                          iconCode: 0,
                      }
                    : row
            )
        )
    }
    function toggleHide(row: DraftShopSlot) {
        if (row._new) {
            setSlots(slots.filter(item => item._key !== row._key))
            return
        }
        if (row.st === 5 && !row._hide) {
            setSlots(
                slots.map(item =>
                    item._key === row._key ? { ...item, st: 0 } : item
                )
            )
            return
        }
        setSlots(
            slots.map(item =>
                item._key === row._key ? { ...item, _hide: !item._hide } : item
            )
        )
    }

    const nextPosition =
        Math.max(
            0,
            ...slots
                .filter(row => !row._hide && row.st !== 5)
                .map(row => row.position)
        ) + 1
    const activeCount = slots.filter(row => !row._hide && row.st === 0).length
    return (
        <div className="space-y-5">
            <div className="flex flex-wrap items-start justify-between gap-3 rounded-lg bg-muted/40 p-4">
                <div>
                    <h3 className="font-semibold">
                        Shop #{shop.container.id} ·{" "}
                        {shop.container.name || "Không tên"}
                    </h3>
                    <p className="text-sm text-muted-foreground">
                        {activeCount} ô hoạt động / {slots.length} ô · dùng
                        chung bởi{" "}
                        {shop.referencingNpcs
                            .map(row => `#${row.id} ${row.name || ""}`)
                            .join(", ")}
                    </p>
                </div>
                <Button
                    type="button"
                    disabled={disabled}
                    onClick={() => {
                        const row = newShopSlot(shop.container.id, nextPosition)
                        setSlots([...slots, row])
                        setSelectedKey(row._key)
                    }}
                >
                    + Thêm ô hàng
                </Button>
            </div>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                {slots.map(row => (
                    <ShopSlotCard
                        key={row._key}
                        row={row}
                        selected={selectedKey === row._key}
                        disabled={disabled}
                        onSelect={() =>
                            setSelectedKey(
                                selectedKey === row._key ? null : row._key
                            )
                        }
                        onToggleHide={() => toggleHide(row)}
                    />
                ))}
            </div>
            {selected && (
                <ShopSlotDetail
                    row={selected}
                    disabled={
                        disabled ||
                        selected._hide ||
                        ![0, 5].includes(selected.st)
                    }
                    editSlot={editSlot}
                    changeTemplateID={changeTemplateID}
                    changeType={changeType}
                    selectTemplate={selectTemplate}
                />
            )}
            <p className="text-xs text-muted-foreground">
                Tên và ảnh lấy trực tiếp từ catalog hiển thị trong game. Vị trí
                ô cũ không sửa được vì Flash bỏ qua cập nhật trường này; thao
                tác Ẩn dùng `st=5`, không xóa dữ liệu.
            </p>
        </div>
    )
}
