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
 for row_id in select distinct item_id::integer from data.data_tbl_quest_pre p where qid=p_id and kind=2 and type in (45,-1)
 and not exists(select 1 from jsonb_array_elements(before_data->'prerequisites') b where b->>'kind'='2' and b->>'type' in ('45','-1') and (b->>'item_id')::numeric=p.item_id) loop
  if row_id=p_id or exists(with recursive reach(id) as (select row_id union select p.item_id::integer from data.data_tbl_quest_pre p join reach r on p.qid=r.id where p.kind=2 and p.type in (45,-1)) select 1 from reach where id=p_id) then raise exception using errcode='22023',message='New prerequisite cycle'; end if;
 end loop;
 after_data:=public.quest_admin_detail(p_id);
 new_hash:=after_data->>'revision';
 update data.quest_admin_audit set after_image=after_data,new_revision=new_hash where request_id=p_request_id;
 return after_data;
end $function$;

REVOKE ALL ON FUNCTION public.quest_admin_save(integer, text, jsonb, text, uuid, uuid) FROM PUBLIC, anon, authenticated;
