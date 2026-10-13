"use client"

import Image from "next/image"

import { Badge } from "@/components/ui/badge"
import {
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import { Separator } from "@/components/ui/separator"
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger,
} from "@/components/ui/tooltip"
import type { ItemBrowserRow } from "@/services/item.service"

import {
    getBindTypeLabel,
    getKindLabel,
    getPropTypeLabel,
    getTemplateTableLabel,
    getTradableLabel,
    getUseTypeLabel,
} from "../_lib/shared"

type SendItemSidebarProps = {
    item: ItemBrowserRow | null
    supportsEquipmentOptions: boolean
    itemTypeLabel: string
}

function formatTemplateExpiry(value: number): string {
    if (value <= 0) {
        return "Vĩnh viễn"
    }

    const raw = String(value)
    if (/^\d{12}$/.test(raw)) {
        return `${raw.slice(6, 8)}/${raw.slice(4, 6)}/${raw.slice(0, 4)} ${raw.slice(8, 10)}:${raw.slice(10, 12)}`
    }

    if (/^\d{14}$/.test(raw)) {
        return `${raw.slice(6, 8)}/${raw.slice(4, 6)}/${raw.slice(0, 4)} ${raw.slice(8, 10)}:${raw.slice(10, 12)}:${raw.slice(12, 14)}`
    }

    return `${value} min`
}

export function SendItemSidebar({
    item,
    supportsEquipmentOptions,
    itemTypeLabel,
}: SendItemSidebarProps) {
    if (!item) {
        return null
    }

    return (
        <>
            <DialogHeader className="gap-2 text-left">
                <DialogTitle>Gửi vật phẩm cho người chơi</DialogTitle>
                <DialogDescription>{`Item #${item.itemId} - ${item.name}`}</DialogDescription>
            </DialogHeader>

            <div className="flex items-start gap-4 rounded-xl border bg-background p-4">
                {item.iconDataUrl ? (
                    <Image
                        className="size-20 rounded-lg object-cover"
                        src={item.iconDataUrl}
                        alt={`icon-${item.iconId}`}
                        width={80}
                        height={80}
                        unoptimized
                    />
                ) : (
                    <div className="flex size-20 items-center justify-center rounded-lg border bg-muted text-xs text-muted-foreground">
                        Không có biểu tượng
                    </div>
                )}

                <div className="flex min-w-0 flex-1 flex-col gap-3">
                    <div className="flex flex-wrap gap-2">
                        <Badge variant="outline">#{item.itemId}</Badge>
                        <Badge variant="secondary">{itemTypeLabel}</Badge>
                        {item.requiredLevel > 0 ? (
                            <Badge variant="outline">
                                Cấp {item.requiredLevel}
                            </Badge>
                        ) : null}
                    </div>

                    <div>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <p className="truncate text-base font-semibold">
                                    {item.name}
                                </p>
                            </TooltipTrigger>
                            <TooltipContent>{item.name}</TooltipContent>
                        </Tooltip>
                        <p className="mt-1 text-sm text-muted-foreground">
                            {item.description ||
                                "Vật phẩm này chưa có mô tả."}
                        </p>
                    </div>
                </div>
            </div>

            <div className="grid gap-3 rounded-xl border bg-background p-4 text-sm">
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Loại</span>
                    <span className="font-medium">{itemTypeLabel}</span>
                </div>
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Bảng nguồn</span>
                    <span className="font-medium">{`${getTemplateTableLabel(item.templateTableId, item.templateTableName)} (TBL ${item.templateTableId})`}</span>
                </div>
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Cộng dồn tối đa</span>
                    <span className="font-medium">{item.maxStack}</span>
                </div>
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Mẫu</span>
                    <span className="font-medium">{`Type ${item.templateType} | ${getKindLabel(item.kind)}`}</span>
                </div>
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Cách sử dụng</span>
                    <span className="font-medium">
                        {getUseTypeLabel(item.useType)}
                    </span>
                </div>
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Khóa</span>
                    <span className="font-medium">
                        {getBindTypeLabel(item.bindType)}
                    </span>
                </div>
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Giao dịch</span>
                    <span className="font-medium">
                        {getTradableLabel(item.tradable)}
                    </span>
                </div>
                <div className="flex items-center justify-between gap-4">
                    <span className="text-muted-foreground">Cấp vật phẩm</span>
                    <span className="font-medium">
                        {item.templateLevel || "-"}
                    </span>
                </div>
                {supportsEquipmentOptions ? (
                    <>
                        <div className="flex items-center justify-between gap-4">
                            <span className="text-muted-foreground">
                                Ô khảm mặc định
                            </span>
                            <span className="font-medium">
                                {item.meta.socketCount}
                            </span>
                        </div>
                        <div className="flex items-center justify-between gap-4">
                            <span className="text-muted-foreground">
                                Thuộc tính khóa mặc định
                            </span>
                            <span className="font-medium">
                                {item.meta.bindPropNum}
                            </span>
                        </div>
                        <div className="flex items-center justify-between gap-4">
                            <span className="text-muted-foreground">
                                Độ bền mặc định
                            </span>
                            <span className="font-medium">
                                {item.meta.endureMax || "-"}
                            </span>
                        </div>
                        <div className="flex items-center justify-between gap-4">
                            <span className="text-muted-foreground">
                                ID bộ trang bị
                            </span>
                            <span className="font-medium">
                                {item.meta.setId || "-"}
                            </span>
                        </div>
                        <div className="flex items-center justify-between gap-4">
                            <span className="text-muted-foreground">
                                Vị trí trang bị
                            </span>
                            <span className="font-medium">
                                {item.meta.equipPosition || "-"}
                            </span>
                        </div>
                        <div className="flex items-center justify-between gap-4">
                            <span className="text-muted-foreground">
                                Thời hạn theo mẫu
                            </span>
                            <span className="font-medium">
                                {formatTemplateExpiry(item.meta.expireMinutes)}
                            </span>
                        </div>
                    </>
                ) : null}
            </div>

            {supportsEquipmentOptions ? (
                <>
                    <Separator />

                    <div className="flex flex-col gap-4">
                        <div className="rounded-xl border border-dashed bg-background p-4 text-sm text-muted-foreground">
                            Vật phẩm được gửi qua API quản trị máy chủ. Nhân vật trực tuyến trong bản đồ
                            nhận ngay `onAddCharactorSlot`; nếu ngoại tuyến, vật phẩm được lưu vào
                            cơ sở dữ liệu và nhận khi đăng nhập lại.
                        </div>

                        <div className="flex flex-col gap-2">
                            <p className="text-sm font-medium">
                                Thuộc tính mẫu
                            </p>
                            {item.meta.stats.length > 0 ? (
                                <div className="flex flex-col gap-2 text-sm">
                                    {item.meta.stats.map(stat => (
                                        <div
                                            key={`${stat.type}-${stat.value}`}
                                            className="flex items-center justify-between gap-4 rounded-lg border bg-background px-3 py-2"
                                        >
                                            <span className="text-muted-foreground">
                                                {getPropTypeLabel(stat.type)}
                                            </span>
                                            <span className="font-medium">
                                                {stat.value.toLocaleString(
                                                    "vi-VN"
                                                )}
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            ) : (
                                <div className="rounded-lg border border-dashed bg-background px-3 py-4 text-sm text-muted-foreground">
                                    Mẫu này không có dòng thuộc tính sẵn.
                                </div>
                            )}
                        </div>
                    </div>
                </>
            ) : (
                <div className="rounded-xl border border-dashed bg-background p-4 text-sm text-muted-foreground">
                    Vật phẩm được gửi theo mẫu, ưu tiên cộng dồn vào ô hiện có
                    nếu cùng vật phẩm và trạng thái khóa/độ bền.
                </div>
            )}
        </>
    )
}
