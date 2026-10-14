BLOODBAG PATCH - THỬ NGHIỆM, CHƯA HOÀN CHỈNH

Đã sửa:
- UseBloodBag tính giới hạn HP có bonus trang bị, không hồi miễn phí 500 HP.
- Cập nhật HP tại player.character_attributes và trừ characterHp trong cùng transaction PostgreSQL.
- Không gửi callback setBloodBag không tồn tại trong com.qeedoo.game.system.CallBack.

CHƯA giải quyết:
- Client vẫn chưa có callback hợp lệ để cập nhật thanh Sinh Mệnh Hoàn tức thời, nên cần thay đổi SWF hoặc xác định callback sẵn có.
- character_repository_scan.go vẫn giới hạn HP về HP cơ bản khi nạp nhân vật. Phải thiết kế bổ sung một luồng tải HP tối đa hiệu lực trước khi dùng trên server thật.
- Chưa chạy go test vì ZIP không có go.mod và toàn bộ dự án. Cần test ở máy local trước khi triển khai.
- Tránh triển khai khi có người chơi: các luồng hồi phục khác hoặc cập nhật HP đồng thời cần được audit thêm.

ỨNG DỤNG: Sao lưu 3 file; giải nén ZIP vào gốc I:\mcgame, giữ cấu trúc thư mục.
KIỂM TRA: gofmt, go test ./internal/application/bloodbag/... ./internal/presentation/rtmp/handlers/combat/... ./internal/infrastructure/persistence/postgres/... ; go build ./cmd/gameserver
