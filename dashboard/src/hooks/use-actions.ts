"use client"

import { useCallback } from "react"
import { toast as sonnerToast } from "sonner"

export function useEventActions() {
    const toggleEventStatus = useCallback(
        (
            eventId: string,
            events: any[],
            setEvents: (events: any[]) => void
        ) => {
            setEvents(
                events.map((e: any) =>
                    e.id === eventId ? { ...e, active: !e.active } : e
                )
            )

            const event = events.find((e: any) => e.id === eventId)
            const newStatus = !event?.active

            if (newStatus) {
                sonnerToast.success(`Đã bật ${event?.name}`, {
                    description: "Sự kiện đã được kích hoạt",
                })
            } else {
                sonnerToast.warning(`Đã tắt ${event?.name}`, {
                    description: "Sự kiện đã được tắt",
                })
            }
        },
        []
    )

    const createEvent = useCallback(() => {
        sonnerToast.success("Đã tạo sự kiện!", {
            description: "Sự kiện mới đã được tạo và kích hoạt.",
        })
    }, [])

    const saveEvent = useCallback((eventName?: string) => {
        sonnerToast.success("Đã lưu thay đổi", {
            description: `Đã cập nhật cấu hình ${eventName}.`,
        })
    }, [])

    const deleteEvent = useCallback((eventName: string) => {
        sonnerToast.error("Đã xóa sự kiện", {
            description: `Đã xóa ${eventName} khỏi hệ thống.`,
        })
    }, [])

    return {
        toggleEventStatus,
        createEvent,
        saveEvent,
        deleteEvent,
    }
}

export function useServerActions() {
    const restartServer = useCallback((serverName: string) => {
        sonnerToast.success("Đã yêu cầu khởi động lại máy chủ", {
            description: `${serverName} sẽ khởi động lại trong vài phút.`,
        })
    }, [])

    const toggleMaintenance = useCallback(
        (serverName: string, enabled: boolean) => {
            if (enabled) {
                sonnerToast.info("Chế độ bảo trì", {
                    description: `${serverName} đã chuyển sang bảo trì.`,
                })
            } else {
                sonnerToast.success("Máy chủ trực tuyến", {
                    description: `${serverName} đã hoạt động trở lại.`,
                })
            }
        },
        []
    )

    const broadcastMessage = useCallback((message: string) => {
        sonnerToast.success("Đã gửi thông báo", {
            description: `Đã gửi đến tất cả máy chủ: "${message}"`,
        })
    }, [])

    return {
        restartServer,
        toggleMaintenance,
        broadcastMessage,
    }
}

export function usePlayerActions() {
    const banPlayer = useCallback((playerName: string, reason: string) => {
        sonnerToast.error(`Đã khóa tài khoản`, {
            description: `${playerName} đã bị khóa. Lý do: ${reason}`,
        })
    }, [])

    const unbanPlayer = useCallback((playerName: string) => {
        sonnerToast.success(`Đã mở khóa tài khoản`, {
            description: `${playerName} đã được mở khóa.`,
        })
    }, [])

    const mutePlayer = useCallback((playerName: string, duration: number) => {
        sonnerToast.warning(`Đã cấm trò chuyện`, {
            description: `${playerName} bị cấm trò chuyện trong ${duration} phút.`,
        })
    }, [])

    const kickPlayer = useCallback((playerName: string) => {
        sonnerToast.warning(`Đã ngắt kết nối người chơi`, {
            description: `${playerName} đã bị ngắt kết nối khỏi trò chơi.`,
        })
    }, [])

    const compensatePlayer = useCallback(
        (playerName: string, reward: string) => {
            sonnerToast.success(`Đã đền bù`, {
                description: `${playerName} đã nhận: ${reward}`,
            })
        },
        []
    )

    return {
        banPlayer,
        unbanPlayer,
        mutePlayer,
        kickPlayer,
        compensatePlayer,
    }
}

export function useEconomyActions() {
    const updateShopItem = useCallback((itemName: string) => {
        sonnerToast.success(`Đã cập nhật`, {
            description: `Đã cập nhật giá và tồn kho của "${itemName}".`,
        })
    }, [])

    const createShopItem = useCallback(() => {
        sonnerToast.success(`Đã thêm vật phẩm`, {
            description: "Đã thêm vật phẩm mới vào cửa hàng.",
        })
    }, [])

    const deleteShopItem = useCallback((itemName: string) => {
        sonnerToast.error(`Đã xóa vật phẩm`, {
            description: `Đã xóa "${itemName}" khỏi cửa hàng.`,
        })
    }, [])

    return {
        updateShopItem,
        createShopItem,
        deleteShopItem,
    }
}
