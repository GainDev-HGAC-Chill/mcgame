create or replace function public.quest_admin_apply_fields(p_table text,p_id integer,p_before jsonb,p_changes jsonb,p_group text) returns void
language plpgsql security invoker set search_path = '' as $$
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
end $$;

create or replace function public.quest_admin_save(p_id integer,p_expected_revision text,p_patch jsonb,p_reason text,p_request_id uuid,p_actor_id uuid) returns jsonb
language plpgsql security invoker set search_path = '' as $$
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
 values(p_request_id,p_actor_id,'quest',p_id,p_reason,'data.quest_admin_audit.before_image',req_hash,before_data,old_hash);
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
end $$;

create or replace function public.quest_admin_save_loop(p_id integer,p_expected_revision text,p_expected_affected_count integer,p_confirm_shared boolean,p_patch jsonb,p_reason text,p_request_id uuid,p_actor_id uuid) returns jsonb
language plpgsql security invoker set search_path = '' as $$
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
 values(p_request_id,p_actor_id,'loop',p_id,p_reason,'data.quest_admin_audit.before_image',req_hash,before_data,p_expected_revision);
 perform public.quest_admin_apply_fields('data_tbl_quest_loop',p_id,before_data->'loop',p_patch,'loop');
 after_data:=public.quest_admin_loop_detail(p_id);
 update data.quest_admin_audit set after_image=after_data,new_revision=after_data->>'revision' where request_id=p_request_id;
 return after_data;
end $$;

revoke all on function public.quest_admin_apply_fields(text,integer,jsonb,jsonb,text),public.quest_admin_save(integer,text,jsonb,text,uuid,uuid),public.quest_admin_save_loop(integer,text,integer,boolean,jsonb,text,uuid,uuid) from public,anon,authenticated;
grant execute on function public.quest_admin_apply_fields(text,integer,jsonb,jsonb,text),public.quest_admin_save(integer,text,jsonb,text,uuid,uuid),public.quest_admin_save_loop(integer,text,integer,boolean,jsonb,text,uuid,uuid) to service_role;
revoke insert,update,delete,truncate,references,trigger on data.data_tbl_quest,data.data_tbl_quest_require,data.data_tbl_quest_award,data.data_tbl_quest_pre,data.data_tbl_quest_loop from anon,authenticated;
notify pgrst, 'reload schema';
