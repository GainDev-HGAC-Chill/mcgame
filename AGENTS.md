## Quy tắc lưu tiến độ bắt buộc

Đối với mọi nhiệm vụ sửa lỗi hoặc phát triển tính năng:

- Trước khi bắt đầu: đọc WORK_STATE.md và các mục bộ nhớ liên quan.
- Sau khi hoàn thành một bước sửa code có ý nghĩa: cập nhật WORK_STATE.md.
- Sau khi hoàn thành một nhóm lỗi: cập nhật FIX_HISTORY.md và trạng thái kiểm thử.
- Trước các thao tác rủi ro như migration, restart hoặc thay đổi dữ liệu: ghi checkpoint trước.
- Không chờ đến cuối phiên mới lưu tiến độ.
- Ghi rõ file đã sửa, nguyên nhân, bước hoàn thành, bước tiếp theo và các kiểm tra chưa chạy.
- Nếu phiên bị ngắt, phiên sau phải đối chiếu checkpoint với Git diff trước khi làm tiếp.
- Không ghi toàn bộ source, log hoặc nội dung phân tích dài vào checkpoint.
- Không commit hoặc push tự động nếu chưa được yêu cầu.

Mỗi lần cập nhật checkpoint chỉ chỉnh những mục thay đổi, không viết lại toàn bộ file.