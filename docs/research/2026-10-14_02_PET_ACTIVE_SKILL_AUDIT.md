# Pet Active Skill Audit

## Phạm vi

- Nguồn roster: skill book Pet `TBL_ITEM_TEMPLATE.type=506,useType=2` ghép theo tên với `TBL_SKILL.kind=1`.
- Tổng cộng: 144 template level, 52 họ skill chủ động.
- Đối chiếu: `TBL_SKILL`, `TBL_BUFF`, mô tả hiển thị và combat runtime hiện tại.
- Không thay đổi dữ liệu hoặc source trong bước kiểm tra này.

## Kết luận chắc chắn

### Số mục tiêu

- 49/52 họ có `targetType/targetNum/areaAttack` mà resolver hiện tại có thể chọn đúng số lượng theo dữ liệu: đơn thể, dọc tối đa 2, ngang tối đa 5, chữ thập tối đa 4, hoặc toàn bộ tối đa 10.
- Ba họ `targetType=11` đang sai chắc chắn trong server: `Đoạt Mệnh`, `Liêu Hồn`, `Nuốt Chửng`. Server biến type 11 thành toàn bộ sinh vật sống của cả hai phe và bỏ qua `targetNum=1`.
- `areaAttack=4` hiện không shuffle, nhưng các skill Pet dùng kiểu này đều có `targetNum=10`; trong đội hình tối đa 10 mục tiêu cùng predicate nên hiện chưa làm sai số lượng thực tế.

### Sát thương và hồi phục

- Cả 144 template không có trường combat multiplier riêng đang được server đọc đúng.
- Adapter hiện dùng `expSkill` làm base damage và `dexSkill` làm multiplier. Hai trường này là điều kiện tiến triển/học skill, không phải công thức chiến đấu.
- 113/144 template có `expSkill=1,dexSkill=-1`, nên runtime gần như gây lượng damage/heal cố định bằng 1 thay vì phần trăm trong mô tả.
- 31/144 template có `expSkill=0,dexSkill=-1`, nên fallback thành khoảng `100% attack + 10`, không khớp các mức 30%–190% trong mô tả.
- `attackNum` đã được lặp đúng số hit; `Dã Thú Cuồng Vũ` chạy 2/2/3 hit, nhưng damage mỗi hit vẫn sai.
- Skill type 5 được đi qua healing path, nhưng trị số hồi phục vẫn sai. `Thanh Phong Phục Sinh` chưa có revive path.
- Các công thức đặc biệt theo phần trăm HP/MP hiện không tồn tại: `Hủy Diệt`, `Phóng Thích Năng Lượng`, `Đoạt Mệnh`, `Liêu Hồn`, `Quang Hợp`, `Ma Pháp Thuẫn`.

### Hiệu ứng

- Runtime đang ánh xạ trực tiếp `buffId=1..10` sang status. Dữ liệu Pet dùng các buff ID như 14, 16, 121, 187, 729, 934, 1362... nên hầu hết trả về `StatusNormal` và bị bỏ qua.
- Cần đọc `TBL_BUFF.state`, `kind`, `percentFlag`, `prop1..6`, `propNum1..6`; runtime hiện chưa làm việc này.
- Buff/debuff stat (công, thủ, tốc độ, né, bạo kích, miễn giảm) chưa được áp vào Participant.
- DoT/HoT theo trị số mô tả chưa được hỗ trợ; poison hiện chỉ có hardcode 5% max HP.
- Shield, taunt, nghỉ một lượt, tự hi sinh, hồi sinh, dispel/steal buff và hồi HP/MP khi dispel đều chưa có implementation.
- Type 6/7 hiện vẫn đi qua damage path, nên skill thuần buff/debuff có thể gây thêm damage vật lý ngoài mô tả.
- Tỉ lệ buff AoE hiện roll một lần rồi áp cho toàn bộ mục tiêu, thay vì roll riêng từng mục tiêu.

## Ma trận 52 họ skill

Ký hiệu target: `TT` = targetType, `N` = targetNum, `A` = areaAttack, `H` = attackNum. Tất cả dòng có damage/heal đều đang sai trị số theo kết luận trên; các dòng có buff/special đều chưa khớp hoàn toàn.

| Skill | Target theo data | Kết quả target | Damage/heal | Effect |
|---|---|---|---|---|
| Ác Ma Chi Kích | TT4 N1 | Đúng số lượng | Sai; mô tả không có hệ số | Không đủ mô tả |
| Ảo Hoa Kỳ Hương | TT4 N1 | Đúng | Không được gây damage | Mê Hồn không áp |
| Bát Diệp Trảm | TT4 N1 | Đúng | Sai | Giảm công vật lý không áp |
| Cự Mộc Áp Thân | TT4 N1 | Đúng | Sai 80/90/100% | N/A |
| Cự Mộc Áp Thân Cao Cấp | TT4 N1 | Đúng | Sai 115/135/155% | N/A |
| Dã Thú Cuồng Vũ | TT4 N1 H2/2/3 | Đúng target/hit | Sai mỗi hit | N/A |
| Dập Hỏa Thuật | TT4 N2 A1 | Đúng | Sai 60/70/90% | Giảm thủ không áp |
| Đế Vương Nộ | TT2 N1 | Đúng theo data | Không được gây damage | Tăng công/điều kiện HP không áp |
| Diệp Vũ Chi Thuật Cao Cấp | TT4 N3 A2 | Đúng | Sai 55/65/75% | N/A |
| Diệt Vong Chi Ca | TT4 N1 | Đúng | Sai 80% | N/A |
| Đoạt Mệnh | TT11 N1 | Sai: đánh toàn chiến trường | Sai 130/160/190% và HP scaling | Crit đặc biệt chưa có |
| Độc Huyết Trảo | TT4 N1 | Đúng | Sai 75/105% | Giảm công không áp |
| Độc Thích Đơn Thể | TT4 N1 | Đúng | Không được gây hit trực tiếp | DoT 0.3/0.6/0.9 không áp |
| Độc Thích Quần Thể | TT4 N4/5/10 | Đúng số lượng | Không được gây hit trực tiếp | DoT không áp |
| Duy Trì Tốc Độ | TT8 N1 | Đúng | Không được gây damage | Né tránh không áp |
| Hắc Dạ Ma Ảnh | TT4 N1 | Đúng | Sai 55/75/105% | Giảm công phép không áp |
| Hấp Thu Năng Lượng | TT7 N1 | Đúng non-boss | Không được gây damage | Steal/debuff không áp |
| Hiệp Ước Tử Thần | TT2 N1 | Đúng theo data | Không xác định | Thiếu mô tả/buff |
| Hủy Diệt | TT4 N3 A2 | Đúng | Sai % tổng HP | Hi sinh/điều kiện HP chưa có |
| Khóa Mục Tiêu | TT4 N2 A1 | Đúng | Không được gây damage | Tăng damage nhận vào không áp |
| Kí Sinh Chủng Tử Cao Cấp | TT4 N1 | Đúng | Sai 115/135/155% | Giảm công phép không áp; rate mâu thuẫn |
| Ký Sinh Chủng Tử | TT4 N1 | Đúng | Sai 80/90% | Giảm công phép không áp |
| Liêu Hồn | TT11 N1 | Sai: đánh toàn chiến trường | Sai 120/140/160% và HP scaling | Crit đặc biệt chưa có |
| Lời Ngọt Ngào | TT4 N10 A4 | Đúng theo data | Sai | Mô tả không định lượng |
| Long Khiếu | TT4 N10 A4 | Đúng | Sai 20/22/26% | Hỗn loạn không áp |
| Ma Pháp Thuẫn | TT8 N1 | Đúng | Không được gây damage | Shield và cost 60% MP chưa có |
| Nuốt Chửng | TT11 N1 | Sai: damage toàn chiến trường | Không được gây damage | Dispel/consume/heal chưa có |
| Pháo Kích | TT4 N4 A3 | Đúng theo data | Sai/không đủ mô tả | Choáng buff 14 không áp |
| Phong Hôn Chi Nhẫn | TT4 N3/3/4 A2 | Đúng | Sai 60/75/60% | N/A |
| Phóng Thích Năng Lượng | TT4 N1 | Đúng | Sai % tổng HP | Tự tê liệt 1 lượt chưa có |
| Quang Hợp | TT2 N1 | Đúng theo data | Heal/chuyển HP-MP sai | HoT MP chưa có |
| Quyến Luyến Không Rời | TT4 N3 A2 | Đúng | Sai 36/48/60% | N/A |
| Sương Đông Đơn Thể | TT4 N1 | Đúng | Sai 100/110/120% | Giảm tốc không áp |
| Sương Đông Quần Thể | TT4 N4 A3 | Đúng | Sai 70/75/80% | Giảm tốc không áp |
| Suy Tiến Trang Trí | TT4 N2 A1 | Đúng | Sai 30/40/60% | Choáng gần đúng loại nhưng semantics chưa đủ |
| Suy Yếu Linh Hồn | TT4 N1 | Đúng | Không được gây damage | Giảm miễn vật/phép không áp |
| Tấn Công Cấp Tốc | TT4 N1 | Đúng | Sai; mô tả không định lượng | N/A |
| Tàn Nhẫn Tư Liệt | TT4 N1 | Đúng | Sai 55/75/105% | Chảy máu theo điểm thần chưa có |
| Táng Thần Đơn Thể | TT4 N1 | Đúng | Sai 140/160/190% | Nghỉ 1 lượt chưa có |
| Thanh Phong Phục Sinh | TT8 N1 | Chỉ chọn mục tiêu sống | Sai | Không hồi sinh mục tiêu chết |
| Thích Nhĩ Hô Khiếu | TT4 N1 | Đúng | Sai 120/130% | N/A |
| Thiên Hựu | TT9 N1 | Đúng theo data | Không được gây damage | Thiếu mô tả; buff 1537 cần xác nhận |
| Thú Tính Gào Thét | TT4 N10 A4 | Đúng | Không được gây damage | Giảm thủ vật lý không áp |
| Thuốc Kích Thích | TT2 N1 | Đúng theo data | Không được gây damage | HP cost/tốc/liên kích/bạo kích chưa có |
| Thủy Cam Lộ | TT8 N1 | Đúng | Heal sai 110/130/160% | HoT 75/145/238 chưa có |
| Tích Lực Sơ Cấp | TT2 N1 | Đúng theo data | Không được gây damage | Bạo kích không áp |
| Tích Lực Trung Cấp | TT2 N1 | Đúng theo data | Không được gây damage | Bạo kích không áp |
| Toái Băng Phong Bạo | TT4 N1 | Đúng | Sai 80/90/100% | N/A |
| Trào Phúng Đơn Thể | TT4 N1 | Đúng | Không được gây damage | Taunt chưa có |
| Tự Nhiên Thuẫn Đơn Thể | TT8 N1 | Đúng theo data | Không được gây damage | Thủ/HoT không áp; L3 mô tả mâu thuẫn |
| Tự Nhiên Thuẫn Quần Thể | TT8 N4/5/10 | Đúng | Không được gây damage | Thủ/HoT không áp |
| Vạn Thú Tề Phát | TT8 N1 | Đúng | Không được gây damage | Công/tốc/giảm thủ không áp |

## Điểm dữ liệu mâu thuẫn cần quyết định

1. Phần trăm sát thương nên là multiplier của `Attack`/`MagicAttack` trước pipeline phòng thủ, chí mạng, nguyên tố và variance hay là phần trăm damage cuối.
2. `Đoạt Mệnh` và `Liêu Hồn`: mô tả nói một mục tiêu nhưng data dùng `targetType=11`; cần xác nhận đơn mục tiêu hay toàn chiến trường.
3. `Nuốt Chửng`: cần xác nhận chọn một mục tiêu, toàn bộ hai phe, hay nuốt đúng một trạng thái từ mỗi phía.
4. Khi description skill và `TBL_BUFF` khác nhau, cần nguồn ưu tiên:
   - Bát Diệp Trảm L3: description giảm 25%, buff giảm 33%.
   - Kí Sinh Chủng Tử Cao Cấp: description rate 25/40/55%, `buffRate=10%` cả ba level.
   - Hấp Thu Năng Lượng L3: description/buff text 20%, `propNum=-25%`.
   - Tự Nhiên Thuẫn Đơn Thể L3: description nói đồng đội hàng ngang nhưng `targetNum=1`.
5. Các skill thiếu hoặc hỏng mô tả cần quy tắc riêng: Ác Ma Chi Kích, Lời Ngọt Ngào, Hiệp Ước Tử Thần, Thiên Hựu, phần cuối Pháo Kích.

## Quyết định người dùng đã xác nhận

- `X% sát thương vật lý/ma pháp` = `X% × Attack/MagicAttack` của Pet trước pipeline phòng thủ, chí mạng, nguyên tố và variance.
- `Đoạt Mệnh` và `Liêu Hồn` chỉ đánh một mục tiêu được chọn; proc đặc biệt dựa trên HP hiện tại của mục tiêu.
- `Nuốt Chửng` xử lý một mục tiêu được chọn: một debuff của đồng đội hoặc một buff của kẻ địch, sau đó hồi HP/MP một lần.
- Khi mô tả skill mâu thuẫn với `TBL_SKILL`/`TBL_BUFF`, ưu tiên hành vi trong mô tả mà người chơi nhìn thấy.

## Bằng chứng bổ sung cho skill thiếu mô tả

- Book `Thiên Hựu` ghi rõ: khiến một nhân vật đồng đội rơi vào trạng thái vô địch nhưng không thể hành động. Skill dùng `targetType=9`, một mục tiêu, hai lượt.
- `Hiệp Ước Tử Thần` không liên kết `buffId`, nhưng catalog có ba buff cùng tên tăng cả công vật lý, công ma pháp và phòng thủ lần lượt 50/75/100%. Skill catalog hiện chỉ có level 1 và duration 3 lượt, nên chưa đủ bằng chứng chọn tier nào.
- Book `Ác Ma Chi Kích` chỉ ghi “tấn công ma pháp”; không có multiplier.
- Book `Lời Ngọt Ngào` chỉ ghi “tấn công ma pháp”; handbook xác nhận đánh đồng thời 10 mục tiêu nhưng không có multiplier.

## Quy tắc bổ sung do người dùng chốt

- Proc của `Đoạt Mệnh`/`Liêu Hồn` cộng thêm lần lượt 5/10/15% HP hiện tại của mục tiêu ở level 1/2/3; xác suất proc vẫn là 10/20/30% theo mô tả.
- `Ác Ma Chi Kích` và `Lời Ngọt Ngào` dùng multiplier 100/110/120% cho level 1/2/3.
- `Hiệp Ước Tử Thần` level 1 tăng 50% công vật lý, công ma pháp và phòng thủ trong 3 lượt.
- `Duy Trì Tốc Độ` tăng né tránh 10/20/30%; `Suy Yếu Linh Hồn` giảm miễn thương vật lý và ma pháp 10/20/30%.

## Bổ sung sau audit: skill Pet đặc thù không thuộc roster sách

- Audit 144 template ban đầu được suy ra từ item sách Pet `type=506,useType=2`, nên bỏ sót các skill gắn qua handbook/creature hoặc vật phẩm Pet đặc thù.
- `Hổ Khiếu Sơn Lâm` 7026/7027/7028: toàn bộ địch tối đa 10; physical 80/100/120%; mỗi mục tiêu còn sống có 30/40/50% xóa một buff có lợi ngẫu nhiên.
- `Thợ Săn Sát Thủ` 7014/7015/7016: toàn đội tối đa 10; Speed và Accuracy +10/15/20% trong 5 lượt. Mô tả level 2 bị lỗi ký tự “`5%”; `TBL_BUFF` 80119 và chuỗi level xác nhận 15%.
- `Tứ Phương Khiếp Đảm` 7011/7012/7013: 4/6/8 địch ngẫu nhiên; physical Attack, MagicAttack và Speed −5/8/12% trong 1 lượt; mỗi mục tiêu có 10/20/30% xóa toàn bộ buff có lợi có thể giải.
