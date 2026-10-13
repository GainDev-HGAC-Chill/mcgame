SET local check_function_bodies = off;

CREATE TABLE "data"."quest_admin_audit" (
  "request_id"       uuid                     NOT NULL,
  "actor_id"         uuid                     NOT NULL,
  "target_kind"      text                     NOT NULL,
  "target_id"        integer                  NOT NULL,
  "reason"           text                     NOT NULL,
  "backup_reference" text                     NOT NULL,
  "request_hash"     text                     NOT NULL,
  "before_image"     jsonb                    NOT NULL,
  "after_image"      jsonb,
  "old_revision"     text                     NOT NULL,
  "new_revision"     text,
  "created_at"       timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT "quest_admin_audit_pkey" PRIMARY KEY (request_id)
);

ALTER TABLE "data"."quest_admin_audit"
  ENABLE ROW LEVEL SECURITY;

CREATE OR REPLACE FUNCTION public.quest_admin_apply_fields (
  p_table   text,
  p_id      integer,
  p_before  jsonb,
  p_changes jsonb,
  p_group   text
)
  RETURNS void
  LANGUAGE plpgsql
  SET search_path TO ''
  AS $function$
declare assignments text; actual jsonb; allowed text[];
begin
 if p_table not in ('data_tbl_quest','data_tbl_quest_require','data_tbl_quest_award','data_tbl_quest_pre','data_tbl_quest_loop') then raise exception using errcode='22023',message='Invalid internal table'; end if;
 allowed:=case p_group when 'quest' then array['name','info','start_text','complete_text','type','sub_type','at','award_exp_re','color','start_npc','finish_npc','gender','is_rebirth','lm','max_level','min_level','money_type','money_num','pre_quest_type','req_class','rt','rn','start_grant_item','start_grant_qty'] when 'requirements' then array['item_id','kind','num','q','type'] when 'awards' then array['item_id','kind','num','q','type','b'] when 'prerequisites' then array['item_id','kind','num','type'] when 'loop' then array['name','info','nid','num','min_level','max_level','refresh','team'] end;
 if allowed is null or jsonb_typeof(p_changes) is distinct from 'object' or exists(select 1 from jsonb_object_keys(p_changes) k where not k=any(allowed)) then raise exception using errcode='22023',message='Unknown or immutable patch field'; end if;
 select coalesce(jsonb_object_agg(key,value),'{}'::jsonb) into actual from jsonb_each(p_changes) where p_before->key is distinct from value;
 if actual='{}'::jsonb then return; end if;
 perform public.quest_admin_validate(p_group,p_before,actual);
 select string_agg(format('%I = r.%I',key,key),',') into assignments from jsonb_object_keys(actual) key;
 execute format('update data.%I as t set %s from jsonb_populate_record(null::data.%I,$1) r where t.id=$2',p_table,assignments,p_table) using p_before || actual,p_id;
end $function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_apply_fields"(text, integer, jsonb, jsonb, text) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_detail (
  p_id integer
)
  RETURNS jsonb
  LANGUAGE sql
  STABLE
  SET search_path TO ''
  AS $function$
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
$function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_detail"(integer) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_list (
  p_search    text    DEFAULT ''::text,
  p_id        integer DEFAULT NULL::integer,
  p_type      integer DEFAULT NULL::integer,
  p_min_level integer DEFAULT NULL::integer,
  p_max_level integer DEFAULT NULL::integer,
  p_page      integer DEFAULT 1,
  p_page_size integer DEFAULT 25
)
  RETURNS jsonb
  LANGUAGE plpgsql
  STABLE
  SET search_path TO ''
  AS $function$
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
end $function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_list"(text, integer, integer, integer, integer, integer, integer) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_lookups (
  p_kind      text,
  p_search    text    DEFAULT ''::text,
  p_page      integer DEFAULT 1,
  p_page_size integer DEFAULT 25
)
  RETURNS jsonb
  LANGUAGE plpgsql
  STABLE
  SET search_path TO ''
  AS $function$
declare v_table text; v_result jsonb;
begin
 v_table := case p_kind when 'npc' then 'data_tbl_npc' when 'monster' then 'data_tbl_creature' when 'pet' then 'data_tbl_creature'
 when 'item' then 'data_tbl_item_template' when 'equipment' then 'data_tbl_equipt_template' when 'quest' then 'data_tbl_quest'
 when 'loop' then 'data_tbl_quest_loop' when 'class' then 'data_tbl_class' end;
 if v_table is null or p_page not between 1 and 100000 or p_page_size not between 1 and 100 or length(p_search)>200 then raise exception using errcode='22023',message='Invalid lookup'; end if;
 execute format('with f as (select id,name from data.%I where ($1='''' or name ilike ''%%'' || $1 || ''%%'' or id::text=$1)), p as (select * from f order by id limit $2 offset $3) select jsonb_build_object(''items'',coalesce((select jsonb_agg(jsonb_build_object(''id'',id,''name'',name,''table'',$4) order by id) from p),''[]''::jsonb),''total'',(select count(*) from f),''page'',$5,''pageSize'',$2)',v_table)
 into v_result using p_search,p_page_size,(p_page-1)*p_page_size,v_table,p_page;
 return v_result;
end $function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_lookups"(text, text, integer, integer) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_loop_detail (
  p_id integer
)
  RETURNS jsonb
  LANGUAGE sql
  STABLE
  SET search_path TO ''
  AS $function$
 select jsonb_build_object('loop', public.quest_admin_wire(to_jsonb(l)), 'revision',md5(to_jsonb(l)::text),
 'activeUse',jsonb_build_object('activeLoops',(select count(*) from player.character_quest_loops where loop_id=l.id and active),
 'activeQuests',(select count(*) from player.character_quests cq join data.data_tbl_quest q on q.id=cq.quest_id where cq.status in (0,1) and q.type=7 and q.sub_type='7-' || case when l.id=1 then 10 else l.id end::text)),
 'affectedQuestCount',(select count(*) from data.data_tbl_quest q where q.type=7 and q.sub_type='7-' || case when l.id=1 then 10 else l.id end::text))
 from data.data_tbl_quest_loop l where l.id=p_id
$function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_loop_detail"(integer) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_save (
  p_id                integer,
  p_expected_revision text,
  p_patch             jsonb,
  p_reason            text,
  p_request_id        uuid,
  p_actor_id          uuid
)
  RETURNS jsonb
  LANGUAGE plpgsql
  SET search_path TO ''
  AS $function$
declare before_data jsonb; after_data jsonb; old_hash text; new_hash text; req_hash text; prior data.quest_admin_audit%rowtype;
 grp text; tbl text; delta jsonb; entry jsonb; old_row jsonb; changes jsonb; row_id integer; ids integer[]; next_id integer;
begin
 if p_actor_id is null or p_request_id is null or p_expected_revision !~ '^[a-f0-9]{32}$' or p_expected_revision is null or length(trim(p_reason)) not between 3 and 1000 or p_reason is null or pg_column_size(p_patch)>131072 or jsonb_typeof(p_patch)<>'object' or p_patch is null or p_patch='{}'::jsonb then raise exception using errcode='22023',message='Invalid save envelope'; end if;
 if exists(select 1 from jsonb_object_keys(p_patch) k where k not in ('quest','requirements','awards','prerequisites')) then raise exception using errcode='22023',message='Unknown patch group'; end if;
 -- Serialize all catalog writers, including non-cooperating table DML, and max-ID allocation.
 lock table data.data_tbl_quest,data.data_tbl_quest_require,data.data_tbl_quest_award,data.data_tbl_quest_pre,data.data_tbl_quest_loop in share row exclusive mode;
 req_hash:=md5(jsonb_build_array(p_id,p_expected_revision,p_patch,p_reason,p_actor_id)::text);
 select * into prior from data.quest_admin_audit where request_id=p_request_id;
 if found then
  if prior.request_hash<>req_hash or prior.actor_id<>p_actor_id or prior.target_kind<>'quest' then raise exception using errcode='40001',message='Request ID conflict'; end if;
  return prior.after_image || jsonb_build_object('revision',prior.new_revision);
 end if;
 before_data:=public.quest_admin_snapshot(p_id);
 if before_data is null then raise exception using errcode='P0002',message='Quest not found'; end if;
 old_hash:=md5(before_data::text);
 if old_hash<>p_expected_revision then raise exception using errcode='40001',message='Quest revision conflict'; end if;
 if exists(select 1 from player.character_quests where quest_id=p_id and status in (0,1)) and
 (exists(select 1 from jsonb_object_keys(coalesce(p_patch->'quest','{}'::jsonb)) k where k not in ('name','info','start_text','complete_text','color') and before_data->'quest'->k is distinct from p_patch->'quest'->k)
 or exists(select 1 from jsonb_each(p_patch) g where g.key in ('requirements','awards','prerequisites') and
 (jsonb_array_length(coalesce(g.value->'insert','[]'::jsonb))>0 or jsonb_array_length(coalesce(g.value->'update','[]'::jsonb))>0 or jsonb_array_length(coalesce(g.value->'deleteIds','[]'::jsonb))>0))) then
  raise exception using errcode='40001',message='ACTIVE_QUEST_IN_USE: only display metadata may change while persisted quests are active';
 end if;
 -- Before image must be durable in this same transaction before any catalog update.
 insert into data.quest_admin_audit(request_id,actor_id,target_kind,target_id,reason,backup_reference,request_hash,before_image,old_revision)
 values(p_request_id,p_actor_id,'quest',p_id,p_reason,'.local/backups/quest-admin/20261012-205511',req_hash,before_data,old_hash);
 if p_patch ? 'quest' then perform public.quest_admin_apply_fields('data_tbl_quest',p_id,before_data->'quest',p_patch->'quest','quest'); end if;
 foreach grp in array array['requirements','awards','prerequisites'] loop
  if not p_patch ? grp then continue; end if;
  delta:=p_patch->grp;
  tbl:=case grp when 'requirements' then 'data_tbl_quest_require' when 'awards' then 'data_tbl_quest_award' else 'data_tbl_quest_pre' end;
  if jsonb_typeof(delta)<>'object' or exists(select 1 from jsonb_object_keys(delta) k where k not in ('insert','update','deleteIds')) then raise exception using errcode='22023',message='Invalid child delta'; end if;
  foreach changes in array array[coalesce(delta->'insert','[]'::jsonb),coalesce(delta->'update','[]'::jsonb),coalesce(delta->'deleteIds','[]'::jsonb)] loop
   if jsonb_typeof(changes)<>'array' or jsonb_array_length(changes)>100 then raise exception using errcode='22023',message='Invalid child array'; end if;
  end loop;
  ids:=array[]::integer[];
  for entry in select value from jsonb_array_elements(coalesce(delta->'update','[]'::jsonb)) loop
   if jsonb_typeof(entry)<>'object' or exists(select 1 from jsonb_object_keys(entry) k where k not in ('id','changes')) or jsonb_typeof(entry->'id')<>'number' or (entry->>'id') !~ '^[1-9][0-9]{0,9}$' then raise exception using errcode='22023',message='Invalid child update'; end if;
   row_id:=(entry->>'id')::integer;
   if row_id=any(ids) then raise exception using errcode='22023',message='Duplicate row ID'; end if;
   ids:=array_append(ids,row_id);
   execute format('select public.quest_admin_wire(to_jsonb(t)) from data.%I t where id=$1 and qid=$2',tbl) into old_row using row_id,p_id;
   if old_row is null then raise exception using errcode='22023',message='Child does not belong to quest'; end if;
   if entry->'changes' is null or jsonb_typeof(entry->'changes')<>'object' then raise exception using errcode='22023',message='Missing child changes'; end if;
   perform public.quest_admin_apply_fields(tbl,row_id,old_row,entry->'changes',grp);
  end loop;
  for entry in select value from jsonb_array_elements(coalesce(delta->'deleteIds','[]'::jsonb)) loop
   if jsonb_typeof(entry)<>'number' or entry::text !~ '^[1-9][0-9]{0,9}$' then raise exception using errcode='22023',message='Invalid delete ID'; end if;
   row_id:=entry::text::integer;
   if row_id=any(ids) then raise exception using errcode='22023',message='Duplicate row ID'; end if;
   ids:=array_append(ids,row_id);
   execute format('delete from data.%I where id=$1 and qid=$2 returning to_jsonb(id)',tbl) into old_row using row_id,p_id;
   if old_row is null then raise exception using errcode='22023',message='Child does not belong to quest'; end if;
  end loop;
  for entry in select value from jsonb_array_elements(coalesce(delta->'insert','[]'::jsonb)) loop
   if jsonb_typeof(entry)<>'object' or not (entry ?& (case grp when 'requirements' then array['item_id','kind','num','q','type'] when 'awards' then array['item_id','kind','num','q','type','b'] else array['item_id','kind','num','type'] end)) then raise exception using errcode='22023',message='Incomplete child insert'; end if;
   perform public.quest_admin_validate(grp,jsonb_build_object('qid',p_id,'id',0),entry);
   execute format('select coalesce(max(id),0)+1 from data.%I',tbl) into next_id;
   execute format('insert into data.%I select r.* from jsonb_populate_record(null::data.%I,$1) r',tbl,tbl) using entry || jsonb_build_object('id',next_id,'qid',p_id);
  end loop;
 end loop;
 -- Only newly added prerequisite edges can introduce new cycles. Old cycles remain untouched.
 for row_id in select distinct item_id::integer from data.data_tbl_quest_pre p where qid=p_id and kind=2 and type in (45,-1)
 and not exists(select 1 from jsonb_array_elements(before_data->'prerequisites') b where b->>'kind'='2' and b->>'type' in ('45','-1') and (b->>'item_id')::numeric=p.item_id) loop
  if row_id=p_id or exists(with recursive reach(id) as (select row_id union select p.item_id::integer from data.data_tbl_quest_pre p join reach r on p.qid=r.id where p.kind=2 and p.type in (45,-1)) select 1 from reach where id=p_id) then raise exception using errcode='22023',message='New prerequisite cycle'; end if;
 end loop;
 after_data:=public.quest_admin_detail(p_id);
 new_hash:=after_data->>'revision';
 update data.quest_admin_audit set after_image=after_data,new_revision=new_hash where request_id=p_request_id;
 return after_data;
end $function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_save"(integer, text, jsonb, text, uuid, uuid) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_save_loop (
  p_id                      integer,
  p_expected_revision       text,
  p_expected_affected_count integer,
  p_confirm_shared          boolean,
  p_patch                   jsonb,
  p_reason                  text,
  p_request_id              uuid,
  p_actor_id                uuid
)
  RETURNS jsonb
  LANGUAGE plpgsql
  SET search_path TO ''
  AS $function$
declare before_data jsonb; after_data jsonb; req_hash text; prior data.quest_admin_audit%rowtype;
begin
 if p_confirm_shared is distinct from true or p_actor_id is null or p_request_id is null or p_expected_revision is null or p_expected_revision !~ '^[a-f0-9]{32}$' or p_expected_affected_count is null or p_expected_affected_count<0 or p_reason is null or length(trim(p_reason)) not between 3 and 1000 or pg_column_size(p_patch)>131072 or p_patch is null or jsonb_typeof(p_patch)<>'object' or p_patch='{}'::jsonb then raise exception using errcode='22023',message='Invalid shared loop envelope'; end if;
 lock table data.data_tbl_quest,data.data_tbl_quest_require,data.data_tbl_quest_award,data.data_tbl_quest_pre,data.data_tbl_quest_loop in share row exclusive mode;
 req_hash:=md5(jsonb_build_array('loop',p_id,p_expected_revision,p_expected_affected_count,p_patch,p_reason,p_actor_id)::text);
 select * into prior from data.quest_admin_audit where request_id=p_request_id;
 if found then
  if prior.request_hash<>req_hash or prior.actor_id<>p_actor_id or prior.target_kind<>'loop' then raise exception using errcode='40001',message='Request ID conflict'; end if;
  return prior.after_image;
 end if;
 before_data:=public.quest_admin_loop_detail(p_id);
 if before_data is null then raise exception using errcode='P0002',message='Loop not found'; end if;
 if before_data->>'revision'<>p_expected_revision or (before_data->>'affectedQuestCount')::integer<>p_expected_affected_count then raise exception using errcode='40001',message='Shared loop revision or impact conflict'; end if;
 if exists(select 1 from jsonb_object_keys(p_patch) k where k not in ('name','info') and before_data->'loop'->k is distinct from p_patch->k)
 and (exists(select 1 from player.character_quest_loops where loop_id=p_id and active)
 or exists(select 1 from player.character_quests cq join data.data_tbl_quest q on q.id=cq.quest_id where cq.status in (0,1) and q.type=7 and q.sub_type='7-' || case when p_id=1 then 10 else p_id end::text)) then
  raise exception using errcode='40001',message='ACTIVE_QUEST_IN_USE: shared loop semantics are in use';
 end if;
 insert into data.quest_admin_audit(request_id,actor_id,target_kind,target_id,reason,backup_reference,request_hash,before_image,old_revision)
 values(p_request_id,p_actor_id,'loop',p_id,p_reason,'.local/backups/quest-admin/20261012-205511',req_hash,before_data,p_expected_revision);
 perform public.quest_admin_apply_fields('data_tbl_quest_loop',p_id,before_data->'loop',p_patch,'loop');
 after_data:=public.quest_admin_loop_detail(p_id);
 update data.quest_admin_audit set after_image=after_data,new_revision=after_data->>'revision' where request_id=p_request_id;
 return after_data;
end $function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_save_loop"(integer, text, integer, boolean, jsonb, text, uuid, uuid) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_snapshot (
  p_id integer
)
  RETURNS jsonb
  LANGUAGE sql
  STABLE
  SET search_path TO ''
  AS $function$
 select jsonb_build_object('questId',q.id,'quest',public.quest_admin_wire(to_jsonb(q)),
 'requirements',coalesce((select jsonb_agg(public.quest_admin_wire(to_jsonb(r)) order by r.id) from data.data_tbl_quest_require r where r.qid=q.id),'[]'::jsonb),
 'awards',coalesce((select jsonb_agg(public.quest_admin_wire(to_jsonb(a)) order by a.id) from data.data_tbl_quest_award a where a.qid=q.id),'[]'::jsonb),
 'prerequisites',coalesce((select jsonb_agg(public.quest_admin_wire(to_jsonb(p)) order by p.id) from data.data_tbl_quest_pre p where p.qid=q.id),'[]'::jsonb))
 from data.data_tbl_quest q where q.id=p_id
$function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_snapshot"(integer) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_validate (
  p_group   text,
  p_before  jsonb,
  p_changes jsonb
)
  RETURNS void
  LANGUAGE plpgsql
  SET search_path TO ''
  AS $function$
declare k text; v jsonb; t text; n numeric; row_data jsonb; allowed text[]; texts text[]; ref_table text; child_table text; found boolean; class_id text;
begin
 allowed := case p_group
 when 'quest' then array['name','info','start_text','complete_text','type','sub_type','at','award_exp_re','color','start_npc','finish_npc','gender','is_rebirth','lm','max_level','min_level','money_type','money_num','pre_quest_type','req_class','rt','rn','start_grant_item','start_grant_qty']
 when 'requirements' then array['item_id','kind','num','q','type']
 when 'awards' then array['item_id','kind','num','q','type','b']
 when 'prerequisites' then array['item_id','kind','num','type']
 when 'loop' then array['name','info','nid','num','min_level','max_level','refresh','team'] end;
 texts := array['name','info','start_text','complete_text','sub_type','req_class'];
 if allowed is null or jsonb_typeof(p_changes)<>'object' or p_changes='{}'::jsonb then raise exception using errcode='22023',message='Empty or invalid patch'; end if;
 row_data := coalesce(p_before,'{}'::jsonb) || p_changes;
 for k,v in select * from jsonb_each(p_changes) loop
  if not k=any(allowed) then raise exception using errcode='22023',message='Unknown or immutable field: ' || k; end if;
  if p_before ? k and p_before->k=v then continue; end if;
  t := v #>> '{}';
  if k=any(texts) then
   if v<>'null'::jsonb and (jsonb_typeof(v)<>'string' or length(t)>16000 or (k='name' and (length(t)=0 or length(t)>300))) then raise exception using errcode='22023',message='Invalid text: ' || k; end if;
   if k='name' and v='null'::jsonb then raise exception using errcode='22023',message='Name required'; end if;
   if k='req_class' and t is not null and t<>'all' then
    if t !~ '^\|[1-9][0-9]*(\|[1-9][0-9]*)*\|$' then raise exception using errcode='22023',message='Invalid req_class'; end if;
    for class_id in select unnest(string_to_array(trim(both '|' from t),'|')) loop
     if not exists(select 1 from data.data_tbl_class where id::text=class_id) then raise exception using errcode='22023',message='Unknown class'; end if;
    end loop;
   end if;
   continue;
  end if;
  if v='null'::jsonb then
   if k in ('is_rebirth','start_grant_item','start_grant_qty') then continue; end if;
   raise exception using errcode='22023',message='Null numeric: ' || k;
  end if;
  if k in ('start_grant_item','start_grant_qty') then
   if jsonb_typeof(v)<>'number' or t !~ '^[1-9][0-9]{0,9}$' or t::numeric>2147483647 then raise exception using errcode='22023',message='Invalid integer ID'; end if;
  elsif jsonb_typeof(v)<>'string' or t !~ '^(-1|0|[1-9][0-9]{0,14})$' then raise exception using errcode='22023',message='Invalid decimal string: ' || k;
  end if;
  n:=t::numeric;
  if n<0 and k not in ('start_npc','finish_npc','nid','lm','q','money_type') then raise exception using errcode='22023',message='Negative field: ' || k; end if;
  if k in ('item_id','num','refresh','start_grant_qty') and n<=0 then raise exception using errcode='22023',message='Positive required: ' || k; end if;
  if k='at' and t<>'1' then raise exception using errcode='22023',message='Award selection unsupported'; end if;
  if k='type' and p_group='quest' and n not between 1 and 12 then raise exception using errcode='22023',message='Unknown quest type'; end if;
  if k='color' and n not between 0 and 4 or k='gender' and n not between 0 and 2 or k='pre_quest_type' and n not between 0 and 2 or k='is_rebirth' and n not between 0 and 1 or k='team' and n not between 0 and 2 then raise exception using errcode='22023',message='Unknown enum: ' || k; end if;
  if k in ('rt','rn') and n<>0 then raise exception using errcode='22023',message='Currency requirement unsupported'; end if;
  if k='money_type' and n not in (-1,0,1,2,3,4) then raise exception using errcode='22023',message='Unverified currency mapping'; end if;
  if k='b' and n<>1 then raise exception using errcode='22023',message='Unbound reward unsupported'; end if;
  if k in ('start_npc','finish_npc','nid') and n>0 and not exists(select 1 from data.data_tbl_npc where id=n) then raise exception using errcode='22023',message='Unknown NPC: ' || k; end if;
  if k='start_grant_item' and not exists(select 1 from data.data_tbl_item_template where id=n) then raise exception using errcode='22023',message='Unknown start grant item'; end if;
 end loop;
 if p_group='quest' then
  if (p_changes ? 'money_type' or p_changes ? 'money_num') and (row_data->>'money_num')::numeric>0 and (row_data->>'money_type')::numeric not in (1,2,3,4) then raise exception using errcode='22023',message='Unsupported reward currency'; end if;
  if (p_changes ? 'min_level' or p_changes ? 'max_level') and (row_data->>'min_level')::numeric>(row_data->>'max_level')::numeric then raise exception using errcode='22023',message='Inverted level range'; end if;
  if p_changes ? 'sub_type' and row_data->>'type'='7' and row_data->>'sub_type' is not null then
   if row_data->>'sub_type' !~ '^7-[0-9]{1,9}$' or not exists(select 1 from data.data_tbl_quest_loop where id=case when substring(row_data->>'sub_type' from 3)::integer=10 then 1 else substring(row_data->>'sub_type' from 3)::integer end) then raise exception using errcode='22023',message='Unknown loop pool'; end if;
  end if;
 elsif p_group='loop' then
  if (p_changes ? 'min_level' or p_changes ? 'max_level') and (row_data->>'min_level')::numeric>(row_data->>'max_level')::numeric then raise exception using errcode='22023',message='Inverted loop level range'; end if;
 elsif p_group in ('requirements','awards','prerequisites') then
  if p_group='prerequisites' then
   ref_table:=case when row_data->>'kind'='1' and row_data->>'type'='29' then 'data_tbl_item_template' when row_data->>'kind'='2' and row_data->>'type' in ('45','-1') then 'data_tbl_quest' end;
  else
   ref_table:=case when row_data->>'kind'='1' and row_data->>'type'='29' then 'data_tbl_item_template' when row_data->>'kind'='1' and row_data->>'type'='19' then 'data_tbl_equipt_template' when row_data->>'type'='12' and (row_data->>'kind'='2' or (p_group='requirements' and row_data->>'kind'='3')) then 'data_tbl_creature' end;
  end if;
  if ref_table is null then raise exception using errcode='22023',message='Unsupported kind/type pair'; end if;
  execute format('select exists(select 1 from data.%I where id=$1)',ref_table) into found using (row_data->>'item_id')::numeric;
  if not found then raise exception using errcode='22023',message='Unknown catalog target'; end if;
  if p_group='awards' then
   if (row_data->>'b') is distinct from '1' then raise exception using errcode='22023',message='Edited awards must be bound'; end if;
   if row_data->>'type'='12' and row_data->>'num'<>'1' then raise exception using errcode='22023',message='Pet reward quantity must be one'; end if;
  end if;
  if p_changes ? 'item_id' or p_changes ? 'kind' or p_changes ? 'type' or p_changes ? 'q' then
   child_table:=case p_group when 'requirements' then 'data_tbl_quest_require' when 'awards' then 'data_tbl_quest_award' else 'data_tbl_quest_pre' end;
   if p_before ? 'qid' then
    execute format('select exists(select 1 from data.%I where qid=$1 and kind=$2 and type=$3 and item_id=$4 and id<>$5 %s)',child_table,case when p_group='prerequisites' then '' else 'and q=$6' end)
    into found using (p_before->>'qid')::numeric,(row_data->>'kind')::numeric,(row_data->>'type')::numeric,(row_data->>'item_id')::numeric,(p_before->>'id')::integer,(row_data->>'q')::numeric;
    if found then raise exception using errcode='22023',message='New duplicate target'; end if;
   end if;
  end if;
 end if;
end $function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_validate"(text, jsonb, jsonb) FROM PUBLIC, "anon", "authenticated";

CREATE OR REPLACE FUNCTION public.quest_admin_wire (
  p_row jsonb
)
  RETURNS jsonb
  LANGUAGE sql
  IMMUTABLE
  SET search_path TO ''
  AS $function$
 select coalesce(jsonb_object_agg(key, case when jsonb_typeof(value)='number'
   and key not in ('id','start_grant_item','start_grant_qty') then to_jsonb(value::text) else value end),'{}'::jsonb)
 from jsonb_each(p_row)
$function$;

REVOKE ALL ON FUNCTION "public"."quest_admin_wire"(jsonb) FROM PUBLIC, "anon", "authenticated";

REVOKE ALL ON FUNCTION "public"."quest_admin_apply_fields"(text, integer, jsonb, jsonb, text) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_apply_fields"(text, integer, jsonb, jsonb, text) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_apply_fields"(text, integer, jsonb, jsonb, text) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_detail"(integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_detail"(integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_detail"(integer) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_list"(text, integer, integer, integer, integer, integer, integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_list"(text, integer, integer, integer, integer, integer, integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_list"(text, integer, integer, integer, integer, integer, integer) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_lookups"(text, text, integer, integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_lookups"(text, text, integer, integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_lookups"(text, text, integer, integer) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_loop_detail"(integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_loop_detail"(integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_loop_detail"(integer) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_save"(integer, text, jsonb, text, uuid, uuid) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_save"(integer, text, jsonb, text, uuid, uuid) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_save"(integer, text, jsonb, text, uuid, uuid) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_save_loop"(integer, text, integer, boolean, jsonb, text, uuid, uuid) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_save_loop"(integer, text, integer, boolean, jsonb, text, uuid, uuid) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_save_loop"(integer, text, integer, boolean, jsonb, text, uuid, uuid) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_snapshot"(integer) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_snapshot"(integer) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_snapshot"(integer) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_validate"(text, jsonb, jsonb) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_validate"(text, jsonb, jsonb) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_validate"(text, jsonb, jsonb) TO "service_role";

REVOKE ALL ON FUNCTION "public"."quest_admin_wire"(jsonb) FROM "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_wire"(jsonb) TO "postgres";

GRANT EXECUTE ON FUNCTION "public"."quest_admin_wire"(jsonb) TO "service_role";

GRANT INSERT, SELECT, UPDATE ON TABLE "data"."quest_admin_audit" TO "service_role";
