-- Operator-approved starter rewards. Data only; no player writes or schema migration.
-- Each array: source bag, reward table, template, count, quality, prefix, catalog name.
DO $install$
DECLARE
    config jsonb := $starter_awards$
[
 [2918,29,654,5,0,0,"Ngân Phiếu Lễ"],
 [2918,29,125,20,0,0,"Thuốc Nước Bổ Lực"],
 [2918,29,4,20,0,0,"Thuốc Nước Bổ Máu"],
 [2918,19,1009,1,20,5,"Tân Thủ Trang Sức (Trái)"],
 [2918,19,1010,1,20,5,"Tân Thủ Trang Sức (Phải)"],
 [2918,19,1008,1,20,5,"Tân Thủ Giới Chỉ"],
 [2918,19,1007,1,20,5,"Tân Thủ Hạng Liên"],
 [2918,29,298,3,0,0,"Sách Tăng 1.5 Lần EXP (1 Giờ)"],
 [2918,29,2919,1,0,0,"Túi Quà Tân Thủ (cấp 10)"],
 [2919,29,654,5,0,0,"Ngân Phiếu Lễ"],
 [2919,29,2515,20,0,0,"Kết Ước Nguyện"],
 [2919,29,286,50,0,0,"Đông Huyền Hồi Phù"],
 [2919,29,1332,5,0,0,"Đông Huyền Lệnh Kỳ (Lam)"],
 [2919,29,1333,1,0,0,"Đông Huyền Lệnh Kỳ (Tím)"],
 [2919,29,325,1,0,0,"Sách Mở Rộng Túi Đồ"],
 [2919,29,298,3,0,0,"Sách Tăng 1.5 Lần EXP (1 Giờ)"],
 [2919,29,1403,1,0,0,"Thẻ Tiểu Cương Thi"],
 [2919,29,2920,1,0,0,"Túi Quà Tân Thủ (cấp 20)"],
 [2920,29,1515,1,0,0,"Tiêu Diêu Phù (7 ngày)"],
 [2920,29,1132,1,0,0,"Bạch Ngân Lễ (10 Vạn)"],
 [2920,29,1365,1,0,0,"Sách Huấn Luyện Pet Cấp Thấp"],
 [2920,29,326,1,0,0,"Sách Mở Rộng Túi Đựng Pet"],
 [2920,29,721,10,0,0,"Hình Nhân"],
 [2920,29,324,1,0,0,"Sách Mở Rộng Kho"],
 [2920,29,368,3,0,0,"Sách Tăng 1.5 Lần EXP (3 Giờ)"],
 [2920,29,961,3,0,0,"Thuốc Tăng Trưởng Pet Loại A"],
 [2920,29,2921,1,0,0,"Túi Quà Tân Thủ (cấp 30)"],
 [2921,29,274,2,0,0,"Nội Đơn Bách Luyện"],
 [2921,29,1615,1,0,0,"Radar Nhỏ (7 ngày)"],
 [2921,29,1585,1,0,0,"Thuốc Tăng Trưởng Pet Siêu Cấp Loại A"],
 [2921,29,1857,1,0,0,"Sách Gặp Địch Tại Chỗ (1 Giờ)"],
 [2921,29,2049,1,0,0,"Túi Quà Trang Bị Trác Việt Tím Cấp 30"],
 [2921,29,328,3,0,0,"Sách Gặp Địch Điên Cuồng (15 phút)"],
 [2921,29,1140,1,0,0,"Sách Tăng 3 Lần EXP (1 Giờ)"],
 [2921,29,719,10,0,0,"Loa"],
 [2921,29,273,2,0,0,"Chìa Khóa May Mắn"],
 [2921,12,91,1,1.9,0,"Hổ Bì Dương"],
 [2921,29,2922,1,0,0,"Túi Quà Tân Thủ (cấp 40)"],
 [2922,29,127,20,0,0,"Pháp Lực Hoàn"],
 [2922,29,1534,1,0,0,"Thọ Mệnh Đan Cao Cấp"],
 [2922,29,2050,1,0,0,"Túi Quà Trang Bị Trác Việt Tím Cấp 40"],
 [2922,29,1140,1,0,0,"Sách Tăng 3 Lần EXP (1 Giờ)"],
 [2922,29,113,20,0,0,"Sinh Mệnh Hoàn"],
 [2922,29,1064,1,0,0,"Hoàng Kim Lễ"],
 [2922,29,281,2,0,0,"Nội Đơn Băng Thú"],
 [2922,29,2923,1,0,0,"Túi Quà Tân Thủ (cấp 50)"],
 [2923,29,721,5,0,0,"Hình Nhân"],
 [2923,29,1366,1,0,0,"Sách Huấn Luyện Pet Trung Cấp"],
 [2923,29,719,10,0,0,"Loa"],
 [2923,29,16,5,0,0,"Kim Cương"],
 [2923,29,371,20,0,0,"Chiến Thần Thánh Dược"],
 [2923,29,370,20,0,0,"Pháp Thần Thánh Dược"],
 [2923,29,282,2,0,0,"Nội Đơn Tinh Quái"],
 [2923,29,1064,1,0,0,"Hoàng Kim Lễ"]
]
$starter_awards$::jsonb;
BEGIN
    CREATE TEMP TABLE starter_award_input ON COMMIT DROP AS
    SELECT (810000000 + ordinality)::integer AS id,
           (v->>0)::numeric AS item_id, (v->>1)::numeric AS type,
           (v->>2)::numeric AS award_id, (v->>3)::numeric AS count,
           (v->>4)::numeric AS quality, (v->>5)::numeric AS pre_name_type,
           v->>6 AS expected_name,
           100::numeric AS rate,
           '{"guaranteed":true}'::jsonb ||
             CASE WHEN (v->>0)::integer=2923 AND (v->>2)::integer=16
                  THEN '{"materialColorCode":3}'::jsonb ELSE '{}'::jsonb END AS payload
    FROM pg_catalog.jsonb_array_elements(config) WITH ORDINALITY AS input(v, ordinality);

    IF (SELECT count(*) FROM pg_temp.starter_award_input) <> 54
       OR EXISTS (SELECT 1 FROM pg_temp.starter_award_input WHERE count <= 0 OR type NOT IN (12,19,29)) THEN
        RAISE EXCEPTION 'Invalid approved reward manifest';
    END IF;

    IF EXISTS (
        SELECT 1 FROM (VALUES (2918,1),(2919,10),(2920,20),(2921,30),(2922,40),(2923,50)) AS b(id,level)
        LEFT JOIN "data"."data_tbl_item_template" t ON t.id=b.id
        WHERE t.id IS NULL OR t.req_level<>b.level OR t.type<>550 OR t.use_type<>1
    ) THEN
        RAISE EXCEPTION 'Starter bag template/level mismatch';
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_temp.starter_award_input r
        LEFT JOIN (
            SELECT 29 AS type,id,name FROM "data"."data_tbl_item_template"
            UNION ALL SELECT 19,id,name FROM "data"."data_tbl_equipt_template"
            UNION ALL SELECT 12,id,name FROM "data"."data_tbl_creature"
        ) t ON t.type=r.type AND t.id=r.award_id
        WHERE t.id IS NULL OR
          regexp_replace(lower(btrim(t.name)), '\s+', ' ', 'g') <>
          regexp_replace(lower(btrim(r.expected_name)), '\s+', ' ', 'g')
    ) THEN
        RAISE EXCEPTION 'Reward ID/name mismatch; no data changed';
    END IF;

    LOCK TABLE "data"."data_tbl_item_award" IN SHARE ROW EXCLUSIVE MODE;
    INSERT INTO "data"."data_tbl_item_award" (id,item_id,award_id,type,count,rate,quality,pre_name_type,payload)
    SELECT id,item_id,award_id,type,count,rate,quality,pre_name_type,payload
    FROM pg_temp.starter_award_input ORDER BY id
    ON CONFLICT (id) DO NOTHING;

    IF EXISTS (
        SELECT 1 FROM (
            SELECT id,item_id,award_id,type,count,rate,quality,pre_name_type,payload
            FROM "data"."data_tbl_item_award"
            WHERE item_id IN (2918,2919,2920,2921,2922,2923)
               OR id IN (SELECT id FROM pg_temp.starter_award_input)
        ) a FULL JOIN pg_temp.starter_award_input r ON a.id=r.id
        WHERE a.id IS NULL OR r.id IS NULL OR
          ROW(a.item_id,a.award_id,a.type,a.count,a.rate,a.quality,a.pre_name_type,a.payload)
          IS DISTINCT FROM ROW(r.item_id,r.award_id,r.type,r.count,r.rate,r.quality,r.pre_name_type,r.payload)
    ) THEN
        RAISE EXCEPTION 'Existing reward data conflicts with approved manifest; transaction rolled back';
    END IF;
    RAISE NOTICE 'Verified 54 guaranteed starter reward rows; player data untouched';
END
$install$;
