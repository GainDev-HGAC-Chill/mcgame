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
- Tự commit hoặc push tự động .
## Tài liệu tham khảo về lối chơi (Gameplay)

- https://web.4399.com/mlxt/
Tham khảo từ phiên bản gốc tiếng Trung.

- https://vuaphapthuat.go.vn/
Tham khảo về lối chơi và các tính năng trong phiên bản tiếng Việt.

- https://www.youtube.com/@Magiccampusneo
tham khảo các tính năng trực tiếp như thưởng vật phẩm yêu cầu các thứ.

Khi nghiên cứu hoặc triển khai các tính năng trò chơi:
- Tham khảo cả ba trang web khi cần thiết.
- Đối chiếu các mô tả về lối chơi, hiệu ứng kỹ năng,
hệ thống thú cưng, trang bị, nhiệm vụ và các hoạt động.
- Kiểm chứng cơ chế thông qua tài liệu/client,
các bảng dữ liệu trò chơi (game-data tables) và mã nguồn hiện tại.
- Lưu ý đến những khác biệt giữa các phiên bản trò chơi.
- Tuyệt đối không coi các mô tả trên trang web là căn cứ xác thực
cho các giao thức RPC hoặc thông số kỹ thuật cơ sở dữ liệu.
- Ghi lại các đường dẫn (URL) nguồn có liên quan vào tài liệu nghiên cứu tính năng.

Mỗi lần cập nhật checkpoint chỉ chỉnh những mục thay đổi, không viết lại toàn bộ file.