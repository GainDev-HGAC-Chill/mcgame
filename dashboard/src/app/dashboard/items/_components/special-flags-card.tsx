"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
    Select,
    SelectContent,
    SelectGroup,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"

import {
    ELEMENT_OPTIONS,
    QILING_PROP_OPTIONS,
    buildDefaultPetStoneSlots,
    formatQilingValue,
    getElementLabel,
    getQilingPropLabel,
    getQilingTierMeta,
    type QilingDraftLine,
} from "../_lib/shared"

type SpecialFlagsCardProps = {
    sending: boolean
    supportsSublimationBuilder: boolean
    supportsQilingBuilder: boolean
    supportsPetFlag3: boolean
    sublimeId: string
    sublimeElement: string
    sublimeAdd: string
    qilingLines: QilingDraftLine[]
    petStoneSlots: boolean[]
    petStoneSkillId: string
    generatedFlag: string
    generatedFlag2: string
    generatedFlag3: string
    flagOverride: string
    flag2Override: string
    flag3Override: string
    rawProperties: string
    onSublimeIdChange: (value: string) => void
    onSublimeElementChange: (value: string) => void
    onSublimeAddChange: (value: string) => void
    onQilingLineChange: (
        index: number,
        field: "type" | "value" | "max",
        value: string
    ) => void
    onPetStoneSlotChange: (index: number, value: boolean) => void
    onPetStoneSkillIdChange: (value: string) => void
    onFlagOverrideChange: (value: string) => void
    onFlag2OverrideChange: (value: string) => void
    onFlag3OverrideChange: (value: string) => void
    onRawPropertiesChange: (value: string) => void
}

function buildQilingPreview(line: QilingDraftLine): {
    text: string
    color: string
} {
    const propType = Number.parseInt(line.type || "0", 10)
    const value = Number.parseFloat(line.value || "0")
    const max = Number.parseFloat(line.max || "0")

    if (
        !Number.isFinite(propType) ||
        propType <= 0 ||
        !Number.isFinite(max) ||
        max <= 0
    ) {
        return {
            text: "Để trống",
            color: "#94A3B8",
        }
    }

    const tier = getQilingTierMeta(Number.isFinite(value) ? value : 0, max)
    const fullSuffix = Number.isFinite(value) && value >= max ? " (Day)" : ""

    return {
        text: `${getQilingPropLabel(propType)}: ${formatQilingValue(propType, Number.isFinite(value) ? value : 0)}${fullSuffix}`,
        color: tier.color,
    }
}

function renderGeneratedValue(value: string): string {
    return value || "(khong tao tu form)"
}

export function SpecialFlagsCard({
    sending,
    supportsSublimationBuilder,
    supportsQilingBuilder,
    supportsPetFlag3,
    sublimeId,
    sublimeElement,
    sublimeAdd,
    qilingLines,
    petStoneSlots,
    petStoneSkillId,
    generatedFlag,
    generatedFlag2,
    generatedFlag3,
    flagOverride,
    flag2Override,
    flag3Override,
    rawProperties,
    onSublimeIdChange,
    onSublimeElementChange,
    onSublimeAddChange,
    onQilingLineChange,
    onPetStoneSlotChange,
    onPetStoneSkillIdChange,
    onFlagOverrideChange,
    onFlag2OverrideChange,
    onFlag3OverrideChange,
    onRawPropertiesChange,
}: SpecialFlagsCardProps) {
    const normalizedPetStoneSlots =
        petStoneSlots.length > 0 ? petStoneSlots : buildDefaultPetStoneSlots()

    return (
        <Card>
            <CardHeader>
                <CardTitle>Cờ đặc biệt</CardTitle>
                <p className="text-sm text-muted-foreground">
                    Máy khách đọc `flag` theo nhiều định dạng. Biểu mẫu hỗ trợ thăng hoa, Kỳ Linh
                    và đá thú cưng; phần dưới cho phép ghi đè dữ liệu thô cho trường hợp đặc biệt.
                </p>
            </CardHeader>
            <CardContent className="flex flex-col gap-6">
                <div className="grid gap-6 2xl:grid-cols-2">
                    <div className="flex flex-col gap-4 rounded-xl border p-4">
                        <div className="flex flex-col gap-1">
                            <p className="font-medium">flag / Thăng hoa</p>
                            <p className="text-sm text-muted-foreground">
                                Dùng cho trang bị thường và trang bị thú cưng. Thần khí ở vị trí 15–20
                                có định dạng riêng; hãy dùng ghi đè dữ liệu thô.
                            </p>
                        </div>

                        {supportsSublimationBuilder ? (
                            <>
                                <div className="grid gap-3 md:grid-cols-3">
                                    <div className="flex flex-col gap-2">
                                        <Label htmlFor="send-sublime-id">
                                            Mốc thăng hoa
                                        </Label>
                                        <Input
                                            id="send-sublime-id"
                                            type="number"
                                            min={0}
                                            value={sublimeId}
                                            onChange={event =>
                                                onSublimeIdChange(
                                                    event.target.value
                                                )
                                            }
                                            disabled={sending}
                                        />
                                    </div>

                                    <div className="flex flex-col gap-2">
                                        <Label htmlFor="send-sublime-element">
                                            Nguyên tố
                                        </Label>
                                        <Select
                                            value={sublimeElement || "0"}
                                            onValueChange={
                                                onSublimeElementChange
                                            }
                                            disabled={sending}
                                        >
                                            <SelectTrigger id="send-sublime-element">
                                                <SelectValue placeholder="Chọn nguyên tố" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectGroup>
                                                    {ELEMENT_OPTIONS.map(
                                                        option => (
                                                            <SelectItem
                                                                key={
                                                                    option.value
                                                                }
                                                                value={String(
                                                                    option.value
                                                                )}
                                                            >
                                                                {option.label} (
                                                                {option.value})
                                                            </SelectItem>
                                                        )
                                                    )}
                                                </SelectGroup>
                                            </SelectContent>
                                        </Select>
                                    </div>

                                    <div className="flex flex-col gap-2">
                                        <Label htmlFor="send-sublime-add">
                                            Thưởng tỷ lệ (%)
                                        </Label>
                                        <Input
                                            id="send-sublime-add"
                                            type="number"
                                            min={0}
                                            step="0.001"
                                            value={sublimeAdd}
                                            onChange={event =>
                                                onSublimeAddChange(
                                                    event.target.value
                                                )
                                            }
                                            disabled={sending}
                                        />
                                    </div>
                                </div>

                                <div className="rounded-lg bg-muted/40 p-3 text-sm text-muted-foreground">{`Element hien tai: ${getElementLabel(Number.parseInt(sublimeElement || "0", 10) || 0)} | sublimeAdd duoc client cong vao ti le thanh cong o panel thang hoa.`}</div>
                            </>
                        ) : (
                            <div className="rounded-lg border border-dashed px-3 py-4 text-sm text-muted-foreground">
                                Vật phẩm này dùng định dạng `flag` riêng của thần khí/pháp bảo.
                                Hãy dùng phần ghi đè dữ liệu thô bên dưới.
                            </div>
                        )}

                        <div className="flex flex-col gap-2">
                            <Label htmlFor="generated-flag">
                                Chuỗi dữ liệu được tạo
                            </Label>
                            <Textarea
                                id="generated-flag"
                                value={renderGeneratedValue(generatedFlag)}
                                readOnly
                                className="min-h-[88px] font-mono text-xs"
                            />
                        </div>
                    </div>

                    <div className="flex flex-col gap-4 rounded-xl border p-4">
                        <div className="flex flex-col gap-1">
                            <p className="font-medium">flag2 / Kỳ Linh</p>
                            <p className="text-sm text-muted-foreground">
                                Máy khách đọc 3 dòng `0..2`: `t` là thuộc tính,
                                `v` là giá trị hiện tại, `max` là giới hạn tối đa.
                            </p>
                        </div>

                        {supportsQilingBuilder ? (
                            <div className="flex flex-col gap-3">
                                {qilingLines.map((line, index) => {
                                    const preview = buildQilingPreview(line)
                                    return (
                                        <div
                                            key={line.slot}
                                            className="grid gap-3 rounded-lg border p-3 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)_minmax(0,0.8fr)]"
                                        >
                                            <div className="flex flex-col gap-2">
                                                <Label>{`Dong ${index + 1}`}</Label>
                                                <Select
                                                    value={line.type || "0"}
                                                    onValueChange={value =>
                                                        onQilingLineChange(
                                                            index,
                                                            "type",
                                                            value
                                                        )
                                                    }
                                                    disabled={sending}
                                                >
                                                    <SelectTrigger>
                                                        <SelectValue placeholder="Chọn thuộc tính Kỳ Linh" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        <SelectGroup>
                                                            <SelectItem value="0">
                                                                Để trống
                                                            </SelectItem>
                                                            {QILING_PROP_OPTIONS.map(
                                                                option => (
                                                                    <SelectItem
                                                                        key={
                                                                            option.value
                                                                        }
                                                                        value={String(
                                                                            option.value
                                                                        )}
                                                                    >
                                                                        {
                                                                            option.label
                                                                        }{" "}
                                                                        (
                                                                        {
                                                                            option.value
                                                                        }
                                                                        )
                                                                    </SelectItem>
                                                                )
                                                            )}
                                                        </SelectGroup>
                                                    </SelectContent>
                                                </Select>
                                                <div
                                                    className="rounded-md bg-muted/40 px-3 py-2 text-xs font-medium"
                                                    style={{
                                                        color: preview.color,
                                                    }}
                                                >
                                                    {preview.text}
                                                </div>
                                            </div>

                                            <div className="flex flex-col gap-2">
                                                <Label>Giá trị hiện tại</Label>
                                                <Input
                                                    type="number"
                                                    min={0}
                                                    step="0.001"
                                                    value={line.value}
                                                    onChange={event =>
                                                        onQilingLineChange(
                                                            index,
                                                            "value",
                                                            event.target.value
                                                        )
                                                    }
                                                    disabled={sending}
                                                />
                                            </div>

                                            <div className="flex flex-col gap-2">
                                                <Label>Giới hạn tối đa</Label>
                                                <Input
                                                    type="number"
                                                    min={0}
                                                    step="0.001"
                                                    value={line.max}
                                                    onChange={event =>
                                                        onQilingLineChange(
                                                            index,
                                                            "max",
                                                            event.target.value
                                                        )
                                                    }
                                                    disabled={sending}
                                                />
                                            </div>
                                        </div>
                                    )
                                })}
                            </div>
                        ) : (
                            <div className="rounded-lg border border-dashed px-3 py-4 text-sm text-muted-foreground">
                                Biểu mẫu này chỉ phù hợp với trang bị nhóm 1–4 và cấp yêu cầu trên 150.
                                Vật phẩm khác vẫn có thể dùng ghi đè dữ liệu thô.
                            </div>
                        )}

                        <div className="flex flex-col gap-2">
                            <Label htmlFor="generated-flag-2">
                                Chuỗi dữ liệu được tạo
                            </Label>
                            <Textarea
                                id="generated-flag-2"
                                value={renderGeneratedValue(generatedFlag2)}
                                readOnly
                                className="min-h-[88px] font-mono text-xs"
                            />
                        </div>
                    </div>
                </div>

                <div className="flex flex-col gap-4 rounded-xl border p-4">
                    <div className="flex flex-col gap-1">
                        <p className="font-medium">flag3 / Đá thú cưng</p>
                        <p className="text-sm text-muted-foreground">
                            Chú giải trang bị thú cưng kiểm tra đá ở ô 1–6. Nếu có `flag3[1][2]`,
                            kỹ năng đá ở ô đầu tiên sẽ được hiển thị.
                        </p>
                    </div>

                    {supportsPetFlag3 ? (
                        <>
                            <div className="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
                                {normalizedPetStoneSlots.map(
                                    (enabled, index) => (
                                        <div
                                            key={index}
                                            className="flex items-center justify-between rounded-lg border px-3 py-2"
                                        >
                                            <div className="flex flex-col gap-1">
                                                <span className="text-sm font-medium">{`O da ${index + 1}`}</span>
                                                <span className="text-xs text-muted-foreground">
                                                    Bật/tắt
                                                </span>
                                            </div>
                                            <Switch
                                                checked={enabled}
                                                onCheckedChange={value =>
                                                    onPetStoneSlotChange(
                                                        index,
                                                        value
                                                    )
                                                }
                                                disabled={sending}
                                            />
                                        </div>
                                    )
                                )}
                            </div>

                            <div className="grid gap-3 md:grid-cols-[minmax(0,0.9fr)_minmax(0,1.3fr)]">
                                <div className="flex flex-col gap-2">
                                    <Label htmlFor="send-pet-stone-skill">
                                        ID kỹ năng đá ở ô 1
                                    </Label>
                                    <Input
                                        id="send-pet-stone-skill"
                                        type="number"
                                        min={0}
                                        value={petStoneSkillId}
                                        onChange={event =>
                                            onPetStoneSkillIdChange(
                                                event.target.value
                                            )
                                        }
                                        disabled={sending}
                                    />
                                </div>
                                <div className="rounded-lg bg-muted/40 px-3 py-3 text-sm text-muted-foreground">
                                    Nếu có ID kỹ năng, biểu mẫu tự bật ô đá 1 và lưu vào `flag3[1][2]`
                                    để máy khách hiển thị dòng `KN Bảo Thạch`.
                                </div>
                            </div>
                        </>
                    ) : (
                        <div className="rounded-lg border border-dashed px-3 py-4 text-sm text-muted-foreground">
                            flag3 chủ yếu dùng cho trang bị thú cưng nhóm 9. Vật phẩm hiện tại không hỗ trợ
                            biểu mẫu này nhưng vẫn có thể ghi đè dữ liệu thô để thử nghiệm.
                        </div>
                    )}

                    <div className="flex flex-col gap-2">
                        <Label htmlFor="generated-flag-3">
                            Chuỗi dữ liệu được tạo
                        </Label>
                        <Textarea
                            id="generated-flag-3"
                            value={renderGeneratedValue(generatedFlag3)}
                            readOnly
                            className="min-h-[88px] font-mono text-xs"
                        />
                    </div>
                </div>

                <div className="flex flex-col gap-4 rounded-xl border border-dashed p-4">
                    <div className="flex flex-col gap-1">
                        <p className="font-medium">Ghi đè dữ liệu thô</p>
                        <p className="text-sm text-muted-foreground">
                            Nếu ô bên dưới có giá trị, API ưu tiên chuỗi dữ liệu bạn nhập
                            thay vì chuỗi biểu mẫu tự tạo.
                        </p>
                    </div>

                    <div className="grid gap-4 xl:grid-cols-2">
                        <div className="flex flex-col gap-2">
                            <Label htmlFor="send-flag-override">
                                Ghi đè dữ liệu flag
                            </Label>
                            <Textarea
                                id="send-flag-override"
                                value={flagOverride}
                                onChange={event =>
                                    onFlagOverrideChange(event.target.value)
                                }
                                placeholder="{sublimeElement:1,sublimeAdd:0,sublimeId:50}"
                                disabled={sending}
                            />
                        </div>

                        <div className="flex flex-col gap-2">
                            <Label htmlFor="send-flag-2-override">
                                Ghi đè dữ liệu flag2
                            </Label>
                            <Textarea
                                id="send-flag-2-override"
                                value={flag2Override}
                                onChange={event =>
                                    onFlag2OverrideChange(event.target.value)
                                }
                                placeholder="{0:{t:60,v:0.025,max:0.025}}"
                                disabled={sending}
                            />
                        </div>

                        <div className="flex flex-col gap-2">
                            <Label htmlFor="send-flag-3-override">
                                Ghi đè dữ liệu flag3
                            </Label>
                            <Textarea
                                id="send-flag-3-override"
                                value={flag3Override}
                                onChange={event =>
                                    onFlag3OverrideChange(event.target.value)
                                }
                                placeholder="{1:[1,1,12345],2:[1]}"
                                disabled={sending}
                            />
                        </div>

                        <div className="flex flex-col gap-2">
                            <Label htmlFor="send-raw-properties">
                                Thuộc tính thô (JSON)
                            </Label>
                            <Textarea
                                id="send-raw-properties"
                                value={rawProperties}
                                onChange={event =>
                                    onRawPropertiesChange(event.target.value)
                                }
                                placeholder={'{"timeStamp":null}'}
                                className="min-h-[140px] font-mono text-xs"
                                disabled={sending}
                            />
                        </div>
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}
