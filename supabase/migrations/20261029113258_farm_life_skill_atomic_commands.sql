create table if not exists "player"."character_gathering_completions" (
    "attempt_id" text not null,
    "character_id" bigint not null references "player"."characters"("id") on delete cascade,
    "created_at" timestamp with time zone not null default now(),
    primary key ("attempt_id"),
    constraint "character_gathering_completions_attempt_check" check (length("attempt_id") = 32)
);

create index if not exists "idx_character_gathering_completions_character"
    on "player"."character_gathering_completions" ("character_id", "created_at" desc);

alter table "player"."character_gathering_completions" enable row level security;

create or replace function "player"."farm_life_item_json"(
    p_item "player"."character_items"
)
returns jsonb
language sql
security invoker
set search_path to ''
as $function$
    select jsonb_build_object(
        'id', p_item."id", 'characterId', p_item."character_id",
        'templateId', p_item."template_id", 'itemType', p_item."item_type",
        'slotType', p_item."slot_type", 'slotIndex', p_item."slot_index",
        'stackCount', p_item."stack_count", 'isBound', p_item."is_bound",
        'durability', p_item."durability", 'maxDurability', p_item."max_durability",
        'enchantLevel', p_item."enchant_level", 'starLevel', p_item."star_level",
        'colorCode', p_item."color_code", 'properties', p_item."properties",
        'createdAt', p_item."created_at", 'updatedAt', p_item."updated_at"
    );
$function$;

create or replace function "player"."farm_life_plot_json"(
    p_plot "player"."character_farm_plots"
)
returns jsonb
language sql
security invoker
set search_path to ''
as $function$
    select jsonb_build_object(
        'id', p_plot."id", 'characterId', p_plot."character_id",
        'plotNpcId', p_plot."plot_npc_id", 'cropNpcId', p_plot."crop_npc_id",
        'harvestItemTemplateId', p_plot."harvest_item_template_id",
        'harvestCount', p_plot."harvest_count", 'totalHarvestCount', p_plot."total_harvest_count",
        'sharedHarvesterIds', p_plot."shared_harvester_ids",
        'plantedAt', p_plot."planted_at", 'readyAt', p_plot."ready_at",
        'createdAt', p_plot."created_at", 'updatedAt', p_plot."updated_at"
    );
$function$;

create or replace function "player"."add_farm_life_item_atomic"(
    p_character_id bigint,
    p_template_id integer,
    p_count integer,
    p_slot_type integer
)
returns "player"."character_items"
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_item "player"."character_items"%rowtype;
    v_max_stack integer;
    v_max_slots integer;
    v_slot integer;
begin
    if p_character_id <= 0 or p_template_id <= 0 or p_count <= 0 or p_slot_type not in (0, 3) then
        raise exception 'INVALID_FARM_LIFE_ITEM';
    end if;

    select greatest(t."stack_max"::integer, 1)
    into v_max_stack
    from "data"."data_tbl_item_template" as t
    where t."id" = p_template_id;

    if v_max_stack is null or p_count > v_max_stack then
        raise exception 'INVALID_FARM_LIFE_ITEM';
    end if;

    select i.*
    into v_item
    from "player"."character_items" as i
    where i."character_id" = p_character_id
      and i."template_id" = p_template_id
      and i."item_type" = 3
      and i."slot_type" = p_slot_type
      and i."is_bound" = false
      and i."color_code" = 0
      and i."stack_count" + p_count <= v_max_stack
    order by i."id"
    limit 1
    for update;

    if found then
        update "player"."character_items" as i
        set "stack_count" = i."stack_count" + p_count,
            "updated_at" = now()
        where i."id" = v_item."id"
        returning i.* into v_item;
        return v_item;
    end if;

    v_max_slots := case when p_slot_type = 3 then 30 else 210 end;
    select candidate."slot_index"
    into v_slot
    from generate_series(0, v_max_slots - 1) as candidate("slot_index")
    where not exists (
        select 1
        from "player"."character_items" as occupied
        where occupied."character_id" = p_character_id
          and occupied."slot_type" = p_slot_type
          and occupied."slot_index" = candidate."slot_index"
    )
    order by candidate."slot_index"
    limit 1;

    if v_slot is null then
        raise exception 'INVENTORY_FULL';
    end if;

    insert into "player"."character_items" (
        "character_id", "template_id", "item_type", "slot_type", "slot_index",
        "stack_count", "is_bound", "color_code", "properties"
    ) values (
        p_character_id, p_template_id, 3, p_slot_type, v_slot,
        p_count, false, 0, '{}'::jsonb
    ) returning * into v_item;

    return v_item;
end;
$function$;

create or replace function "player"."complete_life_gathering_atomic"(
    p_attempt_id text,
    p_character_id bigint,
    p_kind integer,
    p_npc_id integer,
    p_resource_level integer,
    p_reward_template_id integer,
    p_tool_item_id bigint,
    p_succeeded boolean,
    p_vigor_cost integer,
    p_mastery_gain integer
)
returns table(character_json jsonb, skill_json jsonb, reward_json jsonb, tool_json jsonb, succeeded boolean)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_attributes "player"."character_attributes"%rowtype;
    v_life "player"."character_life_skills"%rowtype;
    v_skill "player"."character_skills"%rowtype;
    v_tool "player"."character_items"%rowtype;
    v_reward "player"."character_items"%rowtype;
    v_skill_type integer;
    v_expected_level integer;
    v_expected_mastery integer;
    v_rewards integer[];
begin
    if p_attempt_id is null or length(p_attempt_id) <> 32 or p_character_id <= 0
       or p_kind not in (1, 2) or p_resource_level not between 1 and 10
       or p_reward_template_id <= 0 or p_mastery_gain not between 0 and 2 then
        raise exception 'INVALID_GATHERING_COMMAND';
    end if;

    perform pg_advisory_xact_lock(p_character_id);

    select c.* into v_character
    from "player"."characters" as c
    where c."id" = p_character_id
    for update;
    if not found then
        raise exception 'CHARACTER_NOT_FOUND';
    end if;
    select a.* into v_attributes from "player"."character_attributes" as a
    where a."character_id" = p_character_id for update;
    select l.* into v_life from "player"."character_life_skills" as l
    where l."character_id" = p_character_id for update;
    if v_attributes."character_id" is null or v_life."character_id" is null then
        raise exception 'CHARACTER_NOT_FOUND';
    end if;

    if p_kind = 1 then
        v_skill_type := 14;
        v_expected_level := case
            when p_npc_id between 1189 and 1198 then 1
            when p_npc_id in (1199, 1202, 1203, 1204) then 2
            when p_npc_id in (1205, 1206, 1207) then 3
            when p_npc_id in (1208, 1209, 1210, 1211) then 4
            when p_npc_id in (1212, 1213, 1214, 1215) then 5
            when p_npc_id in (1216, 1217, 1218) then 6
            when p_npc_id in (1596, 1597, 1598) then 7
            else 0
        end;
        if p_vigor_cost <> 0 then
            raise exception 'INVALID_GATHERING_COMMAND';
        end if;
    else
        v_skill_type := 16;
        v_expected_level := case when p_npc_id between 1241 and 1250 then p_npc_id - 1240 else 0 end;
        if p_vigor_cost <> 2 or not p_succeeded or p_tool_item_id <> 0 then
            raise exception 'INVALID_GATHERING_COMMAND';
        end if;
    end if;

    if v_expected_level = 0 or v_expected_level <> p_resource_level then
        raise exception 'GATHERING_RESOURCE_UNAVAILABLE';
    end if;

    v_rewards := case p_kind
        when 1 then case p_resource_level
            when 1 then array[2361,2362] when 2 then array[2363,2364]
            when 3 then array[2365,2366] when 4 then array[2367,2368]
            when 5 then array[2369,2370] when 6 then array[2589,2588]
            when 7 then array[2591,2592] when 8 then array[2594,2595]
            when 9 then array[2597,2599] when 10 then array[2600,2602] end
        when 2 then case p_resource_level
            when 1 then array[2491,2492] when 2 then array[2493,2494]
            when 3 then array[2495,2496] when 4 then array[2497,2498]
            when 5 then array[2499,2500] when 6 then array[2604,2603]
            when 7 then array[2606,2605] when 8 then array[2607,2608]
            when 9 then array[2609,2610] when 10 then array[2611,2612] end
    end;
    if not p_reward_template_id = any(v_rewards) then
        raise exception 'GATHERING_RESOURCE_UNAVAILABLE';
    end if;

    select cs.* into v_skill
    from "player"."character_skills" as cs
    join "data"."data_tbl_skill" as st on st."id" = cs."skill_id"
    where cs."character_id" = p_character_id and st."type" = v_skill_type
    order by cs."level" desc, cs."id"
    limit 1
    for update of cs;
    if not found then
        raise exception 'GATHERING_SKILL_NOT_LEARNED';
    end if;
    if v_skill."level" < p_resource_level then
        raise exception 'GATHERING_SKILL_LEVEL_NOT_ENOUGH';
    end if;

    v_expected_mastery := case v_skill."level" - p_resource_level when 0 then 2 when 1 then 1 else 0 end;
    if not p_succeeded then
        v_expected_mastery := 0;
    end if;
    if p_mastery_gain <> v_expected_mastery then
        raise exception 'INVALID_GATHERING_COMMAND';
    end if;

    if p_kind = 1 then
        select i.* into v_tool
        from "player"."character_items" as i
        join "data"."data_tbl_equipt_template" as e on e."id" = i."template_id"
        where i."id" = p_tool_item_id and i."character_id" = p_character_id
          and i."slot_type" = 1 and i."slot_index" = 13 and e."type" = 1201
        for update of i;
        if not found then
            raise exception 'GATHERING_TOOL_NOT_EQUIPPED';
        end if;
        if coalesce(v_tool."durability", 0) <= 0 then
            raise exception 'GATHERING_TOOL_BROKEN';
        end if;
    elsif v_attributes."current_sp" < p_vigor_cost then
        raise exception 'GATHERING_VIGOR_NOT_ENOUGH';
    end if;

    begin
        insert into "player"."character_gathering_completions" ("attempt_id", "character_id")
        values (p_attempt_id, p_character_id);
    exception when unique_violation then
        raise exception 'GATHERING_ATTEMPT_ALREADY_COMPLETED';
    end;

    if p_kind = 2 then
        update "player"."character_attributes" as a
        set "current_sp" = a."current_sp" - p_vigor_cost
        where a."character_id" = p_character_id
        returning a.* into v_attributes;
        update "player"."character_life_skills" as l
        set "herb_dex" = l."herb_dex" + p_mastery_gain
        where l."character_id" = p_character_id
        returning l.* into v_life;
    elsif p_succeeded then
        update "player"."character_life_skills" as l
        set "fish_dex" = l."fish_dex" + p_mastery_gain
        where l."character_id" = p_character_id
        returning l.* into v_life;
    end if;

    if p_succeeded then
        v_reward := "player"."add_farm_life_item_atomic"(p_character_id, p_reward_template_id, 1, 3);
    end if;

    if p_mastery_gain > 0 then
        update "player"."character_skills" as cs
        set "exp" = case when p_kind = 1 then v_life."fish_dex" else v_life."herb_dex" end
        where cs."id" = v_skill."id"
        returning cs.* into v_skill;
    end if;

    return query select
        jsonb_build_object('id', v_character."id", 'currentSp', v_attributes."current_sp", 'fishDex', v_life."fish_dex", 'herbDex', v_life."herb_dex"),
        jsonb_build_object('id', v_skill."id", 'characterId', v_skill."character_id", 'skillId', v_skill."skill_id", 'level', v_skill."level", 'exp', v_skill."exp", 'slotPosition', v_skill."slot_position", 'isAuto', v_skill."is_auto", 'cooldownEnd', v_skill."cooldown_end", 'createdAt', v_skill."created_at"),
        case when p_succeeded then "player"."farm_life_item_json"(v_reward) else null end,
        case when p_kind = 1 then "player"."farm_life_item_json"(v_tool) else null end,
        p_succeeded;
end;
$function$;

create or replace function "player"."plant_farm_crop_atomic"(
    p_character_id bigint,
    p_plot_npc_id integer,
    p_crop_npc_id integer,
    p_harvest_item_template_id integer,
    p_harvest_count integer,
    p_seed_template_id integer,
    p_vigor_cost integer,
    p_planted_at timestamp with time zone,
    p_ready_at timestamp with time zone
)
returns table(character_json jsonb, plot_json jsonb, seed_json jsonb)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_attributes "player"."character_attributes"%rowtype;
    v_plot "player"."character_farm_plots"%rowtype;
    v_seed "player"."character_items"%rowtype;
    v_expected_reward integer;
    v_base_count integer;
    v_withered_ids bigint[];
begin
    if p_character_id <= 0 or p_plot_npc_id not in (1228,1707,1708,1709,1710,1711,1712,1713,1714,1715,1716,1717,1718,1719,1720,1721,1722,1723,1724,1725)
       or p_vigor_cost <> 25 or p_planted_at is null or p_ready_at <= p_planted_at then
        raise exception 'FARM_INVALID_PLOT';
    end if;

    v_expected_reward := case p_crop_npc_id
        when 1229 then 2481 when 1230 then 2482 when 1231 then 2483 when 1232 then 2484
        when 1233 then 2485 when 1234 then 2486 when 1235 then 2487 when 1236 then 2488
        when 1239 then 2489 when 1240 then 2490 when 1525 then 2613 when 1526 then 2614
        when 1527 then 2615 when 1528 then 2616 when 1529 then case when p_seed_template_id = 2909 then 1156 else 2617 end
        else 0
    end;
    v_base_count := case p_crop_npc_id
        when 1229 then 11 when 1230 then 11 when 1231 then 10 when 1232 then 12
        when 1233 then 10 when 1234 then 10 when 1235 then 10 when 1236 then 10
        when 1239 then 10 when 1240 then 10 when 1525 then 10 when 1526 then 10
        when 1527 then 10 when 1528 then 10 when 1529 then case when p_seed_template_id = 2909 then 5 else 10 end
        else 0
    end;
    if v_expected_reward = 0 or p_harvest_item_template_id <> v_expected_reward
       or p_harvest_count not in (v_base_count, v_base_count * 5)
       or p_seed_template_id not in (0, 2909)
       or (p_seed_template_id = 2909 and (p_crop_npc_id <> 1529 or p_harvest_count <> 5)) then
        raise exception 'FARM_INVALID_CROP';
    end if;

    perform pg_advisory_xact_lock(p_character_id);
    perform pg_advisory_xact_lock(-p_plot_npc_id::bigint);

    select c.* into v_character from "player"."characters" as c where c."id" = p_character_id for update;
    if not found then raise exception 'CHARACTER_NOT_FOUND'; end if;
    select a.* into v_attributes from "player"."character_attributes" as a
    where a."character_id" = p_character_id for update;
    if not found then raise exception 'CHARACTER_NOT_FOUND'; end if;
    if v_attributes."current_sp" < p_vigor_cost then raise exception 'FARM_VIGOR_NOT_ENOUGH'; end if;

    delete from "player"."character_farm_plots" as fp
    where (fp."character_id" = p_character_id or fp."plot_npc_id" = p_plot_npc_id)
      and now() >= fp."ready_at" + ((fp."ready_at" - fp."planted_at") * 2 / 3);

    if exists (select 1 from "player"."character_farm_plots" as fp where fp."plot_npc_id" = p_plot_npc_id) then
        raise exception 'FARM_PLOT_OCCUPIED';
    end if;
    if exists (select 1 from "player"."character_farm_plots" as fp where fp."character_id" = p_character_id) then
        raise exception 'FARM_ANOTHER_PLOT_ACTIVE';
    end if;

    if p_seed_template_id > 0 then
        select i.* into v_seed from "player"."character_items" as i
        where i."character_id" = p_character_id and i."template_id" = p_seed_template_id
          and i."slot_type" = 0 and i."stack_count" > 0
        order by i."id" limit 1 for update;
        if not found then raise exception 'FARM_SEED_NOT_FOUND'; end if;
        if v_seed."stack_count" = 1 then
            delete from "player"."character_items" as i where i."id" = v_seed."id";
            v_seed."stack_count" := 0;
        else
            update "player"."character_items" as i set "stack_count" = i."stack_count" - 1, "updated_at" = now()
            where i."id" = v_seed."id" returning i.* into v_seed;
        end if;
    end if;

    update "player"."character_attributes" as a set "current_sp" = a."current_sp" - p_vigor_cost
    where a."character_id" = p_character_id returning a.* into v_attributes;

    insert into "player"."character_farm_plots" (
        "character_id", "plot_npc_id", "crop_npc_id", "harvest_item_template_id",
        "harvest_count", "total_harvest_count", "shared_harvester_ids", "planted_at", "ready_at"
    ) values (
        p_character_id, p_plot_npc_id, p_crop_npc_id, p_harvest_item_template_id,
        p_harvest_count, p_harvest_count, '{}', p_planted_at, p_ready_at
    ) returning * into v_plot;

    return query select
        jsonb_build_object('id', v_character."id", 'currentSp', v_attributes."current_sp"),
        "player"."farm_life_plot_json"(v_plot),
        case when p_seed_template_id > 0 then "player"."farm_life_item_json"(v_seed) else null end;
end;
$function$;

create or replace function "player"."harvest_farm_crop_atomic"(
    p_owner_character_id bigint,
    p_harvester_character_id bigint,
    p_plot_npc_id integer
)
returns table(plot_json jsonb, reward_json jsonb, reward_template_id integer, reward_count integer, shared boolean, plot_deleted boolean)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_plot "player"."character_farm_plots"%rowtype;
    v_reward "player"."character_items"%rowtype;
    v_count integer;
    v_shared boolean;
begin
    if p_owner_character_id <= 0 or p_harvester_character_id <= 0 or p_plot_npc_id <= 0 then
        raise exception 'FARM_INVALID_PLOT';
    end if;
    perform pg_advisory_xact_lock(p_harvester_character_id);
    perform pg_advisory_xact_lock(-p_plot_npc_id::bigint);

    select fp.* into v_plot from "player"."character_farm_plots" as fp
    where fp."character_id" = p_owner_character_id and fp."plot_npc_id" = p_plot_npc_id
    for update;
    if not found then raise exception 'FARM_PLOT_NOT_FOUND'; end if;
    if now() < v_plot."ready_at" then raise exception 'FARM_CROP_NOT_READY'; end if;
    if now() >= v_plot."ready_at" + ((v_plot."ready_at" - v_plot."planted_at") * 2 / 3) then
        raise exception 'FARM_PLOT_NOT_FOUND';
    end if;

    v_shared := p_owner_character_id <> p_harvester_character_id;
    v_count := v_plot."harvest_count";
    if v_shared then
        if p_harvester_character_id = any(v_plot."shared_harvester_ids") then raise exception 'FARM_PLOT_ALREADY_SHARED'; end if;
        if cardinality(v_plot."shared_harvester_ids") > 0 then raise exception 'FARM_PLOT_SHARED_OUT'; end if;
        v_count := greatest(v_plot."total_harvest_count" / 10, 1);
        if v_plot."harvest_count" - v_count < 1 then raise exception 'FARM_PLOT_SHARED_OUT'; end if;
    end if;

    v_reward := "player"."add_farm_life_item_atomic"(p_harvester_character_id, v_plot."harvest_item_template_id", v_count, 0);

    if v_shared then
        update "player"."character_farm_plots" as fp
        set "harvest_count" = fp."harvest_count" - v_count,
            "shared_harvester_ids" = array_append(fp."shared_harvester_ids", p_harvester_character_id),
            "updated_at" = now()
        where fp."id" = v_plot."id" returning fp.* into v_plot;
    else
        delete from "player"."character_farm_plots" as fp where fp."id" = v_plot."id";
    end if;

    return query select "player"."farm_life_plot_json"(v_plot), "player"."farm_life_item_json"(v_reward),
        v_plot."harvest_item_template_id", v_count, v_shared, not v_shared;
end;
$function$;

create or replace function "player"."ripen_and_harvest_farm_crop_atomic"(
    p_character_id bigint,
    p_plot_npc_id integer,
    p_gold_cost bigint
)
returns table(character_json jsonb, plot_json jsonb, reward_json jsonb, reward_template_id integer, reward_count integer, gold_charged bigint)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_plot "player"."character_farm_plots"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_reward "player"."character_items"%rowtype;
    v_charged bigint := 0;
begin
    if p_character_id <= 0 or p_plot_npc_id <= 0 or p_gold_cost <> 9 then raise exception 'FARM_INVALID_PLOT'; end if;
    perform pg_advisory_xact_lock(p_character_id);
    perform pg_advisory_xact_lock(-p_plot_npc_id::bigint);

    select fp.* into v_plot from "player"."character_farm_plots" as fp
    where fp."character_id" = p_character_id and fp."plot_npc_id" = p_plot_npc_id for update;
    if not found then raise exception 'FARM_PLOT_NOT_FOUND'; end if;
    if now() >= v_plot."ready_at" + ((v_plot."ready_at" - v_plot."planted_at") * 2 / 3) then raise exception 'FARM_PLOT_NOT_FOUND'; end if;

    select w.* into v_wallet from "player"."character_wallet" as w where w."character_id" = p_character_id for update;
    if not found then raise exception 'CHARACTER_NOT_FOUND'; end if;
    if now() < v_plot."ready_at" then
        if v_wallet."gold" < p_gold_cost then raise exception 'INSUFFICIENT_FUNDS'; end if;
        update "player"."character_wallet" as w set "gold" = w."gold" - p_gold_cost
        where w."character_id" = p_character_id returning w.* into v_wallet;
        v_charged := p_gold_cost;
    end if;

    v_reward := "player"."add_farm_life_item_atomic"(p_character_id, v_plot."harvest_item_template_id", v_plot."harvest_count", 0);
    delete from "player"."character_farm_plots" as fp where fp."id" = v_plot."id";

    return query select jsonb_build_object('id', p_character_id, 'gold', v_wallet."gold"),
        "player"."farm_life_plot_json"(v_plot), "player"."farm_life_item_json"(v_reward),
        v_plot."harvest_item_template_id", v_plot."harvest_count", v_charged;
end;
$function$;

create or replace function "player"."uproot_farm_crop_atomic"(
    p_character_id bigint,
    p_plot_npc_id integer,
    p_gold_cost bigint
)
returns table(character_json jsonb, plot_json jsonb)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_plot "player"."character_farm_plots"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
begin
    if p_character_id <= 0 or p_plot_npc_id <= 0 or p_gold_cost <> 1 then raise exception 'FARM_INVALID_PLOT'; end if;
    perform pg_advisory_xact_lock(p_character_id);
    perform pg_advisory_xact_lock(-p_plot_npc_id::bigint);

    select fp.* into v_plot from "player"."character_farm_plots" as fp
    where fp."character_id" = p_character_id and fp."plot_npc_id" = p_plot_npc_id for update;
    if not found then raise exception 'FARM_PLOT_NOT_FOUND'; end if;
    select w.* into v_wallet from "player"."character_wallet" as w where w."character_id" = p_character_id for update;
    if not found then raise exception 'CHARACTER_NOT_FOUND'; end if;
    if v_wallet."gold" < p_gold_cost then raise exception 'INSUFFICIENT_FUNDS'; end if;

    update "player"."character_wallet" as w set "gold" = w."gold" - p_gold_cost
    where w."character_id" = p_character_id returning w.* into v_wallet;
    delete from "player"."character_farm_plots" as fp where fp."id" = v_plot."id";

    return query select jsonb_build_object('id', p_character_id, 'gold', v_wallet."gold"), "player"."farm_life_plot_json"(v_plot);
end;
$function$;

revoke all on table "player"."character_gathering_completions" from public, anon, authenticated;
grant select, insert, delete on table "player"."character_gathering_completions" to service_role;

revoke execute on function "player"."farm_life_item_json"("player"."character_items") from public, anon, authenticated;
revoke execute on function "player"."farm_life_plot_json"("player"."character_farm_plots") from public, anon, authenticated;
revoke execute on function "player"."add_farm_life_item_atomic"(bigint, integer, integer, integer) from public, anon, authenticated;
revoke execute on function "player"."complete_life_gathering_atomic"(text, bigint, integer, integer, integer, integer, bigint, boolean, integer, integer) from public, anon, authenticated;
revoke execute on function "player"."plant_farm_crop_atomic"(bigint, integer, integer, integer, integer, integer, integer, timestamp with time zone, timestamp with time zone) from public, anon, authenticated;
revoke execute on function "player"."harvest_farm_crop_atomic"(bigint, bigint, integer) from public, anon, authenticated;
revoke execute on function "player"."ripen_and_harvest_farm_crop_atomic"(bigint, integer, bigint) from public, anon, authenticated;
revoke execute on function "player"."uproot_farm_crop_atomic"(bigint, integer, bigint) from public, anon, authenticated;

grant execute on function "player"."farm_life_item_json"("player"."character_items") to service_role;
grant execute on function "player"."farm_life_plot_json"("player"."character_farm_plots") to service_role;
grant execute on function "player"."add_farm_life_item_atomic"(bigint, integer, integer, integer) to service_role;
grant execute on function "player"."complete_life_gathering_atomic"(text, bigint, integer, integer, integer, integer, bigint, boolean, integer, integer) to service_role;
grant execute on function "player"."plant_farm_crop_atomic"(bigint, integer, integer, integer, integer, integer, integer, timestamp with time zone, timestamp with time zone) to service_role;
grant execute on function "player"."harvest_farm_crop_atomic"(bigint, bigint, integer) to service_role;
grant execute on function "player"."ripen_and_harvest_farm_crop_atomic"(bigint, integer, bigint) to service_role;
grant execute on function "player"."uproot_farm_crop_atomic"(bigint, integer, bigint) to service_role;
