"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { ApiError, fetchAPI } from "@/services/api"
import { DASHBOARD_API_ENDPOINTS } from "@/lib/api/endpoints"
import type {
    PlayerActionRequest,
    PlayerActionResponse,
    PlayerDetailData,
} from "@/types/player-management"

async function fetchPlayerDetail(playerId: string): Promise<PlayerDetailData> {
    const actionPayload: PlayerActionRequest = {
        action: "player_detail",
        targetId: Number(playerId),
    }

    const response = await fetchAPI<PlayerActionResponse>(
        DASHBOARD_API_ENDPOINTS.playerAction(playerId),
        {
            method: "POST",
            body: JSON.stringify(actionPayload),
        }
    )

    if (!response.ok || !response.data) {
        throw new Error(
            response.statusMessage || "Không tải được chi tiết người chơi"
        )
    }

    return response.data as PlayerDetailData
}

export async function executePlayerAction(
    playerId: number,
    action: string,
    payload?: Record<string, unknown>
): Promise<PlayerActionResponse> {
    const actionPayload: PlayerActionRequest = {
        action,
        targetId: playerId,
        payload,
    }

    const response = await fetchAPI<PlayerActionResponse>(
        DASHBOARD_API_ENDPOINTS.playerAction(String(playerId)),
        {
            method: "POST",
            body: JSON.stringify(actionPayload),
        }
    )
    if (!response.ok) {
        throw new ApiError(response.statusMessage || "Thao tác không thành công", 200, response)
    }
    return response
}

export function usePlayerDetail(playerId: string) {
    return useQuery({
        queryKey: ["player-detail", playerId],
        queryFn: () => fetchPlayerDetail(playerId),
        refetchInterval: 5000,
        staleTime: 3000,
        enabled: Boolean(playerId),
    })
}

function humanizeAction(action: string): string {
    const labels: Record<string, string> = {
        kick: "Ngắt kết nối người chơi", transport: "Di chuyển người chơi",
        set_currency: "Cập nhật tiền tệ", set_attributes: "Cập nhật thuộc tính",
        advance_progression: "Tăng tiến trình", set_progression: "Đặt tiến trình",
        set_target_level: "Đặt cấp độ", set_combat_stats: "Làm mới chỉ số chiến đấu",
        set_resources: "Cập nhật sức chứa", set_dress_panel: "Cập nhật thời trang",
        set_life_skills: "Cập nhật kỹ năng sống", set_gm_level: "Đặt cấp GM",
        remove_item: "Xóa vật phẩm", update_item: "Cập nhật vật phẩm",
        update_pet: "Cập nhật thú cưng", delete_pet: "Xóa thú cưng",
        set_feature_state: "Cập nhật hệ thống", set_relationship: "Cập nhật quan hệ",
        delete_relationship: "Xóa quan hệ",
    }
    return labels[action] ?? "Thao tác"
}

export function usePlayerAction(playerId: number) {
    const queryClient = useQueryClient()

    return useMutation({
        mutationFn: ({
            action,
            payload,
        }: {
            action: string
            payload?: Record<string, unknown>
        }) => executePlayerAction(playerId, action, payload),
        onSuccess: response => {
            void queryClient.invalidateQueries({
                queryKey: ["player-detail", String(playerId)],
            })
            toast.success(`${humanizeAction(response.action)} thành công`, {
                description: response.action === "set_target_level" ? response.statusMessage : response.deliveryMode === "database_only"
                    ? "Đã lưu vào cơ sở dữ liệu. Có thể cần mở lại bảng hoặc đăng nhập lại để thấy thay đổi."
                    : response.deliveryMode === "live_session"
                      ? "Đã cập nhật và gửi đến phiên chơi trực tuyến."
                      : "Đã áp dụng thay đổi.",
            })
        },
        onError: (error: Error) => {
            toast.error(error.message)
        },
    })
}
