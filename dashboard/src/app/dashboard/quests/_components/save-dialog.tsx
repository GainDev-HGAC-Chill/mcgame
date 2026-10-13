"use client"

import { Button } from "@/components/ui/button"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { ActivationWarning } from "./activation"

export function SaveDialog({
    open,
    onOpenChange,
    reason,
    setReason,
    saving,
    onSave,
    sharedCount,
}: {
    open: boolean
    onOpenChange: (open: boolean) => void
    reason: string
    setReason: (reason: string) => void
    saving: boolean
    onSave: () => void
    sharedCount?: number
}) {
    return (
        <Dialog
            open={open}
            onOpenChange={value => {
                if (!saving) onOpenChange(value)
            }}
        >
            <DialogContent className="max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>
                        {sharedCount == null
                            ? "Xác nhận lưu nhiệm vụ"
                            : "Xác nhận sửa vòng lặp dùng chung"}
                    </DialogTitle>
                    <DialogDescription>
                        Lưu cần bản sao lưu đã xác minh; máy chủ ghi ảnh trước
                        thay đổi và nhật ký kiểm toán trong cùng giao dịch. Nếu
                        không đáp ứng, yêu cầu lưu bị từ chối.
                    </DialogDescription>
                </DialogHeader>
                {sharedCount != null && (
                    <p className="font-medium text-destructive">
                        Thay đổi ảnh hưởng {sharedCount} nhiệm vụ dùng chung
                        vòng lặp, không chỉ nhiệm vụ đang mở.
                    </p>
                )}
                <p className="text-sm">
                    Không ghi tiến độ / lịch sử, không cấp lại thưởng. Thay đổi
                    mục tiêu có thể làm mất tiến độ khi runtime đối chiếu: lưu
                    cấu trúc bị chặn khi còn người đang nhận. Không tự bỏ nhiệm
                    vụ hoặc khởi động lại.
                </p>
                <ActivationWarning />
                <label className="grid gap-2 text-sm">
                    Lý do bắt buộc
                    <Textarea
                        value={reason}
                        onChange={event => setReason(event.target.value)}
                        placeholder="Mô tả thay đổi và lý do…"
                    />
                </label>
                <div className="flex justify-end gap-2">
                    <Button
                        variant="outline"
                        disabled={saving}
                        onClick={() => onOpenChange(false)}
                    >
                        Hủy
                    </Button>
                    <Button
                        disabled={
                            saving ||
                            reason.trim().length < 3 ||
                            reason.length > 1000
                        }
                        onClick={onSave}
                    >
                        {saving ? "Đang lưu…" : "Xác nhận lưu DB"}
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    )
}
