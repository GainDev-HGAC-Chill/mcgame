-- All catalog changes and audit rows in this test are rolled back. No player writes.
begin;
create temporary table qa_fingerprints(table_name text primary key, fingerprint text);
do $$
declare t record; h text;
begin
 for t in select table_schema,table_name from information_schema.tables where table_type='BASE TABLE' and (table_schema='player' or table_schema='data' and table_name in ('data_tbl_quest','data_tbl_quest_require','data_tbl_quest_award','data_tbl_quest_pre','data_tbl_quest_loop')) loop
  execute format('select md5(coalesce(string_agg(v::text,'''' order by v::text),'''')) from (select to_jsonb(r) v from %I.%I r) s',t.table_schema,t.table_name) into h;
  insert into qa_fingerprints values(t.table_schema || '.' || t.table_name,h);
 end loop;
end $$;
set local role service_role;
do $$
declare before_data jsonb; result jsonb; rev text; actor uuid:='11111111-1111-4111-8111-111111111111'; req uuid:='22222222-2222-4222-8222-222222222222'; cid integer; active_id integer; loop_data jsonb; audit_count integer; item_target integer; inserted_id integer;
begin
 before_data:=public.quest_admin_detail(162); rev:=before_data->>'revision';
 select count(*) into audit_count from data.quest_admin_audit;
 begin perform public.quest_admin_save(162,'00000000000000000000000000000000','{"quest":{"name":"verify"}}','rollback verification',req,actor); raise exception 'Stale revision accepted'; exception when serialization_failure then null; end;
 begin perform public.quest_admin_save(162,rev,'{"quest":{"at":"2"}}','rollback verification',req,actor); raise exception 'Unsupported award mode accepted'; exception when invalid_parameter_value then null; end;
 begin perform public.quest_admin_save(162,rev,'{"quest":{"script_give_up":"x"}}','rollback verification',req,actor); raise exception 'Script edit accepted'; exception when invalid_parameter_value then null; end;
 begin perform public.quest_admin_save(162,rev,'{"quest":{"money_num":"2","money_type":"10"}}','rollback verification',req,actor); raise exception 'Unsupported currency accepted'; exception when invalid_parameter_value then null; end;
 begin perform public.quest_admin_save(162,rev,'{"requirements":{"insert":[{"item_id":"2147483647","kind":"1","num":"1","q":"0","type":"29"}]}}','rollback verification',req,actor); raise exception 'Missing target accepted'; exception when invalid_parameter_value then null; end;
 begin perform public.quest_admin_save(162,rev,'{"requirements":{"insert":[{"item_id":"515","kind":"1","num":"1","q":"-1","type":"29"}]}}','rollback verification',req,actor); raise exception 'Duplicate target accepted'; exception when invalid_parameter_value then null; end;
 begin perform public.quest_admin_save(162,rev,'{"prerequisites":{"insert":[{"item_id":"162","kind":"2","num":"1","type":"45"}]}}','rollback verification',req,actor); raise exception 'Self cycle accepted'; exception when invalid_parameter_value then null; end;
 select id into cid from data.data_tbl_quest_require where qid<>162 limit 1;
 begin perform public.quest_admin_save(162,rev,jsonb_build_object('quest',jsonb_build_object('name','should rollback'),'requirements',jsonb_build_object('update',jsonb_build_array(jsonb_build_object('id',cid,'changes',jsonb_build_object('num','1'))))),'rollback verification',req,actor); raise exception 'Cross-quest row accepted'; exception when invalid_parameter_value then null; end;
 if public.quest_admin_detail(162)<>before_data or (select count(*) from data.quest_admin_audit)<>audit_count then raise exception 'Rejected write/audit changed data'; end if;
 result:=public.quest_admin_save(162,rev,'{"quest":{"name":"Quest Admin rollback-only verification"}}','rollback verification',req,actor);
 if result->'quest'->>'name'<>'Quest Admin rollback-only verification' or result->>'revision'=rev then raise exception 'Valid update failed'; end if;
 if ((result->'quest')-'name')<>((before_data->'quest')-'name') or result->'requirements'<>before_data->'requirements' or result->'awards'<>before_data->'awards' or result->'prerequisites'<>before_data->'prerequisites' then raise exception 'Unchanged content lost'; end if;
 if public.quest_admin_save(162,rev,'{"quest":{"name":"Quest Admin rollback-only verification"}}','rollback verification',req,actor)<>result then raise exception 'Idempotency failed'; end if;
 if not exists(select 1 from data.quest_admin_audit where request_id=req and before_image->'quest'=before_data->'quest' and after_image=result) then raise exception 'Audit before-image failed'; end if;
 perform public.quest_admin_save(162,result->>'revision',jsonb_build_object('quest',jsonb_build_object('name',before_data->'quest'->'name')),'rollback restore name','33333333-3333-4333-8333-333333333333',actor);
 select id into item_target from data.data_tbl_item_template where id<>515 order by id limit 1;
 result:=public.quest_admin_save(162,rev,jsonb_build_object('requirements',jsonb_build_object('insert',jsonb_build_array(jsonb_build_object('item_id',item_target::text,'kind','1','type','29','num','1','q','-1')))),'insert allocation verify','77777777-7777-4777-8777-777777777777',actor);
 select id into inserted_id from data.data_tbl_quest_require where qid=162 and item_id=item_target;
 if inserted_id is null or inserted_id<=9032 then raise exception 'ID allocation failed'; end if;
 result:=public.quest_admin_save(162,result->>'revision',jsonb_build_object('requirements',jsonb_build_object('update',jsonb_build_array(jsonb_build_object('id',inserted_id,'changes',jsonb_build_object('num','2'))))),'child ID update verify','88888888-8888-4888-8888-888888888888',actor);
 if not exists(select 1 from data.data_tbl_quest_require where id=inserted_id and qid=162 and num=2) then raise exception 'Identity-preserving update failed'; end if;
 perform public.quest_admin_save(162,result->>'revision',jsonb_build_object('requirements',jsonb_build_object('deleteIds',jsonb_build_array(inserted_id))),'delete only inserted row','99999999-9999-4999-8999-999999999999',actor);
 select quest_id into active_id from player.character_quests where status in (0,1) limit 1;
 if active_id is not null and public.quest_admin_detail(active_id) is not null then
  begin perform public.quest_admin_save(active_id,public.quest_admin_detail(active_id)->>'revision','{"requirements":{"insert":[{"item_id":"515","kind":"1","num":"1","q":"-1","type":"29"}]}}','active guard verification','44444444-4444-4444-8444-444444444444',actor); raise exception 'Active objective edit accepted'; exception when serialization_failure then null; end;
 end if;
 loop_data:=public.quest_admin_loop_detail(1);
 begin perform public.quest_admin_save_loop(1,loop_data->>'revision',0,true,'{"name":"verify"}','shared verification','55555555-5555-4555-8555-555555555555',actor); raise exception 'Shared impact mismatch accepted'; exception when serialization_failure then null; end;
 result:=public.quest_admin_save_loop(1,loop_data->>'revision',(loop_data->>'affectedQuestCount')::integer,true,'{"name":"Loop rollback verification"}','shared verification','55555555-5555-4555-8555-555555555555',actor);
 if ((result->'loop')-'name')<>((loop_data->'loop')-'name') then raise exception 'Loop legacy fields normalized'; end if;
 perform public.quest_admin_save_loop(1,result->>'revision',(result->>'affectedQuestCount')::integer,true,jsonb_build_object('name',loop_data->'loop'->'name'),'shared restore name','66666666-6666-4666-8666-666666666666',actor);
 raise notice 'PASS stale/enum/script/currency/target/cycle/ownership/rollback/idempotency/audit/active/shared/legacy preservation';
end $$;
reset role;
do $$
declare t record; h text;
begin
 for t in select * from qa_fingerprints loop
  execute format('select md5(coalesce(string_agg(v::text,'''' order by v::text),'''')) from (select to_jsonb(r) v from %s r) s',t.table_name) into h;
  if h<>t.fingerprint then raise exception 'Fingerprint changed: %',t.table_name; end if;
 end loop;
 if exists(select 1 from pg_proc p join pg_namespace n on n.oid=p.pronamespace where n.nspname='public' and p.proname like 'quest_admin_%' and (has_function_privilege('anon',p.oid,'execute') or has_function_privilege('authenticated',p.oid,'execute'))) then raise exception 'Unprivileged function access'; end if;
 if has_table_privilege('anon','data.quest_admin_audit','select') or has_table_privilege('authenticated','data.quest_admin_audit','select') then raise exception 'Audit exposed'; end if;
 raise notice 'PASS all player tables and all five catalog fingerprints unchanged';
end $$;
rollback;
