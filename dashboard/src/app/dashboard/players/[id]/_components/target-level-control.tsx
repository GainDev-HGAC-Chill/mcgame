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

export function TargetLevelControl({
    playerId,
    currentLevel,
}: {
    playerId: number
    currentLevel: number
}) {
    const action = usePlayerAction(playerId)
    const [target, setTarget] = useState("")
    const [confirmed, setConfirmed] = useState(false)
    const level = Number(target)
    const valid =
        target.trim() !== "" &&
        Number.isSafeInteger(level) &&
        level >= 1 &&
        level <= 160
    const lowering = valid && level < currentLevel

    function submit() {
        if (!valid || !confirmed) {
            toast.error("Nhập cấp đích hợp lệ và xác nhận thay đổi.")
            return
        }
        action.mutate(
            { action: "set_target_level", payload: { targetLevel: level } },
            {
                onSuccess: () => {
                    setTarget("")
                    setConfirmed(false)
                },
            }
        )
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Đặt cấp nhân vật</CardTitle>
                <CardDescription>
                    Tăng hoặc hạ đến cấp cụ thể. Chỉ thay đổi nhân vật đang được
                    quản lý. Nhân vật cần đang đăng nhập để máy chủ khóa trạng
                    thái hiện tại an toàn; thay đổi khi ngoại tuyến chưa được hỗ
                    trợ.
                </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
                <div className="space-y-2">
                    <Label htmlFor="target-level">
                        Cấp đích (hiện tại: {currentLevel})
                    </Label>
                    <Input
                        id="target-level"
                        type="number"
                        min={1}
                        max={160}
                        step={1}
                        value={target}
                        onChange={event => {
                            setTarget(event.target.value)
                            setConfirmed(false)
                        }}
                    />
                </div>
                {valid && (
                    <div className="rounded-md border bg-muted/30 p-3 text-sm">
                        <p>
                            Cấp {currentLevel} → {level}.
                        </p>
                        {level === currentLevel ? (
                            <p>Giữ nguyên cấp và kinh nghiệm.</p>
                        ) : (
                            <>
                                <p>
                                    Kinh nghiệm hiện tại được đưa về đầu cấp
                                    đích.
                                </p>
                                {lowering && (
                                    <p>
                                        Hoàn lại điểm chỉ số đã phân bổ và tính
                                        lại chỉ số theo cấp mới. Nếu trang bị
                                        đang mặc không còn hợp lệ, máy chủ sẽ từ
                                        chối; hãy tháo trang bị trước.
                                    </p>
                                )}
                                <p>
                                    Giữ tài sản, thú cưng, kỹ năng đã sở hữu và
                                    lịch sử nhiệm vụ; không phát lại thưởng.
                                </p>
                            </>
                        )}
                    </div>
                )}
                <label className="flex items-start gap-2 text-sm">
                    <input
                        type="checkbox"
                        checked={confirmed}
                        disabled={!valid || action.isPending}
                        onChange={event => setConfirmed(event.target.checked)}
                    />
                    Tôi đã kiểm tra cấp đích và đồng ý với các thay đổi trên.
                </label>
                <Button
                    onClick={submit}
                    disabled={!valid || !confirmed || action.isPending}
                >
                    {action.isPending ? "Đang cập nhật…" : "Xác nhận đặt cấp"}
                </Button>
            </CardContent>
        </Card>
    )
}
