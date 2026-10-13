SELECT q.id, q.name, q.type, q.sub_type, q.info, q.start_npc, q.finish_npc,
       q.min_level, q.max_level, q.req_class, q.rt, q.rn, q.qtest, q.a_script_text,
       (SELECT count(*) FROM data.data_tbl_quest_require r WHERE r.qid = q.id AND (
          (r.type = 29 AND NOT EXISTS (SELECT 1 FROM data.data_tbl_item_template t WHERE t.id = r.item_id)) OR
          (r.type = 19 AND NOT EXISTS (SELECT 1 FROM data.data_tbl_equipt_template t WHERE t.id = r.item_id)) OR
          (r.type = 12 AND NOT EXISTS (SELECT 1 FROM data.data_tbl_creature t WHERE t.id = r.item_id))
       )) AS missing_targets,
       COALESCE((SELECT jsonb_agg(to_jsonb(r) ORDER BY r.id)
                 FROM data.data_tbl_quest_require r WHERE r.qid = q.id), '[]'::jsonb) AS requirements,
       COALESCE((SELECT jsonb_agg(to_jsonb(p) ORDER BY p.id)
                 FROM data.data_tbl_quest_pre p WHERE p.qid = q.id), '[]'::jsonb) AS prerequisites
FROM data.data_tbl_quest q
ORDER BY q.id;
