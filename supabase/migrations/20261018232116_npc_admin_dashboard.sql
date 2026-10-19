set check_function_bodies = off;

CREATE OR REPLACE FUNCTION public.npc_admin_detail(p_id integer)
 RETURNS jsonb
 LANGUAGE sql
 STABLE
 SET search_path TO ''
AS $function$
 select s || jsonb_build_object(
  'npcId',p_id,
  'revision',md5(s::text),
  'warnings',coalesce((select jsonb_agg(message order by message) from (
   select 'NPC references a missing standard shop' message where coalesce((s->'npc'->>'shop_id')::numeric,0)>0 and s->'shop' is null
   union all select 'Legacy qid contains missing quest IDs' where jsonb_array_length(s->'legacyQid'->'missingIds')>0
   union all select 'Shop is shared by multiple NPCs' where jsonb_array_length(coalesce(s->'shop'->'referencingNpcs','[]'::jsonb))>1
   union all select 'Legacy duplicate active shop positions are preserved' where exists(
    select 1 from jsonb_to_recordset(coalesce(s->'shop'->'slots','[]'::jsonb)) x(position numeric,st numeric)
    where coalesce(x.st,0)<>5 group by x.position having count(*)>1
    )
   ) w),'[]'::jsonb)
 ) from public.npc_admin_snapshot(p_id) s where s is not null
$function$
;

CREATE OR REPLACE FUNCTION public.npc_admin_list(p_search text DEFAULT ''::text, p_map_id integer DEFAULT NULL::integer, p_type integer DEFAULT NULL::integer, p_has_dialogue boolean DEFAULT NULL::boolean, p_has_shop boolean DEFAULT NULL::boolean, p_page integer DEFAULT 1, p_page_size integer DEFAULT 25)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE
 SET search_path TO ''
AS $function$
declare v_result jsonb;
begin
 if p_page not between 1 and 100000 or p_page_size not between 1 and 100 or length(p_search)>200 then raise exception using errcode='22023',message='Invalid pagination'; end if;
 with filtered as (
  select n.*,
   (select count(*) from data.data_tbl_quest q where q.start_npc=n.id) offered_count,
   (select count(*) from data.data_tbl_quest q where q.finish_npc=n.id) turn_in_count,
   (select count(*) from data.data_tbl_quest_loop l where l.nid=n.id) loop_count
  from data.data_tbl_npc n where
   (p_search='' or n.name ilike '%'||p_search||'%' or n.id::text=p_search)
   and (p_map_id is null or n.pos_map_id=p_map_id)
   and (p_type is null or n.type=p_type)
   and (p_has_dialogue is null or (nullif(trim(coalesce(n.on_service_text,'')),'') is not null)=p_has_dialogue)
   and (p_has_shop is null or (coalesce(n.shop_id,0)>0)=p_has_shop)
 ), page_rows as (select * from filtered order by id limit p_page_size offset (p_page-1)*p_page_size)
 select jsonb_build_object(
  'items',coalesce((select jsonb_agg(to_jsonb(r) order by r.id) from page_rows r),'[]'::jsonb),
  'total',(select count(*) from filtered),'page',p_page,'pageSize',p_page_size
 ) into v_result;
 return v_result;
end $function$
;

CREATE OR REPLACE FUNCTION public.npc_admin_lookups(p_kind text, p_search text DEFAULT ''::text, p_page integer DEFAULT 1, p_page_size integer DEFAULT 25)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE
 SET search_path TO ''
AS $function$
declare v_table text; v_result jsonb;
begin
 v_table:=case p_kind when 'shop' then 'data_tbl_shop' when 'item' then 'data_tbl_item_template' when 'equipment' then 'data_tbl_equipt_template' when 'map' then 'data_tbl_map' end;
 if v_table is null or p_page not between 1 and 100000 or p_page_size not between 1 and 100 or length(p_search)>200 then raise exception using errcode='22023',message='Invalid lookup'; end if;
 execute format('with f as (select id,name from data.%I where ($1='''' or name ilike ''%%''||$1||''%%'' or id::text=$1)), p as (select * from f order by id limit $2 offset $3) select jsonb_build_object(''items'',coalesce((select jsonb_agg(jsonb_build_object(''id'',id,''name'',name,''table'',$4) order by id) from p),''[]''::jsonb),''total'',(select count(*) from f),''page'',$5,''pageSize'',$2)',v_table)
 into v_result using p_search,p_page_size,(p_page-1)*p_page_size,v_table,p_page;
 return v_result;
end $function$
;

CREATE OR REPLACE FUNCTION public.npc_admin_save(p_id integer, p_expected_revision text, p_expected_shared_count integer, p_confirm_shared boolean, p_patch jsonb, p_reason text, p_request_id uuid, p_actor_id uuid)
 RETURNS jsonb
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
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
 if jsonb_typeof(npc_patch)<>'object' or exists(select 1 from jsonb_object_keys(npc_patch) k where k not in ('name','on_service_text','shop_id')) then raise exception using errcode='22023',message='Unknown or immutable NPC field'; end if;
 if p_patch ? 'npc' and npc_patch='{}'::jsonb then raise exception using errcode='22023',message='Empty NPC patch'; end if;
 if npc_patch ? 'name' and (jsonb_typeof(npc_patch->'name')<>'string' or length(trim(npc_patch->>'name')) not between 1 and 300) then raise exception using errcode='22023',message='Invalid NPC name'; end if;
 if npc_patch ? 'on_service_text' and npc_patch->'on_service_text'<>'null'::jsonb and (jsonb_typeof(npc_patch->'on_service_text')<>'string' or length(npc_patch->>'on_service_text')>16000) then raise exception using errcode='22023',message='Invalid NPC dialogue'; end if;
 if npc_patch ? 'shop_id' and npc_patch->'shop_id'<>'null'::jsonb and (jsonb_typeof(npc_patch->'shop_id')<>'number' or (npc_patch->>'shop_id')::numeric<>trunc((npc_patch->>'shop_id')::numeric) or (npc_patch->>'shop_id')::numeric<0) then raise exception using errcode='22023',message='Invalid shop ID'; end if;
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
end $function$
;

CREATE OR REPLACE FUNCTION public.npc_admin_snapshot(p_id integer)
 RETURNS jsonb
 LANGUAGE sql
 STABLE
 SET search_path TO ''
AS $function$
 select jsonb_build_object(
  'npc',to_jsonb(n),
  'offeredQuests',coalesce((select jsonb_agg(to_jsonb(q) order by q.id) from data.data_tbl_quest q where q.start_npc=n.id),'[]'::jsonb),
  'turnInQuests',coalesce((select jsonb_agg(to_jsonb(q) order by q.id) from data.data_tbl_quest q where q.finish_npc=n.id),'[]'::jsonb),
  'loops',coalesce((select jsonb_agg(to_jsonb(l) order by l.id) from data.data_tbl_quest_loop l where l.nid=n.id),'[]'::jsonb),
  'legacyQid',jsonb_build_object(
   'raw',n.qid,
   'ids',coalesce((select jsonb_agg(v.id order by v.id) from (
    select distinct part::integer id from regexp_split_to_table(coalesce(n.qid,''),'[|]') part where part ~ '^[1-9][0-9]{0,9}$'
   ) v),'[]'::jsonb),
   'missingIds',coalesce((select jsonb_agg(v.id order by v.id) from (
    select distinct part::integer id from regexp_split_to_table(coalesce(n.qid,''),'[|]') part
    where part ~ '^[1-9][0-9]{0,9}$' and not exists(select 1 from data.data_tbl_quest q where q.id=part::integer)
   ) v),'[]'::jsonb)
  ),
  'shop',case when coalesce(n.shop_id,0)>0 then (
   select jsonb_build_object(
    'container',to_jsonb(s),
     'slots',coalesce((select jsonb_agg(to_jsonb(ss) order by ss.position,ss.id) from data.data_tbl_shop_slot ss where ss.sid=s.id),'[]'::jsonb),
    'referencingNpcs',coalesce((select jsonb_agg(jsonb_build_object('id',rn.id,'name',rn.name) order by rn.id) from data.data_tbl_npc rn where rn.shop_id=s.id),'[]'::jsonb)
   ) from data.data_tbl_shop s where s.id=n.shop_id
  ) else null end
 ) from data.data_tbl_npc n where n.id=p_id
$function$
;

CREATE OR REPLACE FUNCTION public.npc_admin_validate_slot(p_row jsonb)
 RETURNS void
 LANGUAGE plpgsql
 STABLE
 SET search_path TO ''
AS $function$
declare k text; n numeric; integer_fields text[]:=array['id','amount','gold','gt','item_id','money','p_num1','p_num2','p_type1','p_type2','point','position','quality','r','sale','sid','st','type']; supported integer[]:=array[1,2,3,4,7,10,11,12,13,14,15,16,17,18,19,20,21,22,25,26,29,30,31,49,58,60,61,62,64,65,69,70,71];
begin
 if p_row is null or jsonb_typeof(p_row)<>'object' then raise exception using errcode='22023',message='Invalid shop slot'; end if;
 foreach k in array integer_fields loop
  if not p_row ? k or jsonb_typeof(p_row->k)<>'number' then raise exception using errcode='22023',message='Missing or non-numeric shop slot field: '||k; end if;
  n:=(p_row->>k)::numeric;
  if n<>trunc(n) or n>2147483647 or n<(case when k in ('amount','quality') then -1 else 0 end) then raise exception using errcode='22023',message='Invalid shop slot integer: '||k; end if;
 end loop;
 if not p_row ? 'price_all' or p_row->'price_all'<>'null'::jsonb and (jsonb_typeof(p_row->'price_all')<>'number' or (p_row->>'price_all')::numeric<>trunc((p_row->>'price_all')::numeric) or (p_row->>'price_all')::numeric not between 0 and 2147483647) then raise exception using errcode='22023',message='Invalid shop slot integer: price_all'; end if;
 if (p_row->>'id')::integer<1 or (p_row->>'sid')::integer<1 or (p_row->>'position')::integer<0 then raise exception using errcode='22023',message='Invalid slot identity'; end if;
 if (p_row->>'gt')::integer not in (0,1) or (p_row->>'st')::integer not in (0,5) or (p_row->>'type')::integer not in (19,29) then raise exception using errcode='22023',message='Unsupported slot mode'; end if;
 if (p_row->>'type')::integer=29 and not exists(select 1 from data.data_tbl_item_template where id=(p_row->>'item_id')::integer) then raise exception using errcode='22023',message='Unknown item template'; end if;
 if (p_row->>'type')::integer=19 and not exists(select 1 from data.data_tbl_equipt_template where id=(p_row->>'item_id')::integer) then raise exception using errcode='22023',message='Unknown equipment template'; end if;
 if (((p_row->>'p_type1')::integer=0)<>((p_row->>'p_num1')::integer=0)) or (((p_row->>'p_type2')::integer=0)<>((p_row->>'p_num2')::integer=0)) then raise exception using errcode='22023',message='Incomplete currency pair'; end if;
 if ((p_row->>'p_type1')::integer>0 and not ((p_row->>'p_type1')::integer=any(supported))) or ((p_row->>'p_type2')::integer>0 and not ((p_row->>'p_type2')::integer=any(supported))) then raise exception using errcode='22023',message='Unsupported currency'; end if;
 if (p_row->>'point')::integer<>0 then raise exception using errcode='22023',message='Point-only NPC shop pricing is unsupported'; end if;
 if (p_row->>'st')::integer=0 and (p_row->>'p_num1')::integer=0 and (p_row->>'gold')::integer=0 and (p_row->>'money')::integer=0 then raise exception using errcode='22023',message='Active slot needs a supported positive price'; end if;
end $function$
;


  create table "data"."npc_admin_audit" (
    "request_id" uuid not null,
    "actor_id" uuid not null,
    "target_id" integer not null,
    "reason" text not null,
    "backup_reference" text not null,
    "request_hash" text not null,
    "before_image" jsonb not null,
    "after_image" jsonb,
    "old_revision" text not null,
    "new_revision" text,
    "created_at" timestamp with time zone not null default now()
      );


alter table "data"."npc_admin_audit" enable row level security;

CREATE UNIQUE INDEX npc_admin_audit_pkey ON data.npc_admin_audit USING btree (request_id);

alter table "data"."npc_admin_audit" add constraint "npc_admin_audit_pkey" PRIMARY KEY using index "npc_admin_audit_pkey";

grant insert on table "data"."npc_admin_audit" to "service_role";

grant select on table "data"."npc_admin_audit" to "service_role";

grant update on table "data"."npc_admin_audit" to "service_role";

revoke all on function public.npc_admin_snapshot(integer),public.npc_admin_detail(integer),public.npc_admin_list(text,integer,integer,boolean,boolean,integer,integer),public.npc_admin_lookups(text,text,integer,integer),public.npc_admin_validate_slot(jsonb),public.npc_admin_save(integer,text,integer,boolean,jsonb,text,uuid,uuid) from public,anon,authenticated;

grant execute on function public.npc_admin_snapshot(integer),public.npc_admin_detail(integer),public.npc_admin_list(text,integer,integer,boolean,boolean,integer,integer),public.npc_admin_lookups(text,text,integer,integer),public.npc_admin_validate_slot(jsonb),public.npc_admin_save(integer,text,integer,boolean,jsonb,text,uuid,uuid) to service_role;

revoke insert,update,delete,truncate,references,trigger on data.data_tbl_npc,data.data_tbl_shop,data.data_tbl_shop_slot from anon,authenticated;

notify pgrst, 'reload schema';
