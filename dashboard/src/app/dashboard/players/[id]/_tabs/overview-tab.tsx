"use client"

import { useMemo, useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { useMapList } from "@/hooks/use-maps"
import { usePlayerAction } from "@/hooks/use-player-detail"
import { MapPicker } from "../_components/map-picker"

type OverviewTabProps = {
    char: Record<string, unknown>
    online: boolean
    session?: {
        mapId: number
        channelId: number
        positionX: number
        positionY: number
        connectedAt?: string
    }
    playerId: number
}

function FieldGroup({
    label,
    value,
}: {
    label: string
    value: string | number | boolean | null | undefined
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

export function OverviewTab({
    char,
    online,
    session,
    playerId,
}: OverviewTabProps) {
    const action = usePlayerAction(playerId)
    const { data: maps } = useMapList()
    const liveMapId = Number(session?.mapId ?? char.posMapId ?? 0)
    const [transportMapId, setTransportMapId] = useState<number>(liveMapId)

    const selectedMap = useMemo(
        () => maps?.find(m => m.id === transportMapId),
        [maps, transportMapId]
    )

    const handleKick = () => {
        action.mutate({ action: "kick" })
    }

    const handleTransport = () => {
        if (transportMapId <= 0) {
            toast.error("Vui lòng chọn bản đồ trước")
            return
        }
        if (!selectedMap) {
            toast.error("Dữ liệu bản đồ chưa tải xong")
            return
        }
        action.mutate({
            action: "transport",
            payload: {
                mapId: transportMapId,
                x: selectedMap.safeX,
                y: selectedMap.safeY,
            },
        })
    }

    const safePosLabel = selectedMap
        ? `(${selectedMap.safeX}, ${selectedMap.safeY})`
        : "—"

    return (
        <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
                <FieldGroup label="ID" value={char.id as number} />
                <FieldGroup label="Tên" value={char.name as string} />
                <FieldGroup label="Cấp độ" value={char.level as number} />
                <FieldGroup label="ID nghề" value={char.classId as number} />
                <FieldGroup
                    label="Giới tính"
                    value={char.gender === 1 ? "Nữ" : "Nam"}
                />
                <FieldGroup
                    label="Bản đồ / Vị trí"
                    value={`${char.posMapId} (${char.posX}, ${char.posY})`}
                />
                <FieldGroup
                    label="Trực tuyến"
                    value={online ? "Trực tuyến" : "Ngoại tuyến"}
                />
                <FieldGroup label="Cấp GM" value={char.gmLevel as number} />
            </div>

            {online && session && (
                <Card>
                    <CardHeader className="pb-3">
                        <CardTitle className="text-base">
                            Phiên trực tuyến
                        </CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="grid gap-4 md:grid-cols-4">
                            <FieldGroup
                                label="Bản đồ trực tuyến"
                                value={session.mapId}
                            />
                            <FieldGroup
                                label="Kênh"
                                value={session.channelId}
                            />
                            <FieldGroup
                                label="Vị trí"
                                value={`${session.positionX}, ${session.positionY}`}
                            />
                            <FieldGroup
                                label="Thời điểm kết nối"
                                value={session.connectedAt}
                            />
                        </div>
                    </CardContent>
                </Card>
            )}

            <Card>
                <CardHeader className="pb-3">
                    <CardTitle className="text-base">Thao tác nhanh</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="flex flex-wrap gap-3">
                        <Button
                            variant="destructive"
                            size="sm"
                            onClick={handleKick}
                            disabled={!online || action.isPending}
                        >
                            Ngắt kết nối người chơi
                        </Button>
                    </div>

                    <div className="space-y-3">
                        <p className="text-sm font-medium">Di chuyển người chơi</p>
                        <p className="text-xs text-muted-foreground">
                            Di chuyển người chơi đến điểm xuất hiện an toàn của bản đồ đã chọn.
                        </p>
                        <div className="flex flex-wrap items-end gap-3">
                            <div className="space-y-1">
                                <Label className="text-xs">Bản đồ</Label>
                                <MapPicker
                                    value={transportMapId}
                                    onChange={setTransportMapId}
                                    disabled={action.isPending}
                                />
                            </div>
                            <div className="space-y-1">
                                <Label className="text-xs">Vị trí an toàn</Label>
                                <p className="h-9 rounded-md border border-input bg-muted/40 px-3 py-2 text-sm font-mono">
                                    {safePosLabel}
                                </p>
                            </div>
                            <Button
                                size="sm"
                                onClick={handleTransport}
                                disabled={
                                    action.isPending ||
                                    transportMapId <= 0 ||
                                    !selectedMap
                                }
                            >
                                Di chuyển
                            </Button>
                        </div>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
