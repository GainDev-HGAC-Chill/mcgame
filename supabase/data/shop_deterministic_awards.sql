-- Exact catalog-description rewards only. Do not invent random pools or bonus chances.
DO $install$
DECLARE
    config jsonb := $shop_awards$
[
 [1455,29,24,12,0,"Kim Loại Hiếm","12 kim loại hiếm cấp 4"],
 [1456,29,21,12,0,"Gỗ Tốt","12 gỗ tốt cấp 4"],
 [1457,29,23,12,0,"Pha Lê","12 pha lê cấp 4"],
 [1458,29,26,12,0,"Gấm Vóc","12 gấm vóc cấp 4"],
 [1459,29,28,12,0,"Da Thú","12 da thú cấp 4"],
 [1469,29,7,12,0,"Kim Loại","12 kim loại cấp 4"],
 [1470,29,18,12,0,"Gỗ","12 gỗ cấp 4"],
 [1472,29,22,12,0,"Ngọc","12 ngọc cấp 4"],
 [1473,29,25,12,0,"Vải Thô","12 vải cấp 4"],
 [1474,29,27,12,0,"Lông Thú","12 lông thú cấp 4"],
 [1616,29,274,10,0,"Nội Đơn Bách Luyện","10 viên nội đơn bách"],
 [3254,12,1088,1,1.5,"Camy","chứa 1 camy 1.5"],
 [4840,12,728,1,1.5,"Dê Con II","chứa 1 dê con ii"]
]
$shop_awards$::jsonb;
BEGIN
    CREATE TEMP TABLE shop_award_input ON COMMIT DROP AS
    SELECT (810001000+ordinality)::integer AS id,
           (v->>0)::numeric AS item_id,(v->>1)::numeric AS type,
           (v->>2)::numeric AS award_id,(v->>3)::numeric AS count,
           (v->>4)::numeric AS quality,v->>5 AS expected_name,v->>6 AS description_fragment,
           0::numeric AS pre_name_type,100::numeric AS rate,
           '{"guaranteed":true}'::jsonb || CASE
               WHEN (v->>3)::integer=12 THEN '{"materialColorCode":3}'::jsonb
               ELSE '{}'::jsonb END AS payload
    FROM pg_catalog.jsonb_array_elements(config) WITH ORDINALITY AS input(v,ordinality);

    IF (SELECT count(*) FROM pg_temp.shop_award_input)<>13 OR EXISTS (
        SELECT 1 FROM pg_temp.shop_award_input r
        LEFT JOIN "data"."data_tbl_item_template" b ON b.id=r.item_id
        WHERE b.id IS NULL OR b.use_type<>1 OR b.type NOT IN (508,550)
           OR position(r.description_fragment IN regexp_replace(lower(b.description),'\s+',' ','g'))=0
           OR NOT EXISTS (SELECT 1 FROM "data"."data_tbl_shop_slot" s WHERE s.type=29 AND s.item_id=b.id)
    ) THEN
        RAISE EXCEPTION 'Shop bag description/catalog mismatch';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_temp.shop_award_input r
        LEFT JOIN (
            SELECT 29 AS type,id,name FROM "data"."data_tbl_item_template"
            UNION ALL SELECT 12,id,name FROM "data"."data_tbl_creature"
        ) t ON t.type=r.type AND t.id=r.award_id
        WHERE t.id IS NULL OR t.name<>r.expected_name OR r.count<=0
    ) THEN
        RAISE EXCEPTION 'Reward ID/name mismatch';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_temp.shop_award_input r
        JOIN "data"."data_tbl_item_template" t ON t.id=r.award_id
        WHERE r.payload ? 'materialColorCode' AND (r.type<>29 OR t.kind<>6)
    ) THEN
        RAISE EXCEPTION 'Material grade on nonmaterial reward';
    END IF;

    LOCK TABLE "data"."data_tbl_item_award" IN SHARE ROW EXCLUSIVE MODE;
    INSERT INTO "data"."data_tbl_item_award" (id,item_id,award_id,type,count,quality,pre_name_type,rate,payload)
    SELECT id,item_id,award_id,type,count,quality,pre_name_type,rate,payload
    FROM pg_temp.shop_award_input ORDER BY id ON CONFLICT (id) DO NOTHING;

    IF EXISTS (
        SELECT 1 FROM (
            SELECT * FROM "data"."data_tbl_item_award"
            WHERE item_id IN (SELECT item_id FROM pg_temp.shop_award_input)
               OR id IN (SELECT id FROM pg_temp.shop_award_input)
        ) a FULL JOIN pg_temp.shop_award_input r ON a.id=r.id
        WHERE a.id IS NULL OR r.id IS NULL OR
          ROW(a.item_id,a.award_id,a.type,a.count,a.quality,a.pre_name_type,a.rate,a.payload)
          IS DISTINCT FROM ROW(r.item_id,r.award_id,r.type,r.count,r.quality,r.pre_name_type,r.rate,r.payload)
    ) THEN
        RAISE EXCEPTION 'Existing awards conflict; transaction rolled back';
    END IF;
    RAISE NOTICE 'Verified13 deterministic shop bag awards; no player data modified';
END
$install$;
