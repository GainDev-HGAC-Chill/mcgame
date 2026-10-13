create or replace function public.quest_admin_validate(p_group text,p_before jsonb,p_changes jsonb) returns void
language plpgsql security invoker set search_path = '' as $$
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
  if k='q' and n>9999 then raise exception using errcode='22023',message='Quality out of range'; end if;
  if k in ('start_npc','finish_npc','nid') and n>0 and not exists(select 1 from data.data_tbl_npc where id=n) then raise exception using errcode='22023',message='Unknown NPC: ' || k; end if;
  if k='start_grant_item' and not exists(select 1 from data.data_tbl_item_template where id=n) then raise exception using errcode='22023',message='Unknown start grant item'; end if;
 end loop;
 if p_group='quest' then
  if (p_changes ? 'money_type' or p_changes ? 'money_num') and (row_data->>'money_num')::numeric>0 and (row_data->>'money_type')::numeric not in (1,2,3,4) then raise exception using errcode='22023',message='Unsupported reward currency'; end if;
  if (p_changes ? 'min_level' or p_changes ? 'max_level') and (row_data->>'min_level')::numeric>(row_data->>'max_level')::numeric then raise exception using errcode='22023',message='Inverted level range'; end if;
  if (p_changes ? 'sub_type' or p_changes ? 'type') and row_data->>'type'='7' then
   if row_data->>'sub_type' is null or row_data->>'sub_type' !~ '^7-[0-9]{1,9}$' then raise exception using errcode='22023',message='Invalid loop pool'; end if;
   if not exists(select 1 from data.data_tbl_quest_loop where id=case when substring(row_data->>'sub_type' from 3)::integer=10 then 1 else substring(row_data->>'sub_type' from 3)::integer end) then raise exception using errcode='22023',message='Unknown loop pool'; end if;
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
end $$;
revoke all on function public.quest_admin_validate(text,jsonb,jsonb) from public,anon,authenticated;
grant execute on function public.quest_admin_validate(text,jsonb,jsonb) to service_role;
