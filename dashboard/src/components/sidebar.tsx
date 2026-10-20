"use client"

import { useState } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import {
    CalendarCheck,
    CalendarDays,
    ClipboardList,
    ChevronLeft,
    ChevronRight,
    Gift,
    MapPin,
    Network,
    Package,
    PackageOpen,
    Settings,
    Shield,
    Swords,
    Users,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

type NavItem = {
    title: string
    href: string
    icon: typeof Shield
    children?: NavItem[]
}

const navItems: NavItem[] = [
    {
        title: "Kênh máy chủ",
        href: "/dashboard",
        icon: Network,
        children: [
            {
                title: "Cấu hình trò chơi",
                href: "/dashboard/game-config",
                icon: Settings,
            },
        ],
    },
    {
        title: "Quản lý người chơi",
        href: "/dashboard/players",
        icon: Users,
    },
    {
        title: "Mã quà tặng",
        href: "/dashboard/gift-codes",
        icon: Gift,
    },
    {
        title: "Quản lý vật phẩm",
        href: "/dashboard/items",
        icon: Package,
    },
    {
        title: "Hộp vật phẩm",
        href: "/dashboard/box-items",
        icon: PackageOpen,
    },
    {
        title: "Quản lý nhiệm vụ",
        href: "/dashboard/quests",
        icon: ClipboardList,
    },
    {
        title: "Quản lý NPC",
        href: "/dashboard/npcs",
        icon: MapPin,
    },
    {
        title: "Shop hệ thống",
        href: "/dashboard/system-shop",
        icon: Package,
    },
    {
        title: "Bảng vật phẩm rơi",
        href: "/dashboard/loot",
        icon: Swords,
    },
    {
        title: "Phần thưởng thủ lĩnh",
        href: "/dashboard/boss-loot",
        icon: Swords,
    },
    {
        title: "Hoạt động",
        href: "/dashboard/activities",
        icon: CalendarDays,
    },
    {
        title: "Phần thưởng điểm danh",
        href: "/dashboard/daily-signin-rewards",
        icon: CalendarCheck,
    },
    {
        title: "Vật thể bản đồ",
        href: "/dashboard/scene-items",
        icon: MapPin,
    },
]

function isActiveNavItem(pathname: string, href: string): boolean {
    return (
        pathname === href ||
        (href !== "/dashboard" && pathname.startsWith(`${href}/`))
    )
}

type SidebarProps = {
    adminUser: string
}

export function Sidebar({ adminUser }: SidebarProps) {
    const pathname = usePathname()
    const [collapsed, setCollapsed] = useState(false)

    return (
        <div
            className={cn(
                "flex flex-col border-r bg-card transition-all duration-300",
                collapsed ? "w-16" : "w-64"
            )}
        >
            <div className="flex h-16 items-center justify-between border-b px-4">
                {!collapsed ? (
                    <div className="flex items-center gap-2">
                        <Shield className="h-6 w-6 text-primary" />
                        <span className="text-lg font-bold">
                            Bảng quản trị GM
                        </span>
                    </div>
                ) : null}
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setCollapsed(current => !current)}
                    aria-label={
                        collapsed
                            ? "Mở rộng thanh điều hướng"
                            : "Thu gọn thanh điều hướng"
                    }
                    className="h-8 w-8"
                >
                    {collapsed ? (
                        <ChevronRight className="h-4 w-4" />
                    ) : (
                        <ChevronLeft className="h-4 w-4" />
                    )}
                </Button>
            </div>

            <nav className="flex-1 space-y-1 p-2">
                {navItems.map(item => (
                    <NavEntry
                        key={item.href}
                        item={item}
                        pathname={pathname}
                        collapsed={collapsed}
                        depth={0}
                    />
                ))}
            </nav>

            <div className="border-t p-4">
                {!collapsed ? (
                    <div className="text-xs text-muted-foreground">
                        <p>Phiên quản trị:</p>
                        <p className="font-medium text-foreground">
                            {adminUser}
                        </p>
                    </div>
                ) : null}
            </div>
        </div>
    )
}

type NavEntryProps = {
    item: NavItem
    pathname: string
    collapsed: boolean
    depth: number
}

function NavEntry({ item, pathname, collapsed, depth }: NavEntryProps) {
    const Icon = item.icon
    const isActive = isActiveNavItem(pathname, item.href)

    return (
        <>
            <Link
                href={item.href}
                aria-label={item.title}
                className={cn(
                    "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                        ? "bg-primary text-primary-foreground"
                        : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                    collapsed && "justify-center",
                    !collapsed && depth > 0 && "ml-4"
                )}
            >
                <Icon className="h-5 w-5 shrink-0" />
                {!collapsed ? <span>{item.title}</span> : null}
            </Link>
            {!collapsed && item.children
                ? item.children.map(child => (
                      <NavEntry
                          key={child.href}
                          item={child}
                          pathname={pathname}
                          collapsed={collapsed}
                          depth={depth + 1}
                      />
                  ))
                : null}
        </>
    )
}
