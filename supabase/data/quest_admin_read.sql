-- Dedicated service-role-only catalog API. No player writes or runtime grants.
create table if not exists data.quest_admin_audit (
  request_id uuid primary key, actor_id uuid not null, target_kind text not null,
  target_id integer not null, reason text not null, backup_reference text not null,
  request_hash text not null, before_image jsonb not null, after_image jsonb,
  old_revision text not null, new_revision text, created_at timestamptz not null default now()
);
alter table data.quest_admin_audit enable row level security;
revoke all on data.quest_admin_audit from public, anon, authenticated, service_role;
grant select, insert, update on data.quest_admin_audit to service_role;

create or replace function public.quest_admin_wire(p_row jsonb) returns jsonb
language sql immutable security invoker set search_path = '' as $$
 select coalesce(jsonb_object_agg(key, case when jsonb_typeof(value)='number'
   and key not in ('id','start_grant_item','start_grant_qty') then to_jsonb(value::text) else value end),'{}'::jsonb)
 from jsonb_each(p_row)
$$;

create or replace function public.quest_admin_snapshot(p_id integer) returns jsonb
language sql stable security invoker set search_path = '' as $$
 select jsonb_build_object('questId',q.id,'quest',public.quest_admin_wire(to_jsonb(q)),
 'requirements',coalesce((select jsonb_agg(public.quest_admin_wire(to_jsonb(r)) order by r.id) from data.data_tbl_quest_require r where r.qid=q.id),'[]'::jsonb),
 'awards',coalesce((select jsonb_agg(public.quest_admin_wire(to_jsonb(a)) order by a.id) from data.data_tbl_quest_award a where a.qid=q.id),'[]'::jsonb),
 'prerequisites',coalesce((select jsonb_agg(public.quest_admin_wire(to_jsonb(p)) order by p.id) from data.data_tbl_quest_pre p where p.qid=q.id),'[]'::jsonb))
 from data.data_tbl_quest q where q.id=p_id
$$;

create or replace function public.quest_admin_loop_detail(p_id integer) returns jsonb
language sql stable security invoker set search_path = '' as $$
 select jsonb_build_object('loop', public.quest_admin_wire(to_jsonb(l)), 'revision',md5(to_jsonb(l)::text),
 'activeUse',jsonb_build_object('activeLoops',(select count(*) from player.character_quest_loops where loop_id=l.id and active),
 'activeQuests',(select count(*) from player.character_quests cq join data.data_tbl_quest q on q.id=cq.quest_id where cq.status in (0,1) and q.type=7 and q.sub_type='7-' || case when l.id=1 then 10 else l.id end::text)),
 'affectedQuestCount',(select count(*) from data.data_tbl_quest q where q.type=7 and q.sub_type='7-' || case when l.id=1 then 10 else l.id end::text))
 from data.data_tbl_quest_loop l where l.id=p_id
$$;

create or replace function public.quest_admin_detail(p_id integer) returns jsonb
language sql stable security invoker set search_path = '' as $$
 select s || jsonb_build_object('revision',md5(s::text),
 'warnings',coalesce((select jsonb_agg(w.message) from (
 select 'Requirement ' || r.id::text || ': missing/unsupported catalog reference' as message from data.data_tbl_quest_require r where r.qid=p_id and
 not (coalesce(r.kind=1 and r.type=29 and exists(select 1 from data.data_tbl_item_template where id=r.item_id),false)
 or coalesce(r.kind=1 and r.type=19 and exists(select 1 from data.data_tbl_equipt_template where id=r.item_id),false)
 or coalesce(r.kind in (2,3) and r.type=12 and exists(select 1 from data.data_tbl_creature where id=r.item_id),false))
 union all select 'Prerequisite ' || p.id::text || ': missing/unsupported catalog reference' from data.data_tbl_quest_pre p where p.qid=p_id and
 not (coalesce(p.kind=1 and p.type=29 and exists(select 1 from data.data_tbl_item_template where id=p.item_id),false)
 or coalesce(p.kind=2 and p.type in (45,-1) and exists(select 1 from data.data_tbl_quest where id=p.item_id),false))
 union all select 'Award ' || a.id::text || ': missing/unsupported catalog reference' from data.data_tbl_quest_award a where a.qid=p_id and
 not (coalesce(a.kind=1 and a.type=29 and exists(select 1 from data.data_tbl_item_template where id=a.item_id),false)
 or coalesce(a.kind=1 and a.type=19 and exists(select 1 from data.data_tbl_equipt_template where id=a.item_id),false)
 or coalesce(a.kind=2 and a.type=12 and exists(select 1 from data.data_tbl_creature where id=a.item_id),false))
 union all select 'Duplicate requirement target retained' from data.data_tbl_quest_require where qid=p_id group by kind,type,item_id,q having count(*)>1
 ) w),'[]'::jsonb),
 'loopRef',case when s->'quest'->>'type'='7' and s->'quest'->>'sub_type' ~ '^7-[0-9]{1,9}$' then
 public.quest_admin_loop_detail(case when substring(s->'quest'->>'sub_type' from 3)::integer=10 then 1 else substring(s->'quest'->>'sub_type' from 3)::integer end) else null end,
 'activeUse',jsonb_build_object('activeQuests',(select count(*) from player.character_quests where quest_id=p_id and status in (0,1)),
 'completedHistory',(select count(*) from player.quest_history where quest_id=p_id),
 'activeLoops',(select count(*) from player.character_quest_loops where active_quest_id=p_id and active)))
 from public.quest_admin_snapshot(p_id) s where s is not null
$$;

create or replace function public.quest_admin_list(p_search text default '',p_id integer default null,p_type integer default null,p_min_level integer default null,p_max_level integer default null,p_page integer default 1,p_page_size integer default 25) returns jsonb
language plpgsql stable security invoker set search_path = '' as $$
declare v_result jsonb;
begin
 if p_page not between 1 and 100000 or p_page_size not between 1 and 100 or length(p_search)>200 then raise exception using errcode='22023',message='Invalid pagination'; end if;
 with filtered as (select q.* from data.data_tbl_quest q where (p_search='' or q.name ilike '%' || p_search || '%' or q.id::text=p_search)
 and (p_id is null or q.id=p_id) and (p_type is null or q.type=p_type)
 and (p_min_level is null or q.min_level>=p_min_level) and (p_max_level is null or q.max_level<=p_max_level)),
 page_rows as (select * from filtered order by id limit p_page_size offset (p_page-1)*p_page_size)
 select jsonb_build_object('items',coalesce((select jsonb_agg(public.quest_admin_wire(to_jsonb(r)) order by r.id) from page_rows r),'[]'::jsonb),
 'total',(select count(*) from filtered),'page',p_page,'pageSize',p_page_size) into v_result;
 return v_result;
end $$;

create or replace function public.quest_admin_lookups(p_kind text,p_search text default '',p_page integer default 1,p_page_size integer default 25) returns jsonb
language plpgsql stable security invoker set search_path = '' as $$
declare v_table text; v_result jsonb;
begin
 v_table := case p_kind when 'npc' then 'data_tbl_npc' when 'monster' then 'data_tbl_creature' when 'pet' then 'data_tbl_creature'
 when 'item' then 'data_tbl_item_template' when 'equipment' then 'data_tbl_equipt_template' when 'quest' then 'data_tbl_quest'
 when 'loop' then 'data_tbl_quest_loop' when 'class' then 'data_tbl_class' end;
 if v_table is null or p_page not between 1 and 100000 or p_page_size not between 1 and 100 or length(p_search)>200 then raise exception using errcode='22023',message='Invalid lookup'; end if;
 execute format('with f as (select id,name from data.%I where ($1='''' or name ilike ''%%'' || $1 || ''%%'' or id::text=$1)), p as (select * from f order by id limit $2 offset $3) select jsonb_build_object(''items'',coalesce((select jsonb_agg(jsonb_build_object(''id'',id,''name'',name,''table'',$4) order by id) from p),''[]''::jsonb),''total'',(select count(*) from f),''page'',$5,''pageSize'',$2)',v_table)
 into v_result using p_search,p_page_size,(p_page-1)*p_page_size,v_table,p_page;
 return v_result;
end $$;

revoke all on function public.quest_admin_wire(jsonb),public.quest_admin_snapshot(integer),public.quest_admin_loop_detail(integer),public.quest_admin_detail(integer),public.quest_admin_list(text,integer,integer,integer,integer,integer,integer),public.quest_admin_lookups(text,text,integer,integer) from public,anon,authenticated;
grant execute on function public.quest_admin_wire(jsonb),public.quest_admin_snapshot(integer),public.quest_admin_loop_detail(integer),public.quest_admin_detail(integer),public.quest_admin_list(text,integer,integer,integer,integer,integer,integer),public.quest_admin_lookups(text,text,integer,integer) to service_role;
