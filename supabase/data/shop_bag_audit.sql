SELECT t.id,t.name,t.type,t.use_type,t.description,t.info,t.i1,t.i2,t.req_level,
       (SELECT jsonb_agg(jsonb_build_object('shop',s.sid,'state',s.st,'slot',s.id) ORDER BY s.sid,s.id)
        FROM "data"."data_tbl_shop_slot" s WHERE s.type=29 AND s.item_id=t.id) AS listings,
       (SELECT jsonb_agg(jsonb_build_object('id',a.id,'type',a.type,'award',a.award_id,'count',a.count,
                                          'quality',a.quality,'rate',a.rate,'payload',a.payload) ORDER BY a.id)
        FROM "data"."data_tbl_item_award" a WHERE a.item_id=t.id) AS awards
FROM "data"."data_tbl_item_template" t
WHERE t.use_type=1 AND (t.type IN (508,509,550) OR t.name ~* '(túi|hộp|gói|rương|package|box|bag)')
  AND EXISTS (SELECT 1 FROM "data"."data_tbl_shop_slot" s WHERE s.type=29 AND s.item_id=t.id)
ORDER BY t.id;
