"use client"

import { startTransition, useState } from "react"
import { Bell, Search, User } from "lucide-react"
import { usePathname, useRouter } from "next/navigation"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { ThemeToggle } from "@/components/theme-toggle"
import { createClient } from "@/lib/supabase/client"
import { APP_ROUTES } from "@/lib/routes"

const PAGE_HEADER_META: Record<string, { title: string; description: string }> =
    {
        "/dashboard": {
            title: "Bảng quản trị",
            description: "Tổng quan máy chủ trò chơi",
        },
        "/dashboard/players": {
            title: "Quản lý người chơi",
            description: "Tìm kiếm và xem thông tin chi tiết người chơi",
        },
        "/dashboard/items": {
            title: "Quản lý vật phẩm",
            description: "Tra cứu tất cả vật phẩm trong trò chơi",
        },
        "/dashboard/box-items": {
            title: "Quản lý hộp vật phẩm",
            description:
                "Tra cứu vật phẩm có thể sử dụng và cấu hình phần thưởng",
        },
        "/dashboard/economy": {
            title: "Quản lý kinh tế",
            description:
                "Theo dõi giao dịch, quản lý cửa hàng và kiểm tra cảnh báo kinh tế",
        },
        "/dashboard/events": {
            title: "Quản lý sự kiện",
            description: "Cấu hình và quản lý sự kiện trong trò chơi",
        },
        "/dashboard/gift-codes": {
            title: "Quản lý mã quà tặng",
            description: "Tạo và quản lý mã quà tặng",
        },
        "/dashboard/server": {
            title: "Quản lý máy chủ",
            description: "Theo dõi và quản lý trạng thái máy chủ trò chơi",
        },
        "/dashboard/actions": {
            title: "Thao tác GM",
            description: "Thực hiện thao tác quản lý người chơi",
        },
        "/dashboard/map-viewer": {
            title: "Trình xem ô bản đồ",
            description:
                "Xem và kiểm tra các ô bản đồ cùng thông tin tọa độ",
        },
    }

function resolvePageMeta(pathname: string) {
    if (pathname.startsWith("/dashboard/box-items/")) {
        return {
            title: "Phần thưởng hộp vật phẩm",
            description:
                "Chỉnh sửa trực tiếp danh sách phần thưởng của vật phẩm đã chọn",
        }
    }
    if (pathname === "/dashboard/boss-loot") {
        return {
            title: "Phần thưởng thủ lĩnh",
            description:
                "Cấu hình phần thưởng thủ lĩnh theo lịch và hằng ngày: vật phẩm, tiền tệ và kinh nghiệm",
        }
    }
    if (pathname.startsWith("/dashboard/boss-loot/")) {
        return {
            title: "Chỉnh sửa phần thưởng thủ lĩnh",
            description:
                "Điều chỉnh tỷ lệ rơi, số lượng, bộ lọc cấp/nguồn và loại phần thưởng của thủ lĩnh",
        }
    }

    return (
        PAGE_HEADER_META[pathname] ?? {
            title: "Bảng quản trị",
            description: "Quản trị và vận hành hệ thống trò chơi",
        }
    )
}

type HeaderProps = {
    adminUser: string
}

export function Header({ adminUser }: HeaderProps) {
    const pathname = usePathname()
    const router = useRouter()
    const [isLoggingOut, setIsLoggingOut] = useState(false)
    const pageMeta = resolvePageMeta(pathname)

    async function handleLogout() {
        setIsLoggingOut(true)

        try {
            const supabase = createClient()
            await supabase.auth.signOut()
        } finally {
            startTransition(() => {
                router.replace(APP_ROUTES.login)
                router.refresh()
            })
            setIsLoggingOut(false)
        }
    }

    return (
        <header className="sticky top-0 z-10 border-b bg-background px-6 py-3">
            <div className="flex items-center gap-4">
                <div className="flex min-w-0 flex-1 items-center gap-4">
                    <div className="min-w-0 flex-1">
                        <h1 className="truncate text-lg font-semibold leading-6">
                            {pageMeta.title}
                        </h1>
                        <p className="truncate text-sm text-muted-foreground">
                            {pageMeta.description}
                        </p>
                    </div>

                    <div className="relative hidden w-full max-w-md lg:block">
                        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                            type="search"
                            placeholder="Tìm người chơi, vật phẩm, giao dịch..."
                            className="pl-10"
                        />
                    </div>
                </div>

                <div className="flex shrink-0 items-center gap-4">
                    <ThemeToggle />

                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button
                                variant="ghost"
                                size="icon"
                                aria-label="Thông báo"
                                className="relative"
                            >
                                <Bell className="h-5 w-5" />
                                <Badge className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full p-0 text-xs">
                                    3
                                </Badge>
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-80">
                            <DropdownMenuLabel>Thông báo</DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            <div className="max-h-96 overflow-y-auto">
                                <DropdownMenuItem className="flex flex-col items-start gap-1">
                                    <div className="flex items-center gap-2">
                                        <Badge variant="warning">Cảnh báo</Badge>
                                        <span className="text-xs text-muted-foreground">
                                            2 phút trước
                                        </span>
                                    </div>
                                    <p className="text-sm">
                                        Phát hiện giao dịch đáng ngờ từ
                                        người chơi xXGamerXx
                                    </p>
                                </DropdownMenuItem>
                                <DropdownMenuItem className="flex flex-col items-start gap-1">
                                    <div className="flex items-center gap-2">
                                        <Badge variant="success">Thành công</Badge>
                                        <span className="text-xs text-muted-foreground">
                                            15 phút trước
                                        </span>
                                    </div>
                                    <p className="text-sm">
                                        Đã hoàn tất bảo trì máy chủ
                                    </p>
                                </DropdownMenuItem>
                                <DropdownMenuItem className="flex flex-col items-start gap-1">
                                    <div className="flex items-center gap-2">
                                        <Badge variant="destructive">
                                            Báo động
                                        </Badge>
                                        <span className="text-xs text-muted-foreground">
                                            1 giờ trước
                                        </span>
                                    </div>
                                    <p className="text-sm">
                                        Máy chủ 2 sử dụng CPU trên 90%
                                    </p>
                                </DropdownMenuItem>
                            </div>
                        </DropdownMenuContent>
                    </DropdownMenu>

                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button
                                variant="ghost"
                                className="relative h-9 w-9 rounded-full"
                                aria-label="Tài khoản quản trị"
                            >
                                <Avatar className="h-9 w-9">
                                    <AvatarImage
                                        src="/avatar.png"
                                        alt={adminUser}
                                    />
                                    <AvatarFallback>
                                        <User className="h-5 w-5" />
                                    </AvatarFallback>
                                </Avatar>
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-56">
                            <DropdownMenuLabel>Tài khoản của tôi</DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem disabled>
                                <User className="mr-2 h-4 w-4" />
                                <span>{adminUser}</span>
                            </DropdownMenuItem>
                            <DropdownMenuItem disabled>
                                <Bell className="mr-2 h-4 w-4" />
                                <span>Phiên quản trị đang hoạt động</span>
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                                disabled={isLoggingOut}
                                onClick={() => void handleLogout()}
                            >
                                <span>
                                    {isLoggingOut
                                        ? "Đang đăng xuất..."
                                        : "Đăng xuất"}
                                </span>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </div>
        </header>
    )
}
