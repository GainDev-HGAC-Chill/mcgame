import type { Metadata } from "next"
import { Inter } from "next/font/google"
import "./globals.css"
import { QueryProvider } from "@/components/query-provider"
import { Toaster } from "@/components/ui/sonner"
import { ThemeProvider } from "@/components/theme-provider"
import { TooltipProvider } from "@/components/ui/tooltip"

const inter = Inter({ subsets: ["latin"] })

export const metadata: Metadata = {
    title: "Bảng quản trị GM - Quản lý máy chủ trò chơi",
    description: "Bảng quản trị máy chủ trò chơi",
}

export default function RootLayout({
    children,
}: {
    children: React.ReactNode
}) {
    return (
        <html lang="vi" suppressHydrationWarning>
            <body className={inter.className}>
                <QueryProvider>
                    <ThemeProvider
                        defaultTheme="system"
                        storageKey="dashboard-theme"
                    >
                        <TooltipProvider delayDuration={150}>
                            {children}
                            <Toaster />
                        </TooltipProvider>
                    </ThemeProvider>
                </QueryProvider>
            </body>
        </html>
    )
}
