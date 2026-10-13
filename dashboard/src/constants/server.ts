import { CheckCircle, XCircle, AlertTriangle } from "lucide-react"

export const SERVER_STATUS_CONFIG = {
    online: { label: "Trực tuyến", color: "success", icon: CheckCircle },
    offline: { label: "Ngoại tuyến", color: "destructive", icon: XCircle },
    maintenance: {
        label: "Bảo trì",
        color: "warning",
        icon: AlertTriangle,
    },
} as const

export const RESOURCE_THRESHOLDS = {
    HIGH: 80,
    cpu: { warning: 70, critical: 90 },
    memory: { warning: 70, critical: 85 },
    players: { warning: 4500, critical: 4800 },
} as const
