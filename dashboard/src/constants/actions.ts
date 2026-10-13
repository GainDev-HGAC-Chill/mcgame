/**
 * GM Actions constants
 */

export const ACTION_TYPES = {
    ban: {
        label: "Khóa tài khoản",
        color: "destructive",
        description: "Vĩnh viễn hoặc có thời hạn",
    },
    unban: {
        label: "Mở khóa",
        color: "default",
        description: "Gỡ khóa tài khoản",
    },
    mute: {
        label: "Cấm trò chuyện",
        color: "secondary",
        description: "Tắt trò chuyện",
    },
    kick: {
        label: "Ngắt kết nối",
        color: "outline",
        description: "Buộc đăng xuất khỏi trò chơi",
    },
    compensate: {
        label: "Đền bù",
        color: "warning",
        description: "Gửi phần thưởng hoặc tiền tệ đền bù",
    },
} as const

export const ACTION_STATUS = {
    active: { label: "Đang hoạt động", variant: "destructive" },
    expired: { label: "Hết hạn", variant: "secondary" },
    completed: { label: "Hoàn tất", variant: "success" },
} as const

export const BAN_DURATIONS = [
    { value: "30", label: "30 phút" },
    { value: "60", label: "1 giờ" },
    { value: "240", label: "4 giờ" },
    { value: "1440", label: "1 ngày" },
    { value: "10080", label: "7 ngày" },
    { value: "43200", label: "30 ngày" },
    { value: "-1", label: "Vĩnh viễn" },
]

export const COMPENSATION_TYPES = {
    currency: { label: "Tiền tệ", description: "Vàng, đá quý, xu" },
    item: { label: "Vật phẩm", description: "Vật phẩm cụ thể" },
    both: { label: "Cả hai", description: "Tiền tệ và vật phẩm" },
}
