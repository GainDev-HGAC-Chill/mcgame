create or replace function public.npc_admin_save(
 p_id integer,p_expected_revision text,p_expected_shared_count integer,p_confirm_shared boolean,
 p_patch jsonb,p_reason text,p_request_id uuid,p_actor_id uuid
) returns jsonb
language plpgsql security invoker set search_path = '' as $$
declare before_data jsonb; after_data jsonb; before_npc jsonb; old_hash text; req_hash text; prior data.npc_admin_audit%rowtype;
 npc_patch jsonb; slot_delta jsonb; entry jsonb; changes jsonb; old_row jsonb; merged jsonb; target_shop integer; shared_count integer; row_id integer; next_id integer; ids integer[]:=array[]::integer[]; assignments text;
 allowed_slot text[]:=array['amount','gold','gt','item_id','money','p_num1','p_num2','p_type1','p_type2','point','price_all','quality','r','sale','st','type'];
begin
 if p_actor_id is null or p_request_id is null or p_expected_revision is null or p_expected_revision !~ '^[a-f0-9]{32}$' or p_expected_shared_count is null or p_expected_shared_count<0 or p_reason is null or length(trim(p_reason)) not between 3 and 1000 or p_patch is null or jsonb_typeof(p_patch)<>'object' or p_patch='{}'::jsonb or pg_column_size(p_patch)>131072 then raise exception using errcode='22023',message='Invalid save envelope'; end if;
 if exists(select 1 from jsonb_object_keys(p_patch) k where k not in ('npc','slots')) then raise exception using errcode='22023',message='Unknown patch group'; end if;
  lock table data.data_tbl_npc,data.data_tbl_shop,data.data_tbl_shop_slot in share row exclusive mode;
 req_hash:=md5(jsonb_build_array(p_id,p_expected_revision,p_expected_shared_count,p_confirm_shared,p_patch,p_reason,p_actor_id)::text);
 select * into prior from data.npc_admin_audit where request_id=p_request_id;
 if found then
  if prior.request_hash<>req_hash or prior.actor_id<>p_actor_id or prior.target_id<>p_id then raise exception using errcode='40001',message='Request ID conflict'; end if;
  return prior.after_image;
 end if;
 before_data:=public.npc_admin_snapshot(p_id);
 if before_data is null then raise exception using errcode='P0002',message='NPC not found'; end if;
 old_hash:=md5(before_data::text);
 if old_hash<>p_expected_revision then raise exception using errcode='40001',message='NPC revision conflict'; end if;
 before_npc:=before_data->'npc'; npc_patch:=coalesce(p_patch->'npc','{}'::jsonb);
  if jsonb_typeof(npc_patch)<>'object' or exists(select 1 from jsonb_object_keys(npc_patch) k where k not in ('name','on_service_text','shop_id','pos_map_id','pos_x','pos_y')) then raise exception using errcode='22023',message='Unknown or immutable NPC field'; end if;
 if p_patch ? 'npc' and npc_patch='{}'::jsonb then raise exception using errcode='22023',message='Empty NPC patch'; end if;
 if npc_patch ? 'name' and (jsonb_typeof(npc_patch->'name')<>'string' or length(trim(npc_patch->>'name')) not between 1 and 300) then raise exception using errcode='22023',message='Invalid NPC name'; end if;
 if npc_patch ? 'on_service_text' and npc_patch->'on_service_text'<>'null'::jsonb and (jsonb_typeof(npc_patch->'on_service_text')<>'string' or length(npc_patch->>'on_service_text')>16000) then raise exception using errcode='22023',message='Invalid NPC dialogue'; end if;
  if npc_patch ? 'shop_id' and npc_patch->'shop_id'<>'null'::jsonb and (jsonb_typeof(npc_patch->'shop_id')<>'number' or (npc_patch->>'shop_id')::numeric<>trunc((npc_patch->>'shop_id')::numeric) or (npc_patch->>'shop_id')::numeric<0) then raise exception using errcode='22023',message='Invalid shop ID'; end if;
  if npc_patch ? 'pos_map_id' and (jsonb_typeof(npc_patch->'pos_map_id')<>'number' or (npc_patch->>'pos_map_id')::numeric<>trunc((npc_patch->>'pos_map_id')::numeric) or (npc_patch->>'pos_map_id')::numeric not between 1 and 2147483647) then raise exception using errcode='22023',message='Invalid NPC map ID'; end if;
  if npc_patch ? 'pos_map_id' and not exists(select 1 from data.data_tbl_map where id=(npc_patch->>'pos_map_id')::integer) then raise exception using errcode='22023',message='Unknown NPC map'; end if;
  if npc_patch ? 'pos_x' and (jsonb_typeof(npc_patch->'pos_x')<>'number' or (npc_patch->>'pos_x')::numeric<>trunc((npc_patch->>'pos_x')::numeric) or (npc_patch->>'pos_x')::numeric not between -2147483648 and 2147483647) then raise exception using errcode='22023',message='Invalid NPC X coordinate'; end if;
  if npc_patch ? 'pos_y' and (jsonb_typeof(npc_patch->'pos_y')<>'number' or (npc_patch->>'pos_y')::numeric<>trunc((npc_patch->>'pos_y')::numeric) or (npc_patch->>'pos_y')::numeric not between -2147483648 and 2147483647) then raise exception using errcode='22023',message='Invalid NPC Y coordinate'; end if;
 merged:=before_npc||npc_patch; target_shop:=coalesce((merged->>'shop_id')::numeric,0)::integer;
 if target_shop>0 and not exists(select 1 from data.data_tbl_shop where id=target_shop) then raise exception using errcode='22023',message='Unknown standard shop'; end if;
 slot_delta:=coalesce(p_patch->'slots','{}'::jsonb);
 if jsonb_typeof(slot_delta)<>'object' or exists(select 1 from jsonb_object_keys(slot_delta) k where k not in ('insert','update','hideIds')) then raise exception using errcode='22023',message='Invalid slot delta'; end if;
 if p_patch ? 'slots' and (slot_delta='{}'::jsonb or not exists(select 1 from jsonb_each(slot_delta) d where case when jsonb_typeof(d.value)='array' then jsonb_array_length(d.value)>0 else false end)) then raise exception using errcode='22023',message='Empty slot patch'; end if;
 foreach changes in array array[coalesce(slot_delta->'insert','[]'::jsonb),coalesce(slot_delta->'update','[]'::jsonb),coalesce(slot_delta->'hideIds','[]'::jsonb)] loop
  if jsonb_typeof(changes)<>'array' then raise exception using errcode='22023',message='Invalid slot array'; end if;
  if jsonb_array_length(changes)>100 then raise exception using errcode='22023',message='Invalid slot array'; end if;
 end loop;
 if slot_delta<>'{}'::jsonb then
  if target_shop<1 then raise exception using errcode='22023',message='Shop slots require an assigned shop'; end if;
  select count(*) into shared_count from data.data_tbl_npc where shop_id=target_shop;
  if shared_count<>p_expected_shared_count or shared_count>1 and p_confirm_shared is distinct from true then raise exception using errcode='40001',message='Shared shop impact conflict'; end if;
 end if;
 insert into data.npc_admin_audit(request_id,actor_id,target_id,reason,backup_reference,request_hash,before_image,old_revision)
 values(p_request_id,p_actor_id,p_id,p_reason,'.local/backups/npc-admin/20261019-before-schema.sql',req_hash,before_data,old_hash);
 if npc_patch<>'{}'::jsonb then
  select string_agg(format('%I=r.%I',key,key),',') into assignments from jsonb_object_keys(npc_patch) key;
  execute format('update data.data_tbl_npc n set %s from jsonb_populate_record(null::data.data_tbl_npc,$1) r where n.id=$2',assignments) using merged,p_id;
 end if;
 for entry in select value from jsonb_array_elements(coalesce(slot_delta->'update','[]'::jsonb)) loop
  if jsonb_typeof(entry)<>'object' or not (entry ?& array['id','changes']) or exists(select 1 from jsonb_object_keys(entry) k where k not in ('id','changes')) or jsonb_typeof(entry->'id')<>'number' or jsonb_typeof(entry->'changes')<>'object' or entry->'changes'='{}'::jsonb or exists(select 1 from jsonb_object_keys(entry->'changes') k where not k=any(allowed_slot)) then raise exception using errcode='22023',message='Invalid slot update'; end if;
  if (entry->>'id')::numeric<>trunc((entry->>'id')::numeric) or (entry->>'id')::numeric not between 1 and 2147483647 then raise exception using errcode='22023',message='Invalid slot ID'; end if;
  row_id:=(entry->>'id')::integer; if row_id=any(ids) then raise exception using errcode='22023',message='Duplicate slot ID'; end if; ids:=array_append(ids,row_id);
  select to_jsonb(s) into old_row from data.data_tbl_shop_slot s where id=row_id and sid=target_shop;
  if old_row is null then raise exception using errcode='22023',message='Slot does not belong to shop'; end if;
  merged:=old_row||(entry->'changes'); perform public.npc_admin_validate_slot(merged);
  if exists(select 1 from data.data_tbl_shop_slot s where s.sid=target_shop and s.position=(merged->>'position')::numeric and s.id<>row_id and s.st<>5 and (merged->>'st')::numeric<>5) then raise exception using errcode='22023',message='Duplicate active shop position'; end if;
  select string_agg(format('%I=r.%I',key,key),',') into assignments from jsonb_object_keys(entry->'changes') key;
  execute format('update data.data_tbl_shop_slot s set %s from jsonb_populate_record(null::data.data_tbl_shop_slot,$1) r where s.id=$2 and s.sid=$3',assignments) using merged,row_id,target_shop;
 end loop;
 for entry in select value from jsonb_array_elements(coalesce(slot_delta->'hideIds','[]'::jsonb)) loop
  if jsonb_typeof(entry)<>'number' or (entry::text)::numeric<>trunc((entry::text)::numeric) or (entry::text)::numeric not between 1 and 2147483647 then raise exception using errcode='22023',message='Invalid hide ID'; end if;
  row_id:=(entry::text)::integer; if row_id=any(ids) then raise exception using errcode='22023',message='Duplicate slot ID'; end if; ids:=array_append(ids,row_id);
  update data.data_tbl_shop_slot set st=5 where id=row_id and sid=target_shop and st in (0,5);
  if not found then raise exception using errcode='22023',message='Slot does not belong to shop'; end if;
 end loop;
 for entry in select value from jsonb_array_elements(coalesce(slot_delta->'insert','[]'::jsonb)) loop
  if jsonb_typeof(entry)<>'object' or exists(select 1 from jsonb_object_keys(entry) k where k not in ('amount','gold','gt','item_id','money','p_num1','p_num2','p_type1','p_type2','point','position','price_all','quality','r','sale','st','type')) then raise exception using errcode='22023',message='Invalid slot insert'; end if;
  select coalesce(max(id),0)+1 into next_id from data.data_tbl_shop_slot;
  merged:=jsonb_build_object('id',next_id,'sid',target_shop)||entry;
  perform public.npc_admin_validate_slot(merged);
  if (merged->>'position')::integer<1 then raise exception using errcode='22023',message='New slot position must be positive'; end if;
  if exists(select 1 from data.data_tbl_shop_slot s where s.sid=target_shop and s.position=(merged->>'position')::numeric and s.st<>5 and (merged->>'st')::numeric<>5) then raise exception using errcode='22023',message='Duplicate active shop position'; end if;
  insert into data.data_tbl_shop_slot select r.* from jsonb_populate_record(null::data.data_tbl_shop_slot,merged) r;
 end loop;
 after_data:=public.npc_admin_detail(p_id);
 if after_data->>'revision'=old_hash then raise exception using errcode='22023',message='Patch did not change catalog data'; end if;
 update data.npc_admin_audit set after_image=after_data,new_revision=after_data->>'revision' where request_id=p_request_id;
 return after_data;
end $$;

revoke all on function public.npc_admin_save(integer,text,integer,boolean,jsonb,text,uuid,uuid) from public,anon,authenticated;
grant execute on function public.npc_admin_save(integer,text,integer,boolean,jsonb,text,uuid,uuid) to service_role;
revoke insert,update,delete,truncate,references,trigger on data.data_tbl_npc,data.data_tbl_shop,data.data_tbl_shop_slot from anon,authenticated;
notify pgrst, 'reload schema';
