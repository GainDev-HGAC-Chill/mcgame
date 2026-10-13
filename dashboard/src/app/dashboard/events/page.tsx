"use client"

import { useState } from "react"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { Slider } from "@/components/ui/slider"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
    Calendar,
    Clock,
    TrendingUp,
    Package,
    Plus,
    Edit,
    Trash2,
} from "lucide-react"
import { toast } from "sonner"
import { DatePicker } from "@/components/ui/date-picker"

// Constants
import { EVENT_TYPES, DAYS_OF_WEEK } from "@/constants/events"

// Hooks
import { useToggle } from "@/hooks/use-common"
import { useEventActions } from "@/hooks/use-actions"

// Types
import type { GameEvent } from "@/types"

export default function EventsPage() {
    // State
    const [events, setEvents] = useState<GameEvent[]>([])
    const [selectedEvent, setSelectedEvent] = useState<GameEvent | null>(null)
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useToggle(false)
    const [isEditDialogOpen, setIsEditDialogOpen] = useToggle(false)
    const [eventStartDate, setEventStartDate] = useState<Date>()
    const [eventEndDate, setEventEndDate] = useState<Date>()

    // Actions
    const {
        toggleEventStatus: handleToggleEventStatus,
        createEvent,
        saveEvent,
        deleteEvent,
    } = useEventActions()

    // Handlers
    const toggleEventStatus = (eventId: string) => {
        handleToggleEventStatus(eventId, events, setEvents)
    }

    const handleCreateEvent = () => {
        createEvent()
        setIsCreateDialogOpen(false)
    }

    const handleSaveEvent = () => {
        saveEvent(selectedEvent?.name)
        setIsEditDialogOpen(false)
    }

    const handleDeleteEvent = (eventId: string, eventName: string) => {
        deleteEvent(eventName)
        setEvents(events.filter(e => e.id !== eventId))
        setIsEditDialogOpen(false)
    }

    const handleSelectEvent = (event: GameEvent) => {
        setSelectedEvent(event)
        if (event.startTime) setEventStartDate(new Date(event.startTime))
        if (event.endTime) setEventEndDate(new Date(event.endTime))
        setIsEditDialogOpen(true)
    }

    const getEventColor = (type: keyof typeof EVENT_TYPES) =>
        EVENT_TYPES[type].color
    const getEventLabel = (type: keyof typeof EVENT_TYPES) =>
        EVENT_TYPES[type].label

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-end">
                <Dialog
                    open={isCreateDialogOpen}
                    onOpenChange={setIsCreateDialogOpen}
                >
                    <DialogTrigger asChild>
                        <Button>
                            <Plus className="h-4 w-4 mr-2" />
                            Tạo sự kiện mới
                        </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-2xl">
                        <DialogHeader>
                            <DialogTitle>Tạo sự kiện mới</DialogTitle>
                            <DialogDescription>
                                Thiết lập thông tin và cấu hình cho sự kiện mới
                            </DialogDescription>
                        </DialogHeader>
                        <div className="space-y-4 py-4">
                            <div>
                                <Label htmlFor="event-name">Tên sự kiện</Label>
                                <Input
                                    id="event-name"
                                    placeholder="Nhập tên sự kiện..."
                                />
                            </div>
                            <div>
                                <Label htmlFor="event-type">Loại sự kiện</Label>
                                <Select>
                                    <SelectTrigger>
                                        <SelectValue placeholder="Chọn loại sự kiện" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {Object.entries(EVENT_TYPES).map(
                                            ([key, { label }]) => (
                                                <SelectItem
                                                    key={key}
                                                    value={key}
                                                >
                                                    {label}
                                                </SelectItem>
                                            )
                                        )}
                                    </SelectContent>
                                </Select>
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <Label>Thời gian bắt đầu</Label>
                                    <DatePicker
                                        value={eventStartDate}
                                        onChange={setEventStartDate}
                                        placeholder="Chọn ngày bắt đầu"
                                    />
                                </div>
                                <div>
                                    <Label>Thời gian kết thúc</Label>
                                    <DatePicker
                                        value={eventEndDate}
                                        onChange={setEventEndDate}
                                        placeholder="Chọn ngày kết thúc"
                                    />
                                </div>
                            </div>
                            <div>
                                <Label htmlFor="description">Mô tả</Label>
                                <Textarea
                                    id="description"
                                    placeholder="Mô tả sự kiện..."
                                />
                            </div>
                        </div>
                        <div className="flex justify-end gap-2">
                            <Button
                                variant="outline"
                                onClick={() => setIsCreateDialogOpen(false)}
                            >
                                Hủy
                            </Button>
                            <Button onClick={handleCreateEvent}>
                                Tạo sự kiện
                            </Button>
                        </div>
                    </DialogContent>
                </Dialog>
            </div>

            {events.length === 0 ? (
                <Card>
                    <CardContent className="py-8">
                        <p className="text-sm text-muted-foreground text-center">
                            Chưa cấu hình sự kiện.
                        </p>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-6 md:grid-cols-2">
                    {events.map(event => (
                        <Card key={event.id}>
                            <CardHeader>
                                <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-2 mb-1">
                                            <CardTitle className="text-lg">
                                                {event.name}
                                            </CardTitle>
                                            <Badge
                                                className={`${getEventColor(event.type as keyof typeof EVENT_TYPES)} text-white`}
                                            >
                                                {getEventLabel(
                                                    event.type as keyof typeof EVENT_TYPES
                                                )}
                                            </Badge>
                                        </div>
                                        <CardDescription>
                                            {event.description}
                                        </CardDescription>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Switch
                                            checked={event.active}
                                            onCheckedChange={() =>
                                                toggleEventStatus(event.id)
                                            }
                                        />
                                        <Badge
                                            variant={
                                                event.active
                                                    ? "success"
                                                    : "secondary"
                                            }
                                        >
                                            {event.active
                                                ? "Đang hoạt động"
                                                : "Inactive"}
                                        </Badge>
                                    </div>
                                </div>
                            </CardHeader>
                            <CardContent>
                                <div className="space-y-4">
                                    <div className="flex items-center gap-4 text-sm">
                                        <div className="flex items-center gap-1">
                                            <Calendar className="h-4 w-4 text-muted-foreground" />
                                            <span>
                                                Bắt đầu: {event.startTime}
                                            </span>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-4 text-sm">
                                        <div className="flex items-center gap-1">
                                            <Clock className="h-4 w-4 text-muted-foreground" />
                                            <span>Kết thúc: {event.endTime}</span>
                                        </div>
                                    </div>

                                    <div className="pt-4 border-t">
                                        <p className="text-sm font-medium mb-3">
                                            Cấu hình lịch
                                        </p>
                                        <div className="space-y-2">
                                            {event.schedule.map(
                                                (schedule, index) => (
                                                    <div
                                                        key={index}
                                                        className="bg-muted/50 rounded-lg p-3"
                                                    >
                                                        <div className="grid grid-cols-2 gap-2 text-sm">
                                                            <div>
                                                                <span className="text-muted-foreground">
                                                                    Ngày:{" "}
                                                                </span>
                                                                <span className="font-medium">
                                                                    {
                                                                        DAYS_OF_WEEK[
                                                                            schedule
                                                                                .dayOfWeek
                                                                        ]
                                                                    }
                                                                </span>
                                                            </div>
                                                            <div>
                                                                <span className="text-muted-foreground">
                                                                    Thời gian:{" "}
                                                                </span>
                                                                <span className="font-medium">
                                                                    {
                                                                        schedule.startHour
                                                                    }
                                                                    :00 -{" "}
                                                                    {
                                                                        schedule.endHour
                                                                    }
                                                                    :00
                                                                </span>
                                                            </div>
                                                            {schedule.expMultiplier && (
                                                                <div className="flex items-center gap-1">
                                                                    <TrendingUp className="h-3 w-3 text-blue-500" />
                                                                    <span className="text-muted-foreground">
                                                                        EXP:{" "}
                                                                    </span>
                                                                    <span className="font-medium text-blue-500">
                                                                        x
                                                                        {
                                                                            schedule.expMultiplier
                                                                        }
                                                                    </span>
                                                                </div>
                                                            )}
                                                            {schedule.dropMultiplier && (
                                                                <div className="flex items-center gap-1">
                                                                    <Package className="h-3 w-3 text-green-500" />
                                                                    <span className="text-muted-foreground">
                                                                        Vật phẩm rơi:{" "}
                                                                    </span>
                                                                    <span className="font-medium text-green-500">
                                                                        x
                                                                        {
                                                                            schedule.dropMultiplier
                                                                        }
                                                                    </span>
                                                                </div>
                                                            )}
                                                        </div>
                                                    </div>
                                                )
                                            )}
                                        </div>
                                    </div>

                                    <div className="flex gap-2 pt-2">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            className="flex-1"
                                            onClick={() =>
                                                handleSelectEvent(event)
                                            }
                                        >
                                            <Edit className="h-4 w-4 mr-1" />
                                            Chỉnh sửa
                                        </Button>
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() =>
                                                handleDeleteEvent(
                                                    event.id,
                                                    event.name
                                                )
                                            }
                                        >
                                            <Trash2 className="h-4 w-4" />
                                        </Button>
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}

            {/* Edit Dialog */}
            <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
                <DialogContent className="max-w-2xl">
                    <DialogHeader>
                        <DialogTitle>
                            Edit event - {selectedEvent?.name}
                        </DialogTitle>
                    </DialogHeader>
                    <Tabs defaultValue="info">
                        <TabsList className="grid w-full grid-cols-2">
                            <TabsTrigger value="info">Thông tin</TabsTrigger>
                            <TabsTrigger value="schedule">
                                Lịch và tỷ lệ
                            </TabsTrigger>
                        </TabsList>
                        <TabsContent value="info" className="space-y-4">
                            <div>
                                <Label>Tên sự kiện</Label>
                                <Input defaultValue={selectedEvent?.name} />
                            </div>
                            <div>
                                <Label>Mô tả</Label>
                                <Textarea
                                    defaultValue={selectedEvent?.description}
                                />
                            </div>
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <Label>Thời gian bắt đầu</Label>
                                    <DatePicker
                                        value={eventStartDate}
                                        onChange={setEventStartDate}
                                        placeholder="Chọn ngày bắt đầu"
                                    />
                                </div>
                                <div>
                                    <Label>Thời gian kết thúc</Label>
                                    <DatePicker
                                        value={eventEndDate}
                                        onChange={setEventEndDate}
                                        placeholder="Chọn ngày kết thúc"
                                    />
                                </div>
                            </div>
                        </TabsContent>
                        <TabsContent value="schedule" className="space-y-4">
                            <p className="text-sm text-muted-foreground">
                                Cấu hình tỷ lệ kinh nghiệm và vật phẩm rơi theo khung giờ
                            </p>
                            {selectedEvent?.schedule.map((schedule, index) => (
                                <Card key={index}>
                                    <CardContent className="pt-6">
                                        <div className="space-y-4">
                                            <div>
                                                <Label>Ngày trong tuần</Label>
                                                <Select
                                                    defaultValue={schedule.dayOfWeek.toString()}
                                                >
                                                    <SelectTrigger>
                                                        <SelectValue />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        {DAYS_OF_WEEK.map(
                                                            (day, i) => (
                                                                <SelectItem
                                                                    key={i}
                                                                    value={i.toString()}
                                                                >
                                                                    {day}
                                                                </SelectItem>
                                                            )
                                                        )}
                                                    </SelectContent>
                                                </Select>
                                            </div>
                                            <div className="grid grid-cols-2 gap-4">
                                                <div>
                                                    <Label>Giờ bắt đầu</Label>
                                                    <Input
                                                        type="number"
                                                        min="0"
                                                        max="23"
                                                        defaultValue={
                                                            schedule.startHour
                                                        }
                                                    />
                                                </div>
                                                <div>
                                                    <Label>Giờ kết thúc</Label>
                                                    <Input
                                                        type="number"
                                                        min="0"
                                                        max="23"
                                                        defaultValue={
                                                            schedule.endHour
                                                        }
                                                    />
                                                </div>
                                            </div>
                                            {schedule.expMultiplier !==
                                                undefined && (
                                                <div>
                                                    <Label>
                                                        Hệ số kinh nghiệm: x
                                                        {schedule.expMultiplier}
                                                    </Label>
                                                    <Slider
                                                        defaultValue={[
                                                            schedule.expMultiplier *
                                                                50,
                                                        ]}
                                                        max={500}
                                                        step={50}
                                                        className="mt-2"
                                                    />
                                                </div>
                                            )}
                                            {schedule.dropMultiplier !==
                                                undefined && (
                                                <div>
                                                    <Label>
                                                        Hệ số vật phẩm rơi: x
                                                        {
                                                            schedule.dropMultiplier
                                                        }
                                                    </Label>
                                                    <Slider
                                                        defaultValue={[
                                                            schedule.dropMultiplier *
                                                                50,
                                                        ]}
                                                        max={500}
                                                        step={50}
                                                        className="mt-2"
                                                    />
                                                </div>
                                            )}
                                        </div>
                                    </CardContent>
                                </Card>
                            ))}
                            <Button variant="outline" className="w-full">
                                <Plus className="h-4 w-4 mr-2" />
                                Thêm khung giờ
                            </Button>
                        </TabsContent>
                    </Tabs>
                    <div className="flex justify-end gap-2">
                        <Button
                            variant="outline"
                            onClick={() => setIsEditDialogOpen(false)}
                        >
                            Hủy
                        </Button>
                        <Button onClick={handleSaveEvent}>Lưu thay đổi</Button>
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    )
}
