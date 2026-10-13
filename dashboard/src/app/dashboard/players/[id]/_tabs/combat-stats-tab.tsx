"use client"

import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { usePlayerAction } from "@/hooks/use-player-detail"

import {
    type CharacterField,
} from "../_components/character-field-form"

const FIELDS: CharacterField[] = [
    { key: "maxHp", label: "HP tối đa", dtoKey: "hpMax" },
    { key: "maxMp", label: "MP tối đa", dtoKey: "mpMax" },
    { key: "maxSp", label: "SP tối đa", dtoKey: "spMax" },
    { key: "attack", label: "Công vật lý", dtoKey: "baseAttack" },
    { key: "defense", label: "Thủ vật lý", dtoKey: "baseDefense" },
    { key: "magicAttack", label: "Công phép", dtoKey: "baseMagicAttack" },
    { key: "magicDefense", label: "Thủ phép", dtoKey: "baseMagicDefense" },
    { key: "hit", label: "Chính xác", dtoKey: "propHit" },
    { key: "dodge", label: "Né tránh", dtoKey: "propDodge" },
    { key: "critical", label: "Bạo kích", dtoKey: "propCritical" },
    { key: "criticalDmg", label: "Sát thương bạo kích", dtoKey: "baseCriticalDmg" },
    { key: "speed", label: "Tốc độ", dtoKey: "propSpeed" },
]

export function CombatStatsTab({
    char,
    playerId,
}: {
    char: Record<string, unknown>
    playerId: number
}) {
    const mutate = usePlayerAction(playerId)

    const handleRefresh = () => {
        mutate.mutate({
            action: "set_combat_stats",
            payload: { refresh: true },
        })
    }

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">Chỉ số chiến đấu hiện tại</CardTitle>
                    <CardDescription>
                        Máy chủ tính từ thuộc tính, tư chất, trang bị, hiệu ứng và các yếu tố khác đang áp dụng.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
                        {FIELDS.map(field => (
                            <div
                                key={field.key}
                                className="space-y-1 rounded-lg border border-border/60 bg-muted/20 p-3"
                            >
                                <p className="text-xs uppercase tracking-wide text-muted-foreground">
                                    {field.label}
                                </p>
                                <p className="font-medium">
                                    {String(char[field.dtoKey] ?? "-")}
                                </p>
                            </div>
                        ))}
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">Làm mới chỉ số tổng hợp</CardTitle>
                    <CardDescription>
                        Không chỉnh sửa trực tiếp chỉ số chiến đấu tổng hợp. Làm mới người chơi
                        để máy chủ tính lại chỉ số trong bộ nhớ đệm.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <p className="rounded-lg border border-amber-200/60 bg-amber-50/50 px-3 py-2 text-xs text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
                        Dùng thao tác thuộc tính, tiến trình, trang bị, hiệu ứng hoặc vật phẩm để thay đổi
                        dữ liệu nguồn. Thao tác này chỉ yêu cầu tính lại chỉ số.
                    </p>
                    <Button
                        size="sm"
                        onClick={handleRefresh}
                        disabled={mutate.isPending}
                    >
                        Làm mới chỉ số
                    </Button>
                </CardContent>
            </Card>
        </div>
    )
}
