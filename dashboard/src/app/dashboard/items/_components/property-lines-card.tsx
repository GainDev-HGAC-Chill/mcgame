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

import { getPropOptionsForSlot, type EditablePropLine } from "../_lib/shared"

type PropertyLinesCardProps = {
    sending: boolean
    propsLines: EditablePropLine[]
    onUpdatePropLine: (
        index: number,
        field: "type" | "value",
        value: string
    ) => void
}

export function PropertyLinesCard({
    sending,
    propsLines,
    onUpdatePropLine,
}: PropertyLinesCardProps) {
    return (
        <Card className="min-w-0">
            <CardHeader>
                <CardTitle>Dòng thuộc tính vật phẩm</CardTitle>
                <p className="text-sm text-muted-foreground">
                    Máy khách đọc trực tiếp các khóa `mainProp`, `prop` và `activeProp` của vật phẩm.
                    Giá trị tại đây ghi đè mẫu khi hiển thị chú giải.
                </p>
            </CardHeader>
            <CardContent className="flex min-w-0 flex-col gap-3">
                {propsLines.map((line, index) => (
                    <div
                        key={line.slot}
                        className="grid min-w-0 gap-3 rounded-xl border p-3"
                    >
                        <div className="grid gap-3 sm:grid-cols-[minmax(0,0.9fr)_minmax(0,1.3fr)_minmax(0,0.8fr)]">
                            <div className="flex min-w-0 flex-col gap-2">
                                <Label>Ô</Label>
                                <Input
                                    value={line.label}
                                    readOnly
                                    className="bg-muted/40"
                                />
                            </div>

                            <div className="flex min-w-0 flex-col gap-2">
                                <Label>
                                    {line.slot === "active"
                                        ? "Kích hoạt linh hồn"
                                        : "Attribute"}
                                </Label>
                                <Select
                                    value={line.type || "0"}
                                    onValueChange={value =>
                                        onUpdatePropLine(index, "type", value)
                                    }
                                    disabled={sending}
                                >
                                    <SelectTrigger className="w-full min-w-0">
                                        <SelectValue placeholder="Chọn thuộc tính" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectGroup>
                                            <SelectItem value="0">
                                                Trống
                                            </SelectItem>
                                            {getPropOptionsForSlot(
                                                line.slot
                                            ).map(option => (
                                                <SelectItem
                                                    key={option.value}
                                                    value={String(option.value)}
                                                >
                                                    {option.label} (
                                                    {option.value})
                                                </SelectItem>
                                            ))}
                                        </SelectGroup>
                                    </SelectContent>
                                </Select>
                            </div>

                            <div className="flex min-w-0 flex-col gap-2">
                                <Label>Giá trị</Label>
                                <Input
                                    type="number"
                                    min={0}
                                    value={line.value}
                                    onChange={event =>
                                        onUpdatePropLine(
                                            index,
                                            "value",
                                            event.target.value
                                        )
                                    }
                                    disabled={sending}
                                />
                            </div>
                        </div>
                    </div>
                ))}
            </CardContent>
        </Card>
    )
}
