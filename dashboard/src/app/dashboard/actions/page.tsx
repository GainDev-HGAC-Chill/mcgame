"use client"

import { useState } from "react"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Shield, Ban, Unlock, VolumeX, UserX, Gift, Search } from "lucide-react"

// Constants
import {
    ACTION_TYPES,
    ACTION_STATUS,
    BAN_DURATIONS,
    COMPENSATION_TYPES,
} from "@/constants/actions"

// Types
import type { ActionHistory, RecentGMAction } from "@/types"

// Icons map
const ACTION_ICONS: Record<
    string,
    React.ComponentType<{ className?: string }>
> = {
    ban: Ban,
    unban: Unlock,
    mute: VolumeX,
    kick: UserX,
}

// Hooks
import { usePlayerActions } from "@/hooks/use-actions"

export default function ActionsPage() {
    const [searchPlayer, setSearchPlayer] = useState("")
    const [selectedAction, setSelectedAction] = useState("")
    const [reason, setReason] = useState("")
    const [duration, setDuration] = useState("60")
    const [compensationType, setCompensationType] = useState("currency")

    // Actions
    const { banPlayer, unbanPlayer, mutePlayer, kickPlayer, compensatePlayer } =
        usePlayerActions()

    const recentActions: RecentGMAction[] = []
    const actionHistory: ActionHistory[] = []

    const handleAction = (actionType: string) => {
        const durationText =
            duration === "-1" ? "Vĩnh viễn" : `${duration} phút`

        switch (actionType) {
            case "ban":
                banPlayer(searchPlayer, reason)
                break
            case "unban":
                unbanPlayer(searchPlayer)
                break
            case "mute":
                mutePlayer(searchPlayer, parseInt(duration))
                break
            case "kick":
                kickPlayer(searchPlayer)
                break
        }

        setSearchPlayer("")
        setReason("")
        setSelectedAction("")
    }

    const getActionConfig = (type: string) =>
        ACTION_TYPES[type as keyof typeof ACTION_TYPES]
    const getStatusConfig = (status: string) =>
        ACTION_STATUS[status as keyof typeof ACTION_STATUS]

    return (
        <div className="space-y-6">
            <Tabs defaultValue="actions" className="space-y-4">
                <TabsList>
                    <TabsTrigger value="actions">Thực hiện thao tác</TabsTrigger>
                    <TabsTrigger value="compensate">Đền bù</TabsTrigger>
                    <TabsTrigger value="history">Lịch sử thao tác</TabsTrigger>
                </TabsList>

                <TabsContent value="actions" className="space-y-4">
                    <Card>
                        <CardHeader>
                            <CardTitle>Chọn thao tác</CardTitle>
                            <CardDescription>
                                Chọn thao tác GM cần thực hiện
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            {/* Player Search */}
                            <div>
                                <Label>Tìm người chơi</Label>
                                <div className="flex gap-2 mt-2">
                                    <div className="relative flex-1">
                                        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                                        <Input
                                            placeholder="Nhập tên hoặc ID người chơi..."
                                            value={searchPlayer}
                                            onChange={e =>
                                                setSearchPlayer(e.target.value)
                                            }
                                            className="pl-10"
                                        />
                                    </div>
                                    <Button variant="outline">Tìm kiếm</Button>
                                </div>
                            </div>

                            {/* Action Type Selection */}
                            <div>
                                <Label>Loại thao tác</Label>
                                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-2">
                                    {Object.entries(ACTION_TYPES).map(
                                        ([id, config]) => {
                                            const Icon =
                                                ACTION_ICONS[id] || Shield
                                            return (
                                                <Card
                                                    key={id}
                                                    className={`cursor-pointer transition-all ${
                                                        selectedAction === id
                                                            ? "ring-2 ring-primary bg-primary/5"
                                                            : "hover:bg-accent"
                                                    }`}
                                                    onClick={() =>
                                                        setSelectedAction(id)
                                                    }
                                                >
                                                    <CardContent className="p-4">
                                                        <div className="flex flex-col items-center text-center gap-2">
                                                            <div
                                                                className={`p-2 rounded-full ${
                                                                    config.color ===
                                                                    "destructive"
                                                                        ? "bg-red-500/20 text-red-500"
                                                                        : config.color ===
                                                                            "secondary"
                                                                          ? "bg-gray-500/20 text-gray-500"
                                                                          : config.color ===
                                                                              "warning"
                                                                            ? "bg-yellow-500/20 text-yellow-500"
                                                                            : "bg-primary/20 text-primary"
                                                                }`}
                                                            >
                                                                <Icon className="h-5 w-5" />
                                                            </div>
                                                            <div>
                                                                <p className="font-medium text-sm">
                                                                    {
                                                                        config.label
                                                                    }
                                                                </p>
                                                                <p className="text-xs text-muted-foreground">
                                                                    {
                                                                        config.description
                                                                    }
                                                                </p>
                                                            </div>
                                                        </div>
                                                    </CardContent>
                                                </Card>
                                            )
                                        }
                                    )}
                                </div>
                            </div>

                            {/* Action Details */}
                            {selectedAction && (
                                <Card className="border-primary/50">
                                    <CardHeader>
                                        <CardTitle className="text-lg">
                                            Chi tiết thao tác
                                        </CardTitle>
                                    </CardHeader>
                                    <CardContent className="space-y-4">
                                        <div>
                                            <Label>Lý do</Label>
                                            <Textarea
                                                placeholder="Nhập lý do thực hiện thao tác..."
                                                value={reason}
                                                onChange={e =>
                                                    setReason(e.target.value)
                                                }
                                                className="min-h-[80px]"
                                            />
                                        </div>

                                        {(selectedAction === "ban" ||
                                            selectedAction === "mute") && (
                                            <div>
                                                <Label>
                                                    Thời hạn (phút)
                                                </Label>
                                                <Select
                                                    value={duration}
                                                    onValueChange={setDuration}
                                                >
                                                    <SelectTrigger>
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        {BAN_DURATIONS.map(
                                                            dur => (
                                                                <SelectItem
                                                                    key={
                                                                        dur.value
                                                                    }
                                                                    value={
                                                                        dur.value
                                                                    }
                                                                >
                                                                    {dur.label}
                                                                </SelectItem>
                                                            )
                                                        )}
                                                    </SelectContent>
                                                </Select>
                                            </div>
                                        )}

                                        <div className="flex gap-2 pt-4">
                                            <Button
                                                variant="outline"
                                                className="flex-1"
                                                onClick={() => {
                                                    setSelectedAction("")
                                                    setReason("")
                                                }}
                                            >
                                                Hủy
                                            </Button>
                                            <AlertDialog>
                                                <AlertDialogTrigger asChild>
                                                    <Button
                                                        variant={
                                                            selectedAction ===
                                                            "ban"
                                                                ? "destructive"
                                                                : "default"
                                                        }
                                                        className="flex-1"
                                                        disabled={
                                                            !searchPlayer ||
                                                            !reason
                                                        }
                                                    >
                                                        Xác nhận
                                                    </Button>
                                                </AlertDialogTrigger>
                                                <AlertDialogContent>
                                                    <AlertDialogHeader>
                                                        <AlertDialogTitle>
                                                            Xác nhận thao tác?
                                                        </AlertDialogTitle>
                                                        <AlertDialogDescription>
                                                            Bạn đang thực hiện{" "}
                                                            <strong>
                                                                {
                                                                    getActionConfig(
                                                                        selectedAction
                                                                    )?.label
                                                                }
                                                            </strong>{" "}
                                                            với người chơi{" "}
                                                            <strong>
                                                                {searchPlayer}
                                                            </strong>
                                                            .
                                                            <br />
                                                            <br />
                                                            Lý do: {reason}
                                                            {(selectedAction ===
                                                                "ban" ||
                                                                selectedAction ===
                                                                    "mute") && (
                                                                <>
                                                                    <br />
                                                                    Thời hạn:{" "}
                                                                    {duration ===
                                                                    "-1"
                                                                        ? "Vĩnh viễn"
                                                                        : `${duration} minutes`}
                                                                </>
                                                            )}
                                                            <br />
                                                            <br />
                                                            Thao tác này sẽ được ghi vào lịch sử.
                                                        </AlertDialogDescription>
                                                    </AlertDialogHeader>
                                                    <AlertDialogFooter>
                                                        <AlertDialogCancel>
                                                            Hủy
                                                        </AlertDialogCancel>
                                                        <AlertDialogAction
                                                            onClick={() =>
                                                                handleAction(
                                                                    selectedAction
                                                                )
                                                            }
                                                        >
                                                            Xác nhận thao tác
                                                        </AlertDialogAction>
                                                    </AlertDialogFooter>
                                                </AlertDialogContent>
                                            </AlertDialog>
                                        </div>
                                    </CardContent>
                                </Card>
                            )}
                        </CardContent>
                    </Card>

                    {/* Recent Actions */}
                    <Card>
                        <CardHeader>
                            <CardTitle>Thao tác gần đây</CardTitle>
                        </CardHeader>
                        <CardContent>
                            {recentActions.length === 0 ? (
                                <p className="text-sm text-muted-foreground">
                                    Chưa có thao tác gần đây.
                                </p>
                            ) : (
                                <div className="space-y-3">
                                    {recentActions.map(action => {
                                        const config = getActionConfig(
                                            action.type
                                        )
                                        const Icon =
                                            ACTION_ICONS[action.type] || Shield
                                        return (
                                            <div
                                                key={action.id}
                                                className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg"
                                            >
                                                <Icon className="h-5 w-5 text-muted-foreground" />
                                                <div className="flex-1">
                                                    <p className="text-sm">
                                                        <span className="font-medium">
                                                            {action.admin}
                                                        </span>{" "}
                                                        đã thực hiện{" "}
                                                        <span className="font-medium">
                                                            {config?.label}
                                                        </span>{" "}
                                                        với{" "}
                                                        <span className="font-medium">
                                                            {action.target}
                                                        </span>
                                                    </p>
                                                    <p className="text-xs text-muted-foreground">
                                                        {action.reason}
                                                    </p>
                                                </div>
                                                <span className="text-xs text-muted-foreground">
                                                    {action.time}
                                                </span>
                                            </div>
                                        )
                                    })}
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </TabsContent>

                <TabsContent value="compensate" className="space-y-4">
                    <Card>
                        <CardHeader>
                            <CardTitle>Đền bù người chơi</CardTitle>
                            <CardDescription>
                                Gửi vật phẩm hoặc tiền tệ đền bù cho người chơi
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div>
                                <Label>
                                    Người nhận (để trống để gửi cho tất cả)
                                </Label>
                                <Input placeholder="Nhập tên hoặc ID người chơi..." />
                            </div>

                            <div>
                                <Label>Loại đền bù</Label>
                                <Select
                                    value={compensationType}
                                    onValueChange={setCompensationType}
                                >
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {Object.entries(COMPENSATION_TYPES).map(
                                            ([key, config]) => (
                                                <SelectItem
                                                    key={key}
                                                    value={key}
                                                >
                                                    {config.label}
                                                </SelectItem>
                                            )
                                        )}
                                    </SelectContent>
                                </Select>
                            </div>

                            {compensationType === "currency" ||
                            compensationType === "both" ? (
                                <div className="grid grid-cols-3 gap-4">
                                    <div>
                                        <Label>Vàng</Label>
                                        <Input type="number" placeholder="0" />
                                    </div>
                                    <div>
                                        <Label>Đá quý</Label>
                                        <Input type="number" placeholder="0" />
                                    </div>
                                    <div>
                                        <Label>Xu</Label>
                                        <Input type="number" placeholder="0" />
                                    </div>
                                </div>
                            ) : null}

                            {compensationType === "item" ||
                            compensationType === "both" ? (
                                <div>
                                    <Label>ID vật phẩm (phân cách bằng dấu phẩy)</Label>
                                    <Input placeholder="item001, item002, item003" />
                                    <p className="text-xs text-muted-foreground mt-1">
                                        Ví dụ: potion_hp, sword_epic
                                    </p>
                                </div>
                            ) : null}

                            <div>
                                <Label>Số lượng mỗi vật phẩm</Label>
                                <Input
                                    type="number"
                                    placeholder="1"
                                    defaultValue="1"
                                />
                            </div>

                            <div>
                                <Label>Lý do đền bù</Label>
                                <Textarea
                                    placeholder="Nhập lý do..."
                                    className="min-h-[80px]"
                                />
                            </div>

                            <AlertDialog>
                                <AlertDialogTrigger asChild>
                                    <Button className="w-full">
                                        <Gift className="h-4 w-4 mr-2" />
                                        Gửi đền bù
                                    </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                    <AlertDialogHeader>
                                        <AlertDialogTitle>
                                            Xác nhận gửi đền bù?
                                        </AlertDialogTitle>
                                        <AlertDialogDescription>
                                            Vui lòng kiểm tra kỹ thông tin trước khi gửi. Không thể hoàn tác thao tác này.
                                        </AlertDialogDescription>
                                    </AlertDialogHeader>
                                    <AlertDialogFooter>
                                        <AlertDialogCancel>
                                            Hủy
                                        </AlertDialogCancel>
                                        <AlertDialogAction>
                                            Xác nhận gửi
                                        </AlertDialogAction>
                                    </AlertDialogFooter>
                                </AlertDialogContent>
                            </AlertDialog>
                        </CardContent>
                    </Card>
                </TabsContent>

                <TabsContent value="history" className="space-y-4">
                    <Card>
                        <CardHeader>
                            <CardTitle>Lịch sử thao tác GM</CardTitle>
                            <CardDescription>
                                Tất cả thao tác đã thực hiện
                            </CardDescription>
                        </CardHeader>
                        <CardContent>
                            {actionHistory.length === 0 ? (
                                <p className="text-sm text-muted-foreground">
                                    Chưa có lịch sử thao tác.
                                </p>
                            ) : (
                                <div className="space-y-3">
                                    {actionHistory.map(action => {
                                        const Icon =
                                            ACTION_ICONS[action.type] || Shield
                                        const statusConfig = getStatusConfig(
                                            action.status
                                        )
                                        return (
                                            <div
                                                key={action.id}
                                                className="flex items-center gap-3 p-4 border rounded-lg"
                                            >
                                                <div
                                                    className={`p-2 rounded-full ${
                                                        action.type === "ban"
                                                            ? "bg-red-500/20 text-red-500"
                                                            : action.type ===
                                                                "mute"
                                                              ? "bg-gray-500/20 text-gray-500"
                                                              : action.type ===
                                                                  "unban"
                                                                ? "bg-green-500/20 text-green-500"
                                                                : "bg-blue-500/20 text-blue-500"
                                                    }`}
                                                >
                                                    <Icon className="h-4 w-4" />
                                                </div>
                                                <div className="flex-1">
                                                    <div className="flex items-center gap-2">
                                                        <span className="font-medium">
                                                            {action.admin}
                                                        </span>
                                                        <span className="text-sm text-muted-foreground">
                                                            đã thực hiện
                                                        </span>
                                                        <span className="font-medium capitalize">
                                                            {action.type}
                                                        </span>
                                                        <span className="font-medium">
                                                            {action.target}
                                                        </span>
                                                        <Badge
                                                            variant={
                                                                statusConfig.variant
                                                            }
                                                        >
                                                            {statusConfig.label}
                                                        </Badge>
                                                    </div>
                                                    <p className="text-sm text-muted-foreground">
                                                        {action.reason}
                                                    </p>
                                                </div>
                                                <div className="text-right text-sm">
                                                    <p>{action.duration}</p>
                                                    <p className="text-muted-foreground">
                                                        {action.time}
                                                    </p>
                                                </div>
                                            </div>
                                        )
                                    })}
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </TabsContent>
            </Tabs>
        </div>
    )
}
