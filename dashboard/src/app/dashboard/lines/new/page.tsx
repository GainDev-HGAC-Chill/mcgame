import { LineForm } from "@/components/lines/line-form"

export default function NewLinePage() {
    return (
        <div className="space-y-6 p-6">
            <div>
                <h1 className="text-2xl font-bold">Thêm kênh kết nối</h1>
                <p className="text-sm text-muted-foreground">
                    Thêm bản ghi vào public.gateway_config. Máy chủ đang chạy cập nhật qua LISTEN/NOTIFY.
                </p>
            </div>
            <LineForm mode="create" />
        </div>
    )
}
