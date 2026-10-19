"use client"

import Image from "next/image"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import type { DraftShopSlot } from "@/lib/npcs/editor"

export function ShopItemIcon({ row }: { row: DraftShopSlot }) {
    const [failed, setFailed] = useState(false)
    useEffect(() => setFailed(false), [row.iconCode])
    const name =
        row.displayName ||
        `${row.type === 19 ? "Trang bị" : "Vật phẩm"} #${row.item_id}`
    return (
        <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-lg border-2 border-muted bg-muted/40 shadow-inner">
            {row.iconCode > 0 && !failed ? (
                <Image
                    src={`/api/icons/${row.iconCode}`}
                    alt={name}
                    width={72}
                    height={72}
                    unoptimized
                    className="h-[72px] w-[72px] object-contain"
                    onError={() => setFailed(true)}
                />
            ) : (
                <span className="px-2 text-center text-xs text-muted-foreground">
                    Chưa có ảnh
                </span>
            )}
        </div>
    )
}

function priceLines(row: DraftShopSlot): string[] {
    const lines: string[] = []
    if (row.money > 0)
        lines.push(
            `${row.money.toLocaleString("vi-VN")} bạc${row.gt === 1 ? " thường" : " (loại đang chọn)"}`
        )
    if (row.gold > 0)
        lines.push(
            `${row.gold.toLocaleString("vi-VN")} vàng${row.gt === 1 ? " thường" : " (loại đang chọn)"}`
        )
    if (row.p_type1 > 0)
        lines.push(
            `${row.p_num1.toLocaleString("vi-VN")} × tiền #${row.p_type1}`
        )
    if (row.p_type2 > 0)
        lines.push(
            `${row.p_num2.toLocaleString("vi-VN")} × tiền #${row.p_type2}`
        )
    return lines.length ? lines : ["Chưa đặt giá"]
}

export function ShopSlotCard({
    row,
    selected,
    disabled,
    onSelect,
    onToggleHide,
}: {
    row: DraftShopSlot
    selected: boolean
    disabled: boolean
    onSelect: () => void
    onToggleHide: () => void
}) {
    const legacy = ![0, 5].includes(row.st)
    const hidden = row._hide || row.st === 5
    const name =
        row.displayName ||
        `${row.type === 19 ? "Trang bị" : "Vật phẩm"} #${row.item_id}`
    return (
        <article
            className={`flex min-h-72 flex-col rounded-xl border bg-card p-4 shadow-sm transition ${selected ? "border-primary ring-2 ring-primary/20" : "hover:border-primary/50"} ${hidden ? "opacity-60" : ""}`}
        >
            <div className="mb-3 flex items-center justify-between gap-2 text-xs">
                <span className="rounded-full bg-muted px-2 py-1 font-medium">
                    Vị trí {row.position}
                </span>
                <div className="flex gap-1">
                    {row._new && (
                        <span className="rounded-full bg-blue-500/15 px-2 py-1 text-blue-700">
                            Mới
                        </span>
                    )}
                    {legacy && (
                        <span className="rounded-full bg-amber-500/15 px-2 py-1 text-amber-700">
                            Legacy {row.st}
                        </span>
                    )}
                    {hidden && (
                        <span className="rounded-full bg-slate-500/15 px-2 py-1">
                            Đang ẩn
                        </span>
                    )}
                </div>
            </div>
            <div className="flex gap-3">
                <ShopItemIcon row={row} />
                <div className="min-w-0">
                    <h4
                        className="line-clamp-2 min-h-10 font-semibold leading-5"
                        title={name}
                    >
                        {name}
                    </h4>
                    <p className="mt-1 text-xs text-muted-foreground">
                        Template #{row.item_id}
                    </p>
                    <p className="text-xs text-muted-foreground">
                        Slot #{row.id > 0 ? row.id : "mới"} ·{" "}
                        {row.type === 19 ? "Trang bị" : "Vật phẩm"}
                    </p>
                </div>
            </div>
            <div className="mt-4 border-t pt-3">
                <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    Giá bán
                </p>
                {priceLines(row).map(line => (
                    <p
                        key={line}
                        className={
                            line === "Chưa đặt giá"
                                ? "text-sm text-destructive"
                                : "text-sm font-medium"
                        }
                    >
                        {line}
                    </p>
                ))}
            </div>
            <div className="mt-auto flex gap-2 pt-4">
                <Button
                    type="button"
                    className="flex-1"
                    variant={selected ? "default" : "outline"}
                    onClick={onSelect}
                >
                    {selected ? "Đóng sửa" : "Sửa ô"}
                </Button>
                <Button
                    type="button"
                    variant="outline"
                    disabled={disabled || legacy}
                    onClick={onToggleHide}
                >
                    {row._new
                        ? "Bỏ"
                        : row._hide
                          ? "Hoàn tác"
                          : row.st === 5
                            ? "Bật lại"
                            : "Ẩn"}
                </Button>
            </div>
        </article>
    )
}
