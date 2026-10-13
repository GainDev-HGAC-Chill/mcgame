"use client"

import Link from "next/link"
import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
    gatewayConfigService,
    type GatewayLine,
} from "@/services/gateway-config.service"
import { gameConfigService } from "@/services/game-config.service"

const STATUS_COLORS: Record<GatewayLine["status"], "default" | "secondary" | "destructive"> = {
    online: "default",
    maintenance: "secondary",
    offline: "destructive",
}

const MAINTENANCE_KEY = "maintenance_mode"

export default function DashboardPage() {
    const [lines, setLines] = useState<GatewayLine[]>([])
    const [loading, setLoading] = useState(true)
    const [deleteTarget, setDeleteTarget] = useState<GatewayLine | null>(null)
    const [maintenance, setMaintenance] = useState<boolean | null>(null)
    const [pendingMaintenance, setPendingMaintenance] = useState<boolean | null>(null)
    const [maintenanceBusy, setMaintenanceBusy] = useState(false)

    const refresh = useCallback(async () => {
        setLoading(true)
        try {
            const [linesData, serverSettings] = await Promise.all([
                gatewayConfigService.list(),
                gameConfigService.list("server"),
            ])
            setLines(linesData)
            const entry = serverSettings.find(item => item.key === MAINTENANCE_KEY)
            setMaintenance(Boolean(entry?.value))
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Không tải được bảng quản trị")
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        void refresh()
    }, [refresh])

    async function handleStatusChange(line: GatewayLine, next: GatewayLine["status"]) {
        try {
            const updated = await gatewayConfigService.update(line.id, { status: next })
            setLines(prev => prev.map(l => (l.id === updated.id ? updated : l)))
            toast.success(`Đã đổi trạng thái kênh ${line.id}`)
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Cập nhật trạng thái thất bại")
        }
    }

    async function handleDelete() {
        if (!deleteTarget) return
        try {
            await gatewayConfigService.remove(deleteTarget.id)
            setLines(prev => prev.filter(l => l.id !== deleteTarget.id))
            toast.success(`Đã xóa kênh ${deleteTarget.id}`)
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Xóa thất bại")
        } finally {
            setDeleteTarget(null)
        }
    }

    async function commitMaintenance() {
        if (pendingMaintenance === null) return
        const next = pendingMaintenance
        setMaintenanceBusy(true)
        try {
            const item = await gameConfigService.upsert("server", {
                key: MAINTENANCE_KEY,
                value: next,
            })
            setMaintenance(Boolean(item.value))
            toast.success(
                next
                    ? "Đã bật bảo trì — đang ngắt kết nối tất cả người chơi"
                    : "Đã tắt bảo trì — cho phép đăng nhập lại",
            )
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Không cập nhật được chế độ bảo trì")
        } finally {
            setMaintenanceBusy(false)
            setPendingMaintenance(null)
        }
    }

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold">Kênh kết nối máy chủ</h1>
                    <p className="text-sm text-muted-foreground">
                        Cấu hình danh sách kênh trả về qua getLineInfo. Thay đổi có hiệu lực trong dưới 1 giây.
                        Chuyển kênh sang ngoại tuyến/bảo trì sẽ ngắt kết nối tất cả người chơi trên kênh đó.
                    </p>
                </div>
                <Button asChild>
                    <Link href="/dashboard/lines/new">Thêm kênh</Link>
                </Button>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Bảo trì toàn hệ thống</CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="flex items-center justify-between gap-4">
                        <div className="space-y-1">
                            <p className="text-sm font-medium">
                                Chế độ bảo trì
                                {maintenance === true ? (
                                    <Badge variant="destructive" className="ml-2">Bật</Badge>
                                ) : maintenance === false ? (
                                    <Badge variant="secondary" className="ml-2">Tắt</Badge>
                                ) : null}
                            </p>
                            <p className="text-sm text-muted-foreground">
                                Khi bật, mọi lượt đăng nhập đều bị chặn và tất cả người chơi bị ngắt kết nối.
                            </p>
                        </div>
                        <Switch
                            checked={Boolean(maintenance)}
                            disabled={maintenance === null || maintenanceBusy}
                            onCheckedChange={value => setPendingMaintenance(value)}
                        />
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Kênh đã cấu hình ({lines.length})</CardTitle>
                </CardHeader>
                <CardContent>
                    {loading ? (
                        <p className="text-sm text-muted-foreground">Đang tải…</p>
                    ) : lines.length === 0 ? (
                        <p className="text-sm text-muted-foreground">
                            Chưa có kênh. Tạo kênh để hiển thị danh sách cho máy khách Flash.
                        </p>
                    ) : (
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>ID</TableHead>
                                    <TableHead>Tên</TableHead>
                                    <TableHead>URL</TableHead>
                                    <TableHead className="text-right">Tối đa</TableHead>
                                    <TableHead>Đấu giá</TableHead>
                                    <TableHead>Bang hội</TableHead>
                                    <TableHead>Trạng thái</TableHead>
                                    <TableHead className="text-right">Thao tác</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {lines.map(line => (
                                    <TableRow key={line.id}>
                                        <TableCell>{line.id}</TableCell>
                                        <TableCell className="font-medium">{line.name}</TableCell>
                                        <TableCell className="font-mono text-xs">{line.url}</TableCell>
                                        <TableCell className="text-right">{line.max_clients}</TableCell>
                                        <TableCell>
                                            <Badge variant={line.auction ? "default" : "outline"}>
                                                {line.auction ? "yes" : "no"}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>
                                            <Badge variant={line.guild ? "default" : "outline"}>
                                                {line.guild ? "yes" : "no"}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>
                                            <Select
                                                value={line.status}
                                                onValueChange={value =>
                                                    handleStatusChange(line, value as GatewayLine["status"])
                                                }
                                            >
                                                <SelectTrigger className="w-[140px]">
                                                    <SelectValue>
                                                        <Badge variant={STATUS_COLORS[line.status]}>
                                                            {line.status}
                                                        </Badge>
                                                    </SelectValue>
                                                </SelectTrigger>
                                                <SelectContent>
                                                    <SelectItem value="online">Trực tuyến</SelectItem>
                                                    <SelectItem value="maintenance">Bảo trì</SelectItem>
                                                    <SelectItem value="offline">Ngoại tuyến</SelectItem>
                                                </SelectContent>
                                            </Select>
                                        </TableCell>
                                        <TableCell className="text-right space-x-2">
                                            <Button variant="outline" size="sm" asChild>
                                                <Link href={`/dashboard/lines/${line.id}`}>Chỉnh sửa</Link>
                                            </Button>
                                            <Button
                                                variant="destructive"
                                                size="sm"
                                                onClick={() => setDeleteTarget(line)}
                                            >
                                                Xóa
                                            </Button>
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    )}
                </CardContent>
            </Card>

            <AlertDialog open={deleteTarget !== null} onOpenChange={open => !open && setDeleteTarget(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Xóa kênh {deleteTarget?.id}?</AlertDialogTitle>
                        <AlertDialogDescription>
                            Máy khách Flash sẽ ngừng hiển thị kênh này sau khoảng 1 giây. Người chơi đã kết nối
                            không bị ngắt kết nối. Không thể hoàn tác thao tác này.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Hủy</AlertDialogCancel>
                        <AlertDialogAction onClick={handleDelete}>Xóa</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            <AlertDialog
                open={pendingMaintenance !== null}
                onOpenChange={open => !open && !maintenanceBusy && setPendingMaintenance(null)}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>
                            {pendingMaintenance
                                ? "Bật chế độ bảo trì?"
                                : "Tắt chế độ bảo trì?"}
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                            {pendingMaintenance
                                ? "Tất cả người chơi đang kết nối sẽ bị ngắt trong khoảng 1 giây. Đăng nhập mới bị từ chối với thông báo máy chủ chưa sẵn sàng cho đến khi tắt bảo trì."
                                : "Cho phép đăng nhập lại ngay. Người chơi đang kết nối (nếu có) không bị ảnh hưởng."}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={maintenanceBusy}>Hủy</AlertDialogCancel>
                        <AlertDialogAction onClick={commitMaintenance} disabled={maintenanceBusy}>
                            {maintenanceBusy ? "Đang áp dụng…" : "Xác nhận"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}
