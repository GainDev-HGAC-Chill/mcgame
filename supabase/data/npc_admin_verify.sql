-- All catalog and audit mutations in this verification are rolled back.
begin;
create temporary table npc_admin_before_npc as table data.data_tbl_npc;
create temporary table npc_admin_before_shop as table data.data_tbl_shop;
create temporary table npc_admin_before_slot as table data.data_tbl_shop_slot;
savepoint npc_catalog_test;
set local role service_role;
do $$
declare npc_id integer; shop_id integer; slot_id integer; other_slot_id integer; item_template_id integer; before_data jsonb; result jsonb; rev text; shared_count integer; actor uuid:='11111111-1111-4111-8111-111111111111'; req uuid:='22222222-2222-4222-8222-222222222222'; audit_count integer;
begin
 select n.id,n.shop_id::integer into npc_id,shop_id from data.data_tbl_npc n where coalesce(n.shop_id,0)>0 and exists(select 1 from data.data_tbl_shop where id=n.shop_id) and exists(select 1 from data.data_tbl_shop_slot s where s.sid=n.shop_id and s.st=0) order by n.id limit 1;
 if npc_id is null then raise exception 'No fixture NPC with shop'; end if;
 before_data:=public.npc_admin_detail(npc_id); rev:=before_data->>'revision'; shared_count:=jsonb_array_length(before_data->'shop'->'referencingNpcs');
 select count(*) into audit_count from data.npc_admin_audit;
 begin perform public.npc_admin_save(npc_id,'00000000000000000000000000000000',shared_count,true,'{"npc":{"name":"reject"}}','rollback verification',req,actor); raise exception 'Stale revision accepted'; exception when serialization_failure then null; end;
 begin perform public.npc_admin_save(npc_id,rev,shared_count,true,'{"npc":{}}','rollback verification',req,actor); raise exception 'Empty patch accepted'; exception when invalid_parameter_value then null; end;
 begin perform public.npc_admin_save(npc_id,rev,shared_count,true,jsonb_build_object('npc',jsonb_build_object('on_service_text',before_data->'npc'->'on_service_text')),'rollback verification',req,actor); raise exception 'No-op patch accepted'; exception when invalid_parameter_value then null; end;
 select id into slot_id from data.data_tbl_shop_slot where sid=shop_id and st=0 order by id limit 1;
 begin perform public.npc_admin_save(npc_id,rev,shared_count+1,true,jsonb_build_object('slots',jsonb_build_object('hideIds',jsonb_build_array(slot_id))),'rollback verification',req,actor); raise exception 'Shared impact mismatch accepted'; exception when serialization_failure then null; end;
 begin perform public.npc_admin_save(npc_id,rev,shared_count,true,'{"slots":{"hideIds":[1.5]}}','rollback verification',req,actor); raise exception 'Fractional slot ID accepted'; exception when invalid_parameter_value then null; end;
 select id into other_slot_id from data.data_tbl_shop_slot where sid<>shop_id order by id limit 1;
 begin perform public.npc_admin_save(npc_id,rev,shared_count,true,jsonb_build_object('slots',jsonb_build_object('hideIds',jsonb_build_array(other_slot_id))),'rollback verification',req,actor); raise exception 'Cross-shop slot accepted'; exception when invalid_parameter_value then null; end;
 result:=public.npc_admin_save(npc_id,rev,shared_count,true,'{"npc":{"on_service_text":"NPC Admin rollback verification"}}','rollback verification',req,actor);
 if result->'npc'->>'on_service_text'<>'NPC Admin rollback verification' or result->>'revision'=rev then raise exception 'Valid NPC update failed'; end if;
 if public.npc_admin_save(npc_id,rev,shared_count,true,'{"npc":{"on_service_text":"NPC Admin rollback verification"}}','rollback verification',req,actor)<>result then raise exception 'Idempotency failed'; end if;
 result:=public.npc_admin_save(npc_id,result->>'revision',shared_count,true,jsonb_build_object('slots',jsonb_build_object('hideIds',jsonb_build_array(slot_id))),'rollback verification','44444444-4444-4444-8444-444444444444',actor);
 if (select st from data.data_tbl_shop_slot where id=slot_id)<>5 then raise exception 'Soft hide failed'; end if;
 select id into item_template_id from data.data_tbl_item_template order by id limit 1;
 result:=public.npc_admin_save(npc_id,result->>'revision',shared_count,true,jsonb_build_object('slots',jsonb_build_object('insert',jsonb_build_array(jsonb_build_object('amount',-1,'gold',0,'gt',0,'item_id',item_template_id,'money',1,'p_num1',0,'p_num2',0,'p_type1',0,'p_type2',0,'point',0,'position',2147483000,'price_all',0,'quality',0,'r',0,'sale',0,'st',0,'type',29)))),'rollback verification','33333333-3333-4333-8333-333333333333',actor);
 if not exists(select 1 from data.data_tbl_shop_slot s where s.sid=shop_id and s.position=2147483000 and s.item_id=item_template_id) then raise exception 'Slot insertion failed'; end if;
 if (select count(*) from data.npc_admin_audit)<>audit_count+3 then raise exception 'Audit count mismatch'; end if;
 raise notice 'PASS revision/idempotency/no-op/ownership/shared/soft-hide/audit/insert';
end $$;
reset role;
rollback to savepoint npc_catalog_test;
do $$
begin
 if exists((select * from data.data_tbl_npc except select * from npc_admin_before_npc) union all (select * from npc_admin_before_npc except select * from data.data_tbl_npc)) then raise exception 'NPC catalog changed'; end if;
 if exists((select * from data.data_tbl_shop except select * from npc_admin_before_shop) union all (select * from npc_admin_before_shop except select * from data.data_tbl_shop)) then raise exception 'Shop catalog changed'; end if;
 if exists((select * from data.data_tbl_shop_slot except select * from npc_admin_before_slot) union all (select * from npc_admin_before_slot except select * from data.data_tbl_shop_slot)) then raise exception 'Shop-slot catalog changed'; end if;
 if exists(select 1 from pg_proc p join pg_namespace n on n.oid=p.pronamespace where n.nspname='public' and p.proname like 'npc_admin_%' and (has_function_privilege('anon',p.oid,'execute') or has_function_privilege('authenticated',p.oid,'execute'))) then raise exception 'Unprivileged function access'; end if;
 raise notice 'PASS catalog fingerprints and privileges';
end $$;
rollback;
