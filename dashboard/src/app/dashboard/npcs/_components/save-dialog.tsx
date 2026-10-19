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

export function NpcSaveDialog({
    open,
    onOpenChange,
    reason,
    setReason,
    saving,
    sharedCount,
    onSave,
}: {
    open: boolean
    onOpenChange: (open: boolean) => void
    reason: string
    setReason: (value: string) => void
    saving: boolean
    sharedCount: number
    onSave: () => void
}) {
    return (
        <Dialog
            open={open}
            onOpenChange={value => {
                if (!saving) onOpenChange(value)
            }}
        >
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Xác nhận lưu NPC</DialogTitle>
                    <DialogDescription>
                        Ghi NPC/shop-slot và audit trước/sau trong cùng giao
                        dịch. Không thay đổi người chơi và không tự restart.
                    </DialogDescription>
                </DialogHeader>
                {sharedCount > 1 && (
                    <p className="font-medium text-destructive">
                        Shop dùng chung bởi {sharedCount} NPC. Mọi sửa slot ảnh
                        hưởng tất cả NPC đó.
                    </p>
                )}
                <p className="rounded-md border border-amber-500 p-3 text-sm">
                    Sau khi lưu cần restart máy chủ có kiểm soát; client kết nối
                    lại để nhận shop và vào lại map để nhận đối thoại. Nhiệm vụ
                    Flash vẫn bị giới hạn.
                </p>
                <label className="grid gap-2 text-sm">
                    Lý do bắt buộc
                    <Textarea
                        value={reason}
                        maxLength={1000}
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
                        disabled={saving || reason.trim().length < 3}
                        onClick={onSave}
                    >
                        {saving ? "Đang lưu…" : "Xác nhận lưu DB"}
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    )
}
