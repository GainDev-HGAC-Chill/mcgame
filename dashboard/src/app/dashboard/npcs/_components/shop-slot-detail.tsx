"use client"

import { Input } from "@/components/ui/input"
import { EQUIPMENT_QUALITY_OPTIONS } from "@/lib/equipment-quality"
import type { DraftShopSlot } from "@/lib/npcs/editor"
import type { NpcLookupItem, ShopSlotInsert } from "@/types/npcs"
import { CatalogLookup } from "./catalog-lookup"
import { ShopItemIcon } from "./shop-slot-card"

function NumberField({
    label,
    value,
    disabled,
    onChange,
}: {
    label: string
    value: number | null
    disabled?: boolean
    onChange: (value: number) => void
}) {
    return (
        <label className="grid gap-1 text-xs font-medium text-muted-foreground">
            {label}
            <Input
                type="number"
                value={value ?? 0}
                disabled={disabled}
                onChange={event => onChange(Number(event.target.value))}
            />
        </label>
    )
}

export function ShopSlotDetail({
    row,
    disabled,
    editSlot,
    changeTemplateID,
    changeType,
    selectTemplate,
}: {
    row: DraftShopSlot
    disabled: boolean
    editSlot: (
        key: string,
        field: keyof ShopSlotInsert,
        value: number | null
    ) => void
    changeTemplateID: (key: string, itemID: number) => void
    changeType: (key: string, slotType: 19 | 29) => void
    selectTemplate: (key: string, item: NpcLookupItem) => void
}) {
    return (
        <section className="rounded-xl border border-primary/40 bg-card p-5 shadow-sm">
            <div className="mb-5 flex items-center gap-4">
                <ShopItemIcon row={row} />
                <div>
                    <h3 className="text-lg font-semibold">
                        Sửa {row.displayName || `template #${row.item_id}`}
                    </h3>
                    <p className="text-sm text-muted-foreground">
                        Ô vị trí {row.position} · mọi thay đổi chỉ được ghi khi
                        bấm “Lưu NPC”.
                    </p>
                </div>
            </div>
            <div className="grid gap-6 xl:grid-cols-3">
                <div className="space-y-3">
                    <h4 className="font-semibold">Vật phẩm hiển thị</h4>
                    <label className="grid gap-1 text-xs font-medium text-muted-foreground">
                        Loại catalog
                        <select
                            className="rounded-md border bg-background px-3 py-2 text-sm"
                            value={row.type}
                            disabled={disabled}
                            onChange={event =>
                                changeType(
                                    row._key,
                                    Number(event.target.value) as 19 | 29
                                )
                            }
                        >
                            <option value={29}>Vật phẩm</option>
                            <option value={19}>Trang bị</option>
                        </select>
                    </label>
                    <CatalogLookup
                        kind={row.type === 19 ? "equipment" : "item"}
                        label="Template"
                        value={row.item_id}
                        disabled={disabled}
                        onChange={value =>
                            changeTemplateID(row._key, value ?? 0)
                        }
                        onSelect={item => selectTemplate(row._key, item)}
                    />
                    <div className="grid grid-cols-2 gap-3">
                        {row._new && (
                            <NumberField
                                label="Vị trí"
                                value={row.position}
                                disabled={disabled}
                                onChange={value =>
                                    editSlot(row._key, "position", value)
                                }
                            />
                        )}
                        <NumberField
                            label="Giới hạn mua"
                            value={row.amount}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "amount", value)
                            }
                        />
                        <label className="grid gap-1 text-xs font-medium text-muted-foreground">
                            Phẩm chất trang bị
                            <select
                                className="h-10 rounded-md border bg-background px-3 text-sm text-foreground"
                                value={row.quality}
                                disabled={disabled}
                                onChange={event =>
                                    editSlot(
                                        row._key,
                                        "quality",
                                        Number(event.target.value)
                                    )
                                }
                            >
                                {EQUIPMENT_QUALITY_OPTIONS.map(option => (
                                    <option
                                        key={option.quality}
                                        value={option.quality}
                                    >
                                        {option.label}
                                    </option>
                                ))}
                            </select>
                            <span className="font-normal">
                                0 phát trang bị Trắng và không dùng màu mặc định
                                của template.
                            </span>
                        </label>
                    </div>
                </div>
                <div className="space-y-3">
                    <h4 className="font-semibold">Giá bán</h4>
                    <div className="grid grid-cols-2 gap-3">
                        <NumberField
                            label="Bạc"
                            value={row.money}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "money", value)
                            }
                        />
                        <NumberField
                            label="Vàng"
                            value={row.gold}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "gold", value)
                            }
                        />
                        <NumberField
                            label="Loại tiền 1"
                            value={row.p_type1}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "p_type1", value)
                            }
                        />
                        <NumberField
                            label="Số tiền 1"
                            value={row.p_num1}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "p_num1", value)
                            }
                        />
                        <NumberField
                            label="Loại tiền 2"
                            value={row.p_type2}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "p_type2", value)
                            }
                        />
                        <NumberField
                            label="Số tiền 2"
                            value={row.p_num2}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "p_num2", value)
                            }
                        />
                    </div>
                    <label className="grid gap-1 text-xs font-medium text-muted-foreground">
                        Kiểu tiền thường/khóa
                        <select
                            className="rounded-md border bg-background px-3 py-2 text-sm"
                            value={row.gt}
                            disabled={disabled}
                            onChange={event =>
                                editSlot(
                                    row._key,
                                    "gt",
                                    Number(event.target.value)
                                )
                            }
                        >
                            <option value={0}>Ưu tiên loại đang chọn</option>
                            <option value={1}>Tiền thường</option>
                        </select>
                    </label>
                </div>
                <div className="space-y-3">
                    <h4 className="font-semibold">Thiết lập ô</h4>
                    <div className="grid grid-cols-2 gap-3">
                        <NumberField
                            label="Tổng giá"
                            value={row.price_all}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "price_all", value)
                            }
                        />
                        <NumberField
                            label="Sale"
                            value={row.sale}
                            disabled={disabled}
                            onChange={value =>
                                editSlot(row._key, "sale", value)
                            }
                        />
                        <NumberField
                            label="R"
                            value={row.r}
                            disabled={disabled}
                            onChange={value => editSlot(row._key, "r", value)}
                        />
                    </div>
                    <label className="grid gap-1 text-xs font-medium text-muted-foreground">
                        Trạng thái
                        <select
                            className="rounded-md border bg-background px-3 py-2 text-sm"
                            value={row.st}
                            disabled={disabled}
                            onChange={event =>
                                editSlot(
                                    row._key,
                                    "st",
                                    Number(event.target.value)
                                )
                            }
                        >
                            <option value={0}>Đang bán</option>
                            <option value={5}>Đang ẩn</option>
                        </select>
                    </label>
                    <p className="rounded-md bg-muted p-3 text-xs text-muted-foreground">
                        Giá `point` bị khóa ở 0 vì luồng mua NPC hiện không trừ
                        loại giá này. ID shop và vị trí của ô cũ là bất biến.
                    </p>
                </div>
            </div>
        </section>
    )
}
