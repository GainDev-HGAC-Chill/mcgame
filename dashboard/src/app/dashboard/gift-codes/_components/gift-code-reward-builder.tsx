"use client"

import * as React from "react"
import { Loader2, PackageSearch, Plus, Search, Trash2 } from "lucide-react"
import Image from "next/image"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectLabel,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger,
} from "@/components/ui/tooltip"
import { itemService, type ItemBrowserRow } from "@/services/item.service"
import {
    GIFT_CODE_REWARD_OPTIONS,
    NUMERIC_GIFT_CODE_REWARD_OPTIONS,
    type GiftCodeRewardDraft,
    type GiftCodeRewardType,
    getRewardOption,
    isNumericRewardType,
    summarizeRewardDraft,
} from "../_lib/rewards"

type GiftCodeRewardBuilderProps = {
    rewards: GiftCodeRewardDraft[]
    disabled: boolean
    onAddReward: (type: GiftCodeRewardType) => void
    onRewardChange: (
        rewardId: string,
        patch: Partial<GiftCodeRewardDraft>
    ) => void
    onRewardTypeChange: (rewardId: string, type: GiftCodeRewardType) => void
    onRemoveReward: (rewardId: string) => void
}

export function GiftCodeRewardBuilder({
    rewards,
    disabled,
    onAddReward,
    onRewardChange,
    onRewardTypeChange,
    onRemoveReward,
}: GiftCodeRewardBuilderProps) {
    const [loadingRewardId, setLoadingRewardId] = React.useState<string | null>(
        null
    )
    const [itemResultsByRewardId, setItemResultsByRewardId] = React.useState<
        Record<string, ItemBrowserRow[]>
    >({})

    const searchItems = React.useCallback(
        async (reward: GiftCodeRewardDraft) => {
            const query = reward.itemSearch.trim() || reward.itemId.trim()
            if (!query) {
                toast.error("Nhập tên hoặc ID vật phẩm trước khi tìm.")
                return
            }

            setLoadingRewardId(reward.id)
            try {
                const response = await itemService.search({
                    search: query,
                    page: 1,
                    pageSize: 6,
                    sortBy: "item_id",
                    sortDir: "asc",
                })
                setItemResultsByRewardId(prev => ({
                    ...prev,
                    [reward.id]: Array.isArray(response.items)
                        ? response.items
                        : [],
                }))
            } catch (error) {
                const message =
                    error instanceof Error ? error.message : "Không tìm thấy vật phẩm"
                toast.error(message)
            } finally {
                setLoadingRewardId(current =>
                    current === reward.id ? null : current
                )
            }
        },
        []
    )

    const applySelectedItem = React.useCallback(
        (rewardId: string, item: ItemBrowserRow) => {
            onRewardChange(rewardId, {
                itemId: String(item.itemId),
                itemName: item.name || `Item #${item.itemId}`,
                itemIconPath: item.iconDataUrl || item.iconPath || "",
                itemSearch: item.name || String(item.itemId),
            })
            setItemResultsByRewardId(prev => ({
                ...prev,
                [rewardId]: [],
            }))
        },
        [onRewardChange]
    )

    return (
        <Card>
            <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="flex flex-col gap-1.5">
                    <CardTitle>Phần thưởng</CardTitle>
                    <CardDescription>
                        Cấu hình từng phần thưởng bằng biểu mẫu. Phần thưởng dạng số cần số lượng; vật phẩm có biểu mẫu riêng.
                    </CardDescription>
                </div>

                <div className="flex flex-wrap gap-2">
                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => onAddReward("gold")}
                        disabled={disabled}
                    >
                        <Plus data-icon="inline-start" />
                        Thêm phần thưởng dạng số
                    </Button>
                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => onAddReward("item")}
                        disabled={disabled}
                    >
                        <PackageSearch data-icon="inline-start" />
                        Thêm vật phẩm
                    </Button>
                </div>
            </CardHeader>

            <CardContent className="flex flex-col gap-4">
                {rewards.length === 0 ? (
                    <div className="rounded-lg border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
                        Chưa có phần thưởng. Bạn có thể thêm phần thưởng dạng số hoặc vật phẩm ở trên.
                    </div>
                ) : null}

                {rewards.map((reward, index) => {
                    const selectedType = getRewardOption(reward.type)
                    const numericSelectedType = isNumericRewardType(reward.type)
                        ? NUMERIC_GIFT_CODE_REWARD_OPTIONS.find(
                              option => option.value === reward.type
                          )
                        : undefined
                    const itemResults = itemResultsByRewardId[reward.id] || []
                    const searching = loadingRewardId === reward.id

                    return (
                        <Card key={reward.id} className="border-dashed">
                            <CardHeader className="gap-3 pb-4">
                                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                                    <div className="flex flex-col gap-2">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <Badge variant="secondary">
                                                Phần thưởng {index + 1}
                                            </Badge>
                                            <Badge variant="outline">
                                                {selectedType?.label ||
                                                    reward.type}
                                            </Badge>
                                            {reward.meta &&
                                            Object.keys(reward.meta).length >
                                                0 ? (
                                                <Badge variant="outline">
                                                    Có dữ liệu bổ sung
                                                </Badge>
                                            ) : null}
                                        </div>
                                        <CardDescription>
                                            {summarizeRewardDraft(reward)}
                                        </CardDescription>
                                    </div>

                                    <Button
                                        type="button"
                                        variant="ghost"
                                        size="sm"
                                        onClick={() =>
                                            onRemoveReward(reward.id)
                                        }
                                        disabled={
                                            disabled || rewards.length <= 1
                                        }
                                    >
                                        <Trash2 data-icon="inline-start" />
                                        Xóa phần thưởng
                                    </Button>
                                </div>
                            </CardHeader>

                            <CardContent className="flex flex-col gap-4">
                                <div className="grid gap-4 md:grid-cols-2">
                                    <div className="flex flex-col gap-2">
                                        <Label>Loại phần thưởng</Label>
                                        <Select
                                            value={reward.type}
                                            onValueChange={value =>
                                                onRewardTypeChange(
                                                    reward.id,
                                                    value as GiftCodeRewardType
                                                )
                                            }
                                            disabled={disabled}
                                        >
                                            <SelectTrigger>
                                                <SelectValue placeholder="Chọn loại phần thưởng" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectGroup>
                                                    <SelectLabel>
                                                        Phần thưởng dạng số
                                                    </SelectLabel>
                                                    {NUMERIC_GIFT_CODE_REWARD_OPTIONS.map(
                                                        option => (
                                                            <SelectItem
                                                                key={
                                                                    option.value
                                                                }
                                                                value={
                                                                    option.value
                                                                }
                                                            >
                                                                {option.label}
                                                            </SelectItem>
                                                        )
                                                    )}
                                                </SelectGroup>
                                                <SelectGroup>
                                                    <SelectLabel>
                                                        Phần thưởng đặc biệt
                                                    </SelectLabel>
                                                    {GIFT_CODE_REWARD_OPTIONS.filter(
                                                        option =>
                                                            !isNumericRewardType(
                                                                option.value
                                                            )
                                                    ).map(option => (
                                                        <SelectItem
                                                            key={option.value}
                                                            value={option.value}
                                                        >
                                                            {option.label}
                                                        </SelectItem>
                                                    ))}
                                                </SelectGroup>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    {isNumericRewardType(reward.type) ? (
                                        <div className="flex flex-col gap-2">
                                            <Label>
                                                {numericSelectedType?.amountLabel ||
                                                    "Amount"}
                                            </Label>
                                            <Input
                                                type="number"
                                                min={0}
                                                value={reward.amount}
                                                onChange={event =>
                                                    onRewardChange(reward.id, {
                                                        amount: event.target
                                                            .value,
                                                    })
                                                }
                                                disabled={disabled}
                                            />
                                        </div>
                                    ) : null}
                                </div>

                                {reward.type === "item" ? (
                                    <div className="flex flex-col gap-4">
                                        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto]">
                                            <div className="flex flex-col gap-2">
                                                <Label>
                                                    Tìm vật phẩm theo tên hoặc ID
                                                </Label>
                                                <Input
                                                    placeholder="Ví dụ: 9020 hoặc tên vật phẩm"
                                                    value={reward.itemSearch}
                                                    onChange={event =>
                                                        onRewardChange(
                                                            reward.id,
                                                            {
                                                                itemSearch:
                                                                    event.target
                                                                        .value,
                                                            }
                                                        )
                                                    }
                                                    disabled={disabled}
                                                />
                                            </div>
                                            <div className="flex items-end">
                                                <Button
                                                    type="button"
                                                    variant="outline"
                                                    onClick={() =>
                                                        void searchItems(reward)
                                                    }
                                                    disabled={
                                                        disabled || searching
                                                    }
                                                >
                                                    {searching ? (
                                                        <Loader2
                                                            className="animate-spin"
                                                            data-icon="inline-start"
                                                        />
                                                    ) : (
                                                        <Search data-icon="inline-start" />
                                                    )}
                                                    Tìm vật phẩm
                                                </Button>
                                            </div>
                                        </div>

                                        <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_160px]">
                                            <div className="flex flex-col gap-2">
                                                <Label>ID vật phẩm</Label>
                                                <Input
                                                    type="number"
                                                    min={1}
                                                    value={reward.itemId}
                                                    onChange={event =>
                                                        onRewardChange(
                                                            reward.id,
                                                            {
                                                                itemId: event
                                                                    .target
                                                                    .value,
                                                            }
                                                        )
                                                    }
                                                    disabled={disabled}
                                                />
                                            </div>
                                            <div className="flex flex-col gap-2">
                                                <Label>Số lượng</Label>
                                                <Input
                                                    type="number"
                                                    min={1}
                                                    value={reward.count}
                                                    onChange={event =>
                                                        onRewardChange(
                                                            reward.id,
                                                            {
                                                                count: event
                                                                    .target
                                                                    .value,
                                                            }
                                                        )
                                                    }
                                                    disabled={disabled}
                                                />
                                            </div>
                                        </div>

                                        <div className="rounded-lg border bg-muted/30 px-4 py-3">
                                            <div className="flex items-start gap-3">
                                                {reward.itemIconPath ? (
                                                    <Image
                                                        src={
                                                            reward.itemIconPath
                                                        }
                                                        alt={
                                                            reward.itemName ||
                                                            `Item ${reward.itemId}`
                                                        }
                                                        width={48}
                                                        height={48}
                                                        unoptimized
                                                        className="size-12 rounded-md border object-contain bg-background"
                                                    />
                                                ) : (
                                                    <div className="flex size-12 items-center justify-center rounded-md border bg-background text-muted-foreground">
                                                        <PackageSearch className="h-5 w-5" />
                                                    </div>
                                                )}
                                                <div className="min-w-0 flex-1">
                                                    <div className="flex flex-wrap items-center gap-2">
                                                        <span className="font-medium">
                                                            Vật phẩm đã chọn
                                                        </span>
                                                        {reward.itemId ? (
                                                            <Badge variant="outline">
                                                                #{reward.itemId}
                                                            </Badge>
                                                        ) : null}
                                                    </div>
                                                    <Tooltip>
                                                        <TooltipTrigger asChild>
                                                            <p className="mt-1 truncate text-sm text-muted-foreground">
                                                                {reward.itemName ||
                                                                    "Chưa chọn vật phẩm."}
                                                            </p>
                                                        </TooltipTrigger>
                                                        <TooltipContent>
                                                            {reward.itemName ||
                                                                "Chưa chọn vật phẩm."}
                                                        </TooltipContent>
                                                    </Tooltip>
                                                </div>
                                            </div>
                                        </div>

                                        {itemResults.length > 0 ? (
                                            <div className="grid gap-3 lg:grid-cols-2">
                                                {itemResults.map(item => (
                                                    <button
                                                        key={`${reward.id}-${item.itemId}`}
                                                        type="button"
                                                        className="flex items-start gap-3 rounded-lg border p-3 text-left transition-colors hover:border-primary hover:bg-muted/30"
                                                        onClick={() =>
                                                            applySelectedItem(
                                                                reward.id,
                                                                item
                                                            )
                                                        }
                                                        disabled={disabled}
                                                    >
                                                        {item.iconDataUrl ||
                                                        item.iconPath ? (
                                                            <Image
                                                                src={
                                                                    item.iconDataUrl ||
                                                                    item.iconPath
                                                                }
                                                                alt={item.name}
                                                                width={48}
                                                                height={48}
                                                                unoptimized
                                                                className="size-12 rounded-md border object-contain bg-background"
                                                            />
                                                        ) : (
                                                            <div className="flex size-12 items-center justify-center rounded-md border bg-background text-muted-foreground">
                                                                <PackageSearch className="h-5 w-5" />
                                                            </div>
                                                        )}
                                                        <div className="min-w-0 flex-1">
                                                            <div className="flex flex-wrap items-center gap-2">
                                                                <Badge variant="secondary">
                                                                    #
                                                                    {
                                                                        item.itemId
                                                                    }
                                                                </Badge>
                                                                {item.requiredLevel >
                                                                0 ? (
                                                                    <Badge variant="outline">
                                                                        Cấp{" "}
                                                                        {
                                                                            item.requiredLevel
                                                                        }
                                                                    </Badge>
                                                                ) : null}
                                                            </div>
                                                            <Tooltip>
                                                                <TooltipTrigger
                                                                    asChild
                                                                >
                                                                    <div className="mt-2 truncate font-medium">
                                                                        {
                                                                            item.name
                                                                        }
                                                                    </div>
                                                                </TooltipTrigger>
                                                                <TooltipContent>
                                                                    {item.name}
                                                                </TooltipContent>
                                                            </Tooltip>
                                                            <p className="mt-1 text-sm text-muted-foreground line-clamp-2">
                                                                {item.description ||
                                                                    "Chưa có mô tả."}
                                                            </p>
                                                        </div>
                                                    </button>
                                                ))}
                                            </div>
                                        ) : null}
                                    </div>
                                ) : null}
                            </CardContent>
                        </Card>
                    )
                })}
            </CardContent>
        </Card>
    )
}
