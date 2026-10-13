"use client"

import { useState } from "react"
import { toast } from "sonner"

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
import { usePlayerAction } from "@/hooks/use-player-detail"
import { TargetLevelControl } from "../_components/target-level-control"

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

function addPositiveInt(
    payload: Record<string, number>,
    key: string,
    raw: string
) {
    if (raw.trim() === "") return
    const parsed = Math.trunc(Number(raw))
    if (Number.isFinite(parsed) && parsed >= 0) {
        payload[key] = parsed
    }
}

export function ProgressionTab({
    char,
    playerId,
}: {
    char: Record<string, unknown>
    playerId: number
}) {
    const action = usePlayerAction(playerId)
    const [levelDelta, setLevelDelta] = useState("")
    const [experienceDelta, setExperienceDelta] = useState("")
    const [soulLevel, setSoulLevel] = useState("")
    const [soulExp, setSoulExp] = useState("")
    const [soulPoints, setSoulPoints] = useState("")

    const handleApply = () => {
        const payload: Record<string, number> = {}
        const parsedLevelDelta = Math.trunc(Number(levelDelta))
        const parsedExperienceDelta = Math.trunc(Number(experienceDelta))

        if (
            levelDelta.trim() !== "" &&
            Number.isFinite(parsedLevelDelta) &&
            parsedLevelDelta > 0
        ) {
            payload.levelDelta = parsedLevelDelta
        }
        if (
            experienceDelta.trim() !== "" &&
            Number.isFinite(parsedExperienceDelta) &&
            parsedExperienceDelta > 0
        ) {
            payload.experienceDelta = parsedExperienceDelta
        }

        if (Object.keys(payload).length === 0) {
            toast.error("Nhập số cấp tăng thêm hoặc kinh nghiệm cộng thêm lớn hơn 0")
            return
        }

        action.mutate(
            { action: "advance_progression", payload },
            {
                onSuccess: () => {
                    setLevelDelta("")
                    setExperienceDelta("")
                },
            }
        )
    }

    const handleSetProgression = () => {
        const payload: Record<string, number> = {}
        addPositiveInt(payload, "soulLevel", soulLevel)
        addPositiveInt(payload, "soulExp", soulExp)
        addPositiveInt(payload, "soulPoints", soulPoints)

        if (Object.keys(payload).length === 0) {
            toast.error("Nhập ít nhất một giá trị tu luyện linh hồn")
            return
        }

        action.mutate(
            { action: "set_progression", payload },
            {
                onSuccess: () => {
                    setSoulLevel("")
                    setSoulExp("")
                    setSoulPoints("")
                },
            }
        )
    }

    return (
        <div className="space-y-6">
            <TargetLevelControl playerId={playerId} currentLevel={Number(char.level)} />
            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">
                        Tiến trình hiện tại
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="grid gap-3 md:grid-cols-2">
                        <FieldGroup
                            label="Cấp độ"
                            value={char.level as number}
                        />
                        <FieldGroup
                            label="Kinh nghiệm"
                            value={char.experience as number}
                        />
                        <FieldGroup
                            label="Cấp linh hồn"
                            value={char.soulLevel as number}
                        />
                        <FieldGroup
                            label="Kinh nghiệm linh hồn"
                            value={char.soulExp as number}
                        />
                        <FieldGroup
                            label="Điểm linh hồn"
                            value={char.soulPoints as number}
                        />
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">
                        Điều chỉnh tiến trình
                    </CardTitle>
                    <CardDescription>
                        Chỉ dùng giá trị dương. Tăng cấp sẽ cộng vào cấp hiện tại.
                        Cộng kinh nghiệm sẽ cộng vào kinh nghiệm hiện tại.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-1">
                            <Label className="text-xs">Tăng cấp</Label>
                            <Input
                                type="number"
                                min={1}
                                placeholder="Số cấp tăng thêm"
                                value={levelDelta}
                                onChange={e => setLevelDelta(e.target.value)}
                            />
                        </div>
                        <div className="space-y-1">
                            <Label className="text-xs">Cộng kinh nghiệm</Label>
                            <Input
                                type="number"
                                min={1}
                                placeholder="Kinh nghiệm cộng thêm"
                                value={experienceDelta}
                                onChange={e =>
                                    setExperienceDelta(e.target.value)
                                }
                            />
                        </div>
                    </div>
                    <p className="rounded-lg border border-amber-200/60 bg-amber-50/50 px-3 py-2 text-xs text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                        Tăng cấp dùng quy tắc tiến trình của máy chủ và đặt kinh nghiệm
                        hiện tại về 0 trước khi cộng thêm. Cộng kinh nghiệm vẫn tuân theo
                        quy tắc lên cấp bình thường của máy chủ.
                    </p>
                    <Button
                        size="sm"
                        onClick={handleApply}
                        disabled={action.isPending}
                    >
                        Áp dụng tiến trình
                    </Button>
                </CardContent>
            </Card>

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">
                        Đặt tiến trình trực tiếp
                    </CardTitle>
                    <CardDescription>
                        Đặt trực tiếp giá trị tu luyện linh hồn.
                        Để trống để giữ nguyên giá trị hiện tại.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-3">
                        <div className="space-y-1">
                            <Label className="text-xs">Cấp linh hồn</Label>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Cấp linh hồn"
                                value={soulLevel}
                                onChange={e => setSoulLevel(e.target.value)}
                            />
                        </div>
                        <div className="space-y-1">
                            <Label className="text-xs">Kinh nghiệm linh hồn</Label>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Kinh nghiệm linh hồn"
                                value={soulExp}
                                onChange={e => setSoulExp(e.target.value)}
                            />
                        </div>
                        <div className="space-y-1">
                            <Label className="text-xs">Điểm linh hồn</Label>
                            <Input
                                type="number"
                                min={0}
                                placeholder="Điểm linh hồn"
                                value={soulPoints}
                                onChange={e => setSoulPoints(e.target.value)}
                            />
                        </div>
                    </div>
                    <p className="rounded-lg border border-amber-200/60 bg-amber-50/50 px-3 py-2 text-xs text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                        Thao tác đặt giá trị trực tiếp, bỏ qua quy trình lên cấp.
                        Điểm linh hồn là cùng số dư Pha Lê (soulPnt)
                        trong thẻ Tiền tệ.
                    </p>
                    <Button
                        size="sm"
                        onClick={handleSetProgression}
                        disabled={action.isPending}
                    >
                        Đặt tiến trình
                    </Button>
                </CardContent>
            </Card>
        </div>
    )
}
