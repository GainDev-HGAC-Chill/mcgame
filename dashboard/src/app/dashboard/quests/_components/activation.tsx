export function ActivationWarning({ saved = false }: { saved?: boolean }) {
    return (
        <div
            role="status"
            className="space-y-2 rounded-lg border border-amber-500 bg-amber-500/10 p-4 text-sm"
        >
            <p className="font-semibold">
                {saved
                    ? "Đã lưu vào DB · CHƯA kích hoạt đồng bộ trong game"
                    : "Chỉnh sửa danh mục · không tự kích hoạt trong game"}
            </p>
            <p>
                Backend cần nạp lại danh mục, nhưng kích hoạt hiện BỊ CHẶN vì
                chưa có đường xuất bản danh mục Flash được xác minh. Không khởi
                động lại mcgame-server để kích hoạt các chỉnh sửa lúc này. Khởi
                động lại backend hoặc đăng nhập lại không đảm bảo đồng bộ danh
                mục nhúng.
            </p>
            <p>
                Yêu cầu, phần thưởng, NPC và điều kiện tiên quyết có thể khác
                giữa Flash và backend cho đến khi danh mục client được xuất bản.
                Không coi việc lưu DB là đã áp dụng hoàn chỉnh.
            </p>
        </div>
    )
}
