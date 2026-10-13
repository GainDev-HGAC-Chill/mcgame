"use client"

import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { usePlayerAction } from "@/hooks/use-player-detail"

function FieldGroup({
    label,
    value,
}: {
    label: string
    value: string | number | null | undefined
}) {
    const display =
        value === null || value === undefined || value === ""
            ? "-"
            : String(value)
    return (
        <div className="space-y-1 rounded-lg border border-border/60 bg-muted/20 p-3">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
                {label}
            </p>
            <p className="font-medium">{display}</p>
        </div>
    )
}

export function GMTab({
    char,
    playerId,
}: {
    char: Record<string, unknown>
    playerId: number
}) {
    const action = usePlayerAction(playerId)
    const [gmLevel, setGmLevel] = useState(String(char.gmLevel ?? 0))

    const handleSetGMLevel = () => {
        const level = Number(gmLevel)
        if (!Number.isFinite(level) || level < 0 || level > 10) {
            toast.error("Cấp GM phải từ 0 đến 10")
            return
        }
        action.mutate({ action: "set_gm_level", payload: { gmLevel: level } })
    }

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">Cấp GM</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="flex items-end gap-3">
                        <div className="space-y-1">
                            <Label className="text-xs">Cấp GM hiện tại</Label>
                            <p className="font-medium text-lg">
                                {String(char.gmLevel ?? 0)}
                            </p>
                        </div>
                        <div className="space-y-1">
                            <Label className="text-xs">Cấp mới (0–10)</Label>
                            <Input
                                className="w-32"
                                type="number"
                                min={0}
                                max={10}
                                value={gmLevel}
                                onChange={e => setGmLevel(e.target.value)}
                            />
                        </div>
                        <Button
                            size="sm"
                            onClick={handleSetGMLevel}
                            disabled={action.isPending}
                        >
                            Đặt cấp GM
                        </Button>
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">
                        Tóm tắt nhân vật
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="grid gap-3 md:grid-cols-3">
                        <FieldGroup
                        label="Loại VIP"
                            value={char.vipType as number}
                        />
                        <FieldGroup
                        label="Cấp thức tỉnh"
                            value={char.awakenLevel as number}
                        />
                        <FieldGroup
                        label="Cấp chuyển sinh"
                            value={char.rebirthLvl as number}
                        />
                        <FieldGroup
                        label="Tỷ lệ tăng trưởng"
                            value={char.growRate as number}
                        />
                        <FieldGroup
                        label="Loại phẩm chất"
                            value={char.qualityType as number}
                        />
                        <FieldGroup
                        label="Loại sao"
                            value={char.starType as number}
                        />
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
