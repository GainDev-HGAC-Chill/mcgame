create or replace function public.npc_admin_validate_slot(p_row jsonb) returns void
language plpgsql stable security invoker set search_path = '' as $$
declare k text; n numeric; integer_fields text[]:=array['id','amount','gold','gt','item_id','money','p_num1','p_num2','p_type1','p_type2','point','position','quality','r','sale','sid','st','type']; supported integer[]:=array[1,2,3,4,7,10,11,12,13,14,15,16,17,18,19,20,21,22,25,26,29,30,31,49,58,60,61,62,64,65,69,70,71];
begin
 if p_row is null or jsonb_typeof(p_row)<>'object' then raise exception using errcode='22023',message='Invalid shop slot'; end if;
 foreach k in array integer_fields loop
  if not p_row ? k or jsonb_typeof(p_row->k)<>'number' then raise exception using errcode='22023',message='Missing or non-numeric shop slot field: '||k; end if;
  n:=(p_row->>k)::numeric;
  if n<>trunc(n) or n>2147483647 or n<(case when k='amount' then -1 else 0 end) or (k='quality' and n>25) then raise exception using errcode='22023',message='Invalid shop slot integer: '||k; end if;
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
end $$;

revoke all on function public.npc_admin_validate_slot(jsonb) from public,anon,authenticated;
grant execute on function public.npc_admin_validate_slot(jsonb) to service_role;
