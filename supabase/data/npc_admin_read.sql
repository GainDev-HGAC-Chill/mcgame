-- Service-role-only NPC, quest-link, and standard-shop catalog API.
create table if not exists data.npc_admin_audit (
  request_id uuid primary key,
  actor_id uuid not null,
  target_id integer not null,
  reason text not null,
  backup_reference text not null,
  request_hash text not null,
  before_image jsonb not null,
  after_image jsonb,
  old_revision text not null,
  new_revision text,
  created_at timestamptz not null default now()
);
alter table data.npc_admin_audit enable row level security;
revoke all on data.npc_admin_audit from public, anon, authenticated, service_role;
grant select, insert, update on data.npc_admin_audit to service_role;

create or replace function public.npc_admin_snapshot(p_id integer) returns jsonb
language sql stable security invoker set search_path = '' as $$
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
$$;

create or replace function public.npc_admin_detail(p_id integer) returns jsonb
language sql stable security invoker set search_path = '' as $$
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
$$;

create or replace function public.npc_admin_list(
 p_search text default '',p_map_id integer default null,p_type integer default null,
 p_has_dialogue boolean default null,p_has_shop boolean default null,p_page integer default 1,p_page_size integer default 25
) returns jsonb
language plpgsql stable security invoker set search_path = '' as $$
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
end $$;

create or replace function public.npc_admin_lookups(p_kind text,p_search text default '',p_page integer default 1,p_page_size integer default 25) returns jsonb
language plpgsql stable security invoker set search_path = '' as $$
declare v_table text; v_result jsonb;
begin
 v_table:=case p_kind when 'shop' then 'data_tbl_shop' when 'item' then 'data_tbl_item_template' when 'equipment' then 'data_tbl_equipt_template' when 'map' then 'data_tbl_map' end;
 if v_table is null or p_page not between 1 and 100000 or p_page_size not between 1 and 100 or length(p_search)>200 then raise exception using errcode='22023',message='Invalid lookup'; end if;
 execute format('with f as (select id,name from data.%I where ($1='''' or name ilike ''%%''||$1||''%%'' or id::text=$1)), p as (select * from f order by id limit $2 offset $3) select jsonb_build_object(''items'',coalesce((select jsonb_agg(jsonb_build_object(''id'',id,''name'',name,''table'',$4) order by id) from p),''[]''::jsonb),''total'',(select count(*) from f),''page'',$5,''pageSize'',$2)',v_table)
 into v_result using p_search,p_page_size,(p_page-1)*p_page_size,v_table,p_page;
 return v_result;
end $$;

revoke all on function public.npc_admin_snapshot(integer),public.npc_admin_detail(integer),public.npc_admin_list(text,integer,integer,boolean,boolean,integer,integer),public.npc_admin_lookups(text,text,integer,integer) from public,anon,authenticated;
grant execute on function public.npc_admin_snapshot(integer),public.npc_admin_detail(integer),public.npc_admin_list(text,integer,integer,boolean,boolean,integer,integer),public.npc_admin_lookups(text,text,integer,integer) to service_role;
