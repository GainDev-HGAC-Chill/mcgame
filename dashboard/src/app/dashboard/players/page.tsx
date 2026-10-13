"use client"

import { useDeferredValue, useState } from "react"
import { useRouter } from "next/navigation"
import { Eye, RefreshCw, Search, Settings } from "lucide-react"

import { usePlayers, type PlayerListItem } from "@/hooks/use-players"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"

const defaultPlayerLimit = 500

function formatDateTime(value: string | null): string {
    if (!value) return "-"
    const parsed = new Date(value)
    if (Number.isNaN(parsed.valueOf())) return value
    return new Intl.DateTimeFormat("vi-VN", {
        dateStyle: "short",
        timeStyle: "medium",
    }).format(parsed)
}

function formatSexLabel(sex: number): string {
    return sex === 1 ? "Female" : "Male"
}

function getPlayerInitials(player: PlayerListItem): string {
    const trimmedName = player.playerName.trim()
    if (!trimmedName) return String(player.id)
    return trimmedName.slice(0, 1).toUpperCase()
}

function DetailField(props: { label: string; value: string }) {
    return (
        <div className="space-y-1 rounded-lg border border-border/60 bg-muted/20 p-3">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
                {props.label}
            </p>
            <p className="font-medium">{props.value}</p>
        </div>
    )
}

export default function PlayersPage() {
    const router = useRouter()
    const [searchTerm, setSearchTerm] = useState("")
    const deferredSearch = useDeferredValue(searchTerm.trim())
    const [selectedPlayer, setSelectedPlayer] = useState<PlayerListItem | null>(
        null
    )

    const { data, isLoading, isFetching, error, refetch } = usePlayers(
        deferredSearch,
        defaultPlayerLimit
    )

    const players = data?.items ?? []
    const totalPlayers = data?.total ?? 0
    const onlinePlayers = data?.onlineCount ?? 0
    const lastRefreshedAt = data?.refreshedAt ?? null

    return (
        <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="pb-3">
                        <CardDescription>Trực tuyến theo thời gian thực</CardDescription>
                        <CardTitle>{onlinePlayers}</CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader className="pb-3">
                        <CardDescription>Đang hiển thị</CardDescription>
                        <CardTitle>
                            {players.length} / {totalPlayers}
                        </CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader className="pb-3">
                        <CardDescription>Lần làm mới gần nhất</CardDescription>
                        <CardTitle className="text-base">
                            {formatDateTime(lastRefreshedAt)}
                        </CardTitle>
                    </CardHeader>
                </Card>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Tìm người chơi</CardTitle>
                    <CardDescription>
                        Tìm theo tên người chơi, tài khoản, ID hoặc UID. Danh sách tự làm mới mỗi 5 giây.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="flex flex-col gap-3 md:flex-row">
                        <div className="relative flex-1">
                            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                            <Input
                                placeholder="Nhập tên, tài khoản, ID hoặc UID..."
                                value={searchTerm}
                                onChange={event =>
                                    setSearchTerm(event.target.value)
                                }
                                className="pl-10"
                            />
                        </div>
                        <Button
                            variant="outline"
                            onClick={() => void refetch()}
                            disabled={isLoading || isFetching}
                        >
                            <RefreshCw
                                className={`mr-2 h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
                            />
                            Làm mới
                        </Button>
                    </div>

                    {totalPlayers > players.length ? (
                        <p className="text-sm text-muted-foreground">
                            Hiển thị tối đa {players.length} trên {totalPlayers}{" "}
                            người chơi gần nhất. Dùng tìm kiếm để lọc trên máy chủ.
                        </p>
                    ) : null}

                    {error ? (
                        <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                            {error instanceof Error
                                ? error.message
                                : "Không tải được người chơi"}
                        </div>
                    ) : null}
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Danh sách người chơi</CardTitle>
                    <CardDescription>
                        Trạng thái trực tuyến lấy từ Redis đang chạy, không phải cột online trong Postgres.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>ID</TableHead>
                                <TableHead>Người chơi</TableHead>
                                <TableHead>Tài khoản</TableHead>
                                <TableHead>Cấp độ</TableHead>
                                <TableHead>Máy chủ</TableHead>
                                <TableHead>Bản đồ</TableHead>
                                <TableHead>Trạng thái</TableHead>
                                <TableHead>Hoạt động gần nhất</TableHead>
                                <TableHead>Thao tác</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {isLoading ? (
                                <TableRow>
                                    <TableCell
                                        colSpan={9}
                                        className="h-24 text-center text-muted-foreground"
                                    >
                                        Đang tải danh sách người chơi...
                                    </TableCell>
                                </TableRow>
                            ) : players.length === 0 ? (
                                <TableRow>
                                    <TableCell
                                        colSpan={9}
                                        className="h-24 text-center text-muted-foreground"
                                    >
                                        Không tìm thấy người chơi.
                                    </TableCell>
                                </TableRow>
                            ) : (
                                players.map(player => (
                                    <TableRow key={player.id}>
                                        <TableCell className="font-mono">
                                            {player.id}
                                        </TableCell>
                                        <TableCell>
                                            <div className="flex items-center gap-3">
                                                <Avatar className="h-8 w-8">
                                                    <AvatarFallback>
                                                        {getPlayerInitials(
                                                            player
                                                        )}
                                                    </AvatarFallback>
                                                </Avatar>
                                                <div className="space-y-0.5">
                                                    <p className="font-medium">
                                                        {player.playerName}
                                                    </p>
                                                    <p className="text-xs text-muted-foreground">
                                                        UID {player.playerUid}
                                                    </p>
                                                </div>
                                            </div>
                                        </TableCell>
                                        <TableCell>
                                            {player.accountName}
                                        </TableCell>
                                        <TableCell>
                                            <Badge variant="secondary">
                                                {player.level}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>
                                            {player.serverKey}
                                        </TableCell>
                                        <TableCell>
                                            {player.mapId}
                                            {player.instanceId > 0
                                                ? ` / ${player.instanceId}`
                                                : ""}
                                        </TableCell>
                                        <TableCell>
                                            <Badge
                                                variant={
                                                    player.online
                                                        ? "success"
                                                        : "secondary"
                                                }
                                            >
                                                {player.online
                                                    ? "Trực tuyến"
                                                    : "Ngoại tuyến"}
                                            </Badge>
                                        </TableCell>
                                        <TableCell className="text-sm">
                                            {formatDateTime(player.lastSeenAt)}
                                        </TableCell>
                                        <TableCell>
                                            <div className="flex items-center gap-1">
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() =>
                                                        setSelectedPlayer(
                                                            player
                                                        )
                                                    }
                                                >
                                                    <Eye className="mr-1 h-4 w-4" />
                                                    Xem
                                                </Button>
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() =>
                                                        router.push(
                                                            `/dashboard/players/${player.id}`
                                                        )
                                                    }
                                                >
                                                    <Settings className="mr-1 h-4 w-4" />
                                                    Quản lý
                                                </Button>
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                ))
                            )}
                        </TableBody>
                    </Table>
                </CardContent>
            </Card>

            <Dialog
                open={selectedPlayer !== null}
                onOpenChange={open => !open && setSelectedPlayer(null)}
            >
                <DialogContent className="max-w-3xl">
                    <DialogHeader>
                        <DialogTitle>
                            {selectedPlayer?.playerName ?? "Chi tiết người chơi"}
                        </DialogTitle>
                        <DialogDescription>
                            ID người chơi {selectedPlayer?.id ?? "-"} | UID{" "}
                            {selectedPlayer?.playerUid ?? "-"}
                        </DialogDescription>
                    </DialogHeader>

                    {selectedPlayer ? (
                        <div className="grid gap-4 md:grid-cols-2">
                            <DetailField
                                label="Tài khoản"
                                value={selectedPlayer.accountName}
                            />
                            <DetailField
                                label="Máy chủ"
                                value={`${selectedPlayer.serverKey} (#${selectedPlayer.serverId})`}
                            />
                            <DetailField
                                label="Cấp độ"
                                value={String(selectedPlayer.level)}
                            />
                            <DetailField
                                label="Giới tính"
                                value={formatSexLabel(selectedPlayer.sex)}
                            />
                            <DetailField
                                label="Bản đồ / Phụ bản"
                                value={`${selectedPlayer.mapId} / ${selectedPlayer.instanceId}`}
                            />
                            <DetailField
                                label="Vị trí"
                                value={`${selectedPlayer.position.x}, ${selectedPlayer.position.y}`}
                            />
                            <DetailField
                                label="Trực tuyến"
                                value={
                                    selectedPlayer.online ? "Trực tuyến" : "Ngoại tuyến"
                                }
                            />
                            <DetailField
                                label="Lần xuất hiện gần nhất"
                                value={formatDateTime(
                                    selectedPlayer.lastSeenAt
                                )}
                            />
                            <DetailField
                                label="Đăng nhập gần nhất"
                                value={formatDateTime(
                                    selectedPlayer.lastLoginAt
                                )}
                            />
                            <DetailField
                                label="Di chuyển gần nhất"
                                value={formatDateTime(
                                    selectedPlayer.lastMoveAt
                                )}
                            />
                            <DetailField
                                label="Cập nhật lúc"
                                value={formatDateTime(selectedPlayer.updatedAt)}
                            />
                            <DetailField
                                label="Tạo lúc"
                                value={formatDateTime(selectedPlayer.createdAt)}
                            />
                        </div>
                    ) : null}
                </DialogContent>
            </Dialog>
        </div>
    )
}
