create table if not exists "player"."guild_rank_settings" (
    "guild_id" bigint not null references "player"."guilds"("id") on delete cascade,
    "rank" integer not null check ("rank" between 1 and 6),
    "name" varchar(50) not null check (length("name") > 0),
    "can_add" boolean not null,
    "can_quest" boolean not null,
    "can_slot" boolean not null,
    "can_info" boolean not null,
    "can_del" boolean not null,
    "can_duty" boolean not null,
    primary key ("guild_id", "rank")
);

alter table "player"."guild_rank_settings" enable row level security;

create or replace function "player"."list_guild_rank_settings"(p_guild_id bigint)
returns table(
    rank integer,
    name varchar,
    can_add boolean,
    can_quest boolean,
    can_slot boolean,
    can_info boolean,
    can_del boolean,
    can_duty boolean
)
language sql
security invoker
set search_path to ''
as $function$
    select s."rank", s."name", s."can_add", s."can_quest", s."can_slot", s."can_info", s."can_del", s."can_duty"
    from "player"."guild_rank_settings" as s
    where s."guild_id" = p_guild_id
    order by s."rank";
$function$;

create or replace function "player"."replace_guild_rank_settings"(p_guild_id bigint, p_settings jsonb)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_count integer;
begin
    if jsonb_typeof(p_settings) <> 'array' or jsonb_array_length(p_settings) <> 6 then
        return false;
    end if;

    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        return false;
    end if;

    select count(distinct x."rank")
    into v_count
    from jsonb_to_recordset(p_settings) as x(
        "rank" integer,
        "name" text,
        "canAdd" boolean,
        "canQuest" boolean,
        "canSlot" boolean,
        "canInfo" boolean,
        "canDel" boolean,
        "canDuty" boolean
    )
    where x."rank" between 1 and 6
      and length(x."name") > 0
      and (x."rank" <> 1 or (x."canAdd" and x."canQuest" and x."canSlot" and x."canInfo" and x."canDel" and x."canDuty"));

    if v_count <> 6 then
        return false;
    end if;

    delete from "player"."guild_rank_settings" where "guild_id" = p_guild_id;
    insert into "player"."guild_rank_settings" (
        "guild_id", "rank", "name", "can_add", "can_quest", "can_slot", "can_info", "can_del", "can_duty"
    )
    select p_guild_id, x."rank", x."name", x."canAdd", x."canQuest", x."canSlot", x."canInfo", x."canDel", x."canDuty"
    from jsonb_to_recordset(p_settings) as x(
        "rank" integer,
        "name" text,
        "canAdd" boolean,
        "canQuest" boolean,
        "canSlot" boolean,
        "canInfo" boolean,
        "canDel" boolean,
        "canDuty" boolean
    );
    return true;
end;
$function$;

create or replace function "player"."create_guild_atomic"(
    p_leader_id bigint,
    p_name text,
    p_cost bigint,
    p_rank_settings jsonb
)
returns table(
    guild_id bigint,
    guild_name text,
    money bigint,
    money_bind bigint,
    gold bigint,
    guild_contrib bigint,
    donate_contrib bigint,
    restore_normal bigint,
    restore_donate bigint
)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_level integer;
    v_guild_id bigint;
    v_normal bigint;
    v_donate bigint;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_leader_id::text, 0));
    if p_cost <> 1000000 or p_name is null or length(btrim(p_name)) = 0 or length(p_name) > 50 then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    select * into v_character
    from "player"."characters"
    where "id" = p_leader_id
    for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select "level" into v_level from "player"."character_progression" where "character_id" = p_leader_id for update;
    if not found or v_level < 50 then
        raise exception 'INSUFFICIENT_LEVEL';
    end if;
    select * into v_wallet from "player"."character_wallet" where "character_id" = p_leader_id for update;
    if not found then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    if v_character."guild_id" is not null or exists (
        select 1 from "player"."guild_members" where "character_id" = p_leader_id
    ) or exists (select 1 from "player"."guild_applications" where "character_id" = p_leader_id) then
        raise exception 'ALREADY_IN_GUILD';
    end if;
    if exists (select 1 from "player"."guilds" where "name" = p_name) then
        raise exception 'GUILD_NAME_TAKEN';
    end if;
    if v_wallet."money_bind" >= p_cost then
        v_wallet."money_bind" := v_wallet."money_bind" - p_cost;
    elsif v_wallet."money" >= p_cost then
        v_wallet."money" := v_wallet."money" - p_cost;
    else
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    insert into "player"."guilds" (
        "name", "leader_id", "level", "experience", "population", "max_population", "icon", "funds",
        "contribution_total", "announcement", "description", "join_level_req", "join_approval_required", "activity_points"
    ) values (p_name, p_leader_id, 1, 0, 1, 50, 0, 0, 0, '', '', 1, true, 0)
    returning "id" into v_guild_id;

    insert into "player"."guild_members" ("guild_id", "character_id", "rank")
    values (v_guild_id, p_leader_id, 1);

    if not "player"."replace_guild_rank_settings"(v_guild_id, p_rank_settings) then
        raise exception 'INVALID_RANK_SETTINGS';
    end if;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_leader_id, 20, v_character."guild_restore_contrib", 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_normal;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_leader_id, 21, v_character."guild_restore_donate", 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_donate;

    update "player"."guild_members" as m
    set "contribution_normal" = v_normal,
        "contribution_donate" = v_donate,
        "contribution_total" = v_normal + v_donate
    where m."guild_id" = v_guild_id and m."character_id" = p_leader_id;

    update "player"."character_wallet"
    set "money" = v_wallet."money", "money_bind" = v_wallet."money_bind"
    where "character_id" = p_leader_id;

    update "player"."characters"
    set "guild_id" = v_guild_id,
        "guild_restore_contrib" = 0,
        "guild_restore_donate" = 0
    where "id" = p_leader_id;

    return query select v_guild_id, p_name, v_wallet."money", v_wallet."money_bind", v_wallet."gold",
        v_normal, v_donate, 0::bigint, 0::bigint;
end;
$function$;

create or replace function "player"."contribute_guild_money_atomic"(
    p_guild_id bigint,
    p_character_id bigint,
    p_silver bigint,
    p_gold bigint
)
returns table(
    character_name text,
    money bigint,
    money_bind bigint,
    gold bigint,
    guild_contrib bigint,
    donate_contrib bigint,
    restore_normal bigint,
    restore_donate bigint,
    guild_funds bigint,
    member_donate bigint,
    member_total bigint,
    member_weekly bigint
)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_member "player"."guild_members"%rowtype;
    v_funds_gain bigint;
    v_donate_gain bigint;
    v_donate_balance bigint;
    v_guild_funds bigint;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_character_id::text, 0));
    if p_silver < 0 or p_gold < 0 or (p_silver = 0 and p_gold = 0) or p_gold > 4611686018427387
       or p_silver > 9223372036854775807 - p_gold * 2000 then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    v_funds_gain := p_silver + p_gold * 2000;
    v_donate_gain := v_funds_gain / 8;

    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_character from "player"."characters" where "id" = p_character_id for update;
    if not found or v_character."guild_id" is distinct from p_guild_id then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_wallet from "player"."character_wallet" where "character_id" = p_character_id for update;
    if not found then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    select * into v_member
    from "player"."guild_members"
    where "guild_id" = p_guild_id and "character_id" = p_character_id
    for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    if v_wallet."money" < p_silver or v_wallet."gold" < p_gold then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    update "player"."character_wallet" as w
    set "money" = w."money" - p_silver,
        "gold" = w."gold" - p_gold
    where w."character_id" = p_character_id;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_character_id, 21, v_donate_gain, 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_donate_balance;

    update "player"."guild_members"
    set "contribution_donate" = "contribution_donate" + v_donate_gain,
        "contribution_total" = "contribution_total" + v_donate_gain,
        "contribution_weekly" = "contribution_weekly" + v_donate_gain
    where "id" = v_member."id"
    returning "contribution_donate", "contribution_total", "contribution_weekly"
    into v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";

    update "player"."guilds"
    set "funds" = "funds" + v_funds_gain,
        "contribution_total" = "contribution_total" + v_donate_gain
    where "id" = p_guild_id
    returning "funds" into v_guild_funds;

    return query select v_character."name"::text, v_wallet."money" - p_silver, v_wallet."money_bind",
        v_wallet."gold" - p_gold,
        coalesce((select "amount" from "player"."character_currencies" where "character_id" = p_character_id and "currency_type" = 20 and "season" = 0), 0),
        v_donate_balance, v_character."guild_restore_contrib"::bigint, v_character."guild_restore_donate"::bigint,
        v_guild_funds, v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";
end;
$function$;

create or replace function "player"."contribute_guild_material_atomic"(
    p_guild_id bigint,
    p_character_id bigint,
    p_item_id bigint,
    p_item_template_id integer,
    p_item_count integer,
    p_warehouse_template_id integer,
    p_warehouse_count integer,
    p_warehouse_max_stack integer,
    p_warehouse_page_count integer,
    p_donate_gain bigint
)
returns table(
    character_name text,
    item_remaining integer,
    warehouse_slot_id bigint,
    warehouse_slot_index integer,
    warehouse_stack_count integer,
    money bigint,
    money_bind bigint,
    gold bigint,
    guild_contrib bigint,
    donate_contrib bigint,
    restore_normal bigint,
    restore_donate bigint,
    member_donate bigint,
    member_total bigint,
    member_weekly bigint
)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_character "player"."characters"%rowtype;
    v_wallet "player"."character_wallet"%rowtype;
    v_member "player"."guild_members"%rowtype;
    v_item "player"."character_items"%rowtype;
    v_item_remaining integer;
    v_donate_balance bigint;
    v_slot_id bigint;
    v_slot_index integer;
    v_slot_count integer;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_character_id::text, 0));
    if p_item_count <= 0 or p_warehouse_count <= 0 or p_warehouse_max_stack <= 0
       or p_warehouse_count > p_warehouse_max_stack or p_warehouse_page_count not between 1 and 4
       or p_donate_gain < 0 then
        raise exception 'INVALID_CONTRIBUTION';
    end if;
    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_character from "player"."characters" where "id" = p_character_id for update;
    if not found or v_character."guild_id" is distinct from p_guild_id then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_wallet from "player"."character_wallet" where "character_id" = p_character_id for update;
    if not found then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;
    select * into v_member
    from "player"."guild_members"
    where "guild_id" = p_guild_id and "character_id" = p_character_id
    for update;
    if not found then
        raise exception 'NOT_IN_GUILD';
    end if;
    select * into v_item
    from "player"."character_items"
    where "id" = p_item_id and "character_id" = p_character_id
    for update;
    if not found or v_item."template_id" <> p_item_template_id or v_item."slot_type" <> 0 or v_item."stack_count" < p_item_count then
        raise exception 'INVALID_CONTRIBUTION';
    end if;

    v_item_remaining := v_item."stack_count" - p_item_count;
    if v_item_remaining = 0 then
        delete from "player"."character_items" where "id" = p_item_id;
    else
        update "player"."character_items" set "stack_count" = v_item_remaining where "id" = p_item_id;
    end if;

    if coalesce(("player"."load_latest_guild_log_details"(p_guild_id, 9002)->>'pageCount')::integer, 1) <> p_warehouse_page_count then
        raise exception 'INVALID_CONTRIBUTION';
    end if;
    perform 1 from "player"."guild_warehouse" as w where w."guild_id" = p_guild_id order by w."id" for update;
    select w."id", w."slot_index", w."stack_count"
    into v_slot_id, v_slot_index, v_slot_count
    from "player"."guild_warehouse" as w
    where w."guild_id" = p_guild_id and w."template_id" = p_warehouse_template_id
      and w."stack_count" > 0
      and w."stack_count"::bigint + p_warehouse_count::bigint <= p_warehouse_max_stack::bigint
    order by w."id"
    limit 1
    for update;
    if found then
        update "player"."guild_warehouse"
        set "stack_count" = "stack_count" + p_warehouse_count
        where "id" = v_slot_id
        returning "stack_count" into v_slot_count;
    else
        select min(candidate.slot_index)
        into v_slot_index
        from (
            select series.slot_index
            from generate_series(0, p_warehouse_page_count * 30 - 1) as series(slot_index)
            where not exists (
                select 1 from "player"."guild_warehouse" as occupied
                where occupied."guild_id" = p_guild_id and occupied."slot_index" = series.slot_index
            )
        ) as candidate;
        if v_slot_index is null then
            raise exception 'GUILD_WAREHOUSE_FULL';
        end if;
        insert into "player"."guild_warehouse" ("guild_id", "slot_index", "template_id", "stack_count", "deposited_by")
        values (p_guild_id, v_slot_index, p_warehouse_template_id, p_warehouse_count, p_character_id)
        returning "id", "stack_count" into v_slot_id, v_slot_count;
    end if;

    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_character_id, 21, p_donate_gain, 0)
    on conflict ("character_id", "currency_type", "season")
    do update set "amount" = "player"."character_currencies"."amount" + excluded."amount"
    returning "amount" into v_donate_balance;

    update "player"."guild_members"
    set "contribution_donate" = "contribution_donate" + p_donate_gain,
        "contribution_total" = "contribution_total" + p_donate_gain,
        "contribution_weekly" = "contribution_weekly" + p_donate_gain
    where "id" = v_member."id"
    returning "contribution_donate", "contribution_total", "contribution_weekly"
    into v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";

    update "player"."guilds"
    set "contribution_total" = "contribution_total" + p_donate_gain
    where "id" = p_guild_id;

    return query select v_character."name"::text, v_item_remaining, v_slot_id, v_slot_index, v_slot_count,
        v_wallet."money", v_wallet."money_bind", v_wallet."gold",
        coalesce((select "amount" from "player"."character_currencies" where "character_id" = p_character_id and "currency_type" = 20 and "season" = 0), 0),
        v_donate_balance, v_character."guild_restore_contrib"::bigint, v_character."guild_restore_donate"::bigint,
        v_member."contribution_donate", v_member."contribution_total", v_member."contribution_weekly";
end;
$function$;

create or replace function "player"."develop_guild_skill_atomic"(
    p_guild_id bigint, p_actor_id bigint, p_skill_id integer, p_previous_id integer,
    p_level integer, p_exp_cost bigint, p_money_cost bigint, p_family_ids integer[]
)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_guild "player"."guilds"%rowtype;
    v_count integer;
    v_current integer;
begin
    if p_level < 1 or p_level > 6 or p_exp_cost < 0 or p_money_cost < 0 or p_family_ids is null
       or cardinality(p_family_ids) <> 6
       or (select count(distinct family_id) from unnest(p_family_ids) as family(family_id)) <> 6
       or not (p_skill_id = any(p_family_ids)) then
        return false;
    end if;
    select * into v_guild from "player"."guilds" where "id" = p_guild_id for update;
    if not found or v_guild."leader_id" <> p_actor_id then
        return false;
    end if;
    if not exists (
        select 1 from "player"."guild_members"
        where "guild_id" = p_guild_id and "character_id" = p_actor_id and "rank" = 1
    ) then
        return false;
    end if;
    select count(*), coalesce(min("skill_id"), 0) into v_count, v_current
    from "player"."guild_skills"
    where "guild_id" = p_guild_id and "skill_id" = any(p_family_ids);
    if (p_previous_id = 0 and (v_count <> 0 or p_level <> 1))
       or (p_previous_id <> 0 and (v_count <> 1 or v_current <> p_previous_id or p_level <= 1)) then
        return false;
    end if;
    if v_guild."experience" < p_exp_cost or v_guild."funds" < p_money_cost then
        return false;
    end if;
    update "player"."guilds"
    set "experience" = "experience" - p_exp_cost, "funds" = "funds" - p_money_cost
    where "id" = p_guild_id;
    if p_previous_id <> 0 then
        delete from "player"."guild_skills" where "guild_id" = p_guild_id and "skill_id" = p_previous_id;
    end if;
    insert into "player"."guild_skills" ("guild_id", "skill_id", "level") values (p_guild_id, p_skill_id, p_level);
    return true;
end;
$function$;

create or replace function "player"."set_guild_name"(p_guild_id bigint, p_actor_id bigint, p_name text)
returns boolean
language sql
security invoker
set search_path to ''
as $function$
    with changed as (
        update "player"."guilds"
        set "name" = p_name
        where "id" = p_guild_id and "leader_id" = p_actor_id and length(btrim(p_name)) > 0
        returning 1
    )
    select exists(select 1 from changed);
$function$;

create or replace function "player"."guild_permission_authorized"(p_guild_id bigint, p_actor_id bigint, p_permission text)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_leader_id bigint;
    v_rank integer;
    v_allowed boolean;
begin
    select g."leader_id" into v_leader_id from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found then return false; end if;
    select m."rank" into v_rank from "player"."guild_members" as m where m."guild_id" = p_guild_id and m."character_id" = p_actor_id for update;
    if not found then return false; end if;
    if v_rank = 1 and v_leader_id = p_actor_id then return true; end if;
    if v_rank not between 2 and 6 or p_permission not in ('add', 'del', 'info', 'duty') then return false; end if;
    if p_permission = 'add' then
        select r."can_add" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    elsif p_permission = 'del' then
        select r."can_del" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    elsif p_permission = 'info' then
        select r."can_info" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    else
        select r."can_duty" into v_allowed from "player"."guild_rank_settings" as r where r."guild_id" = p_guild_id and r."rank" = v_rank for update;
    end if;
    return coalesce(v_allowed, false);
end;
$function$;

create or replace function "player"."replace_guild_rank_settings_authorized"(p_guild_id bigint, p_actor_id bigint, p_settings jsonb)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
begin
    perform 1 from "player"."guilds" as g where g."id" = p_guild_id and g."leader_id" = p_actor_id for update;
    if not found then return false; end if;
    return "player"."replace_guild_rank_settings"(p_guild_id, p_settings);
end;
$function$;

create or replace function "player"."set_guild_notice"(p_guild_id bigint, p_actor_id bigint, p_announcement text, p_description text)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
begin
    perform 1 from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found or not "player"."guild_permission_authorized"(p_guild_id, p_actor_id, 'info') then return false; end if;
    update "player"."guilds" set "announcement" = p_announcement, "description" = p_description where "id" = p_guild_id;
    return true;
end;
$function$;

revoke all on table "player"."guild_rank_settings" from public, anon, authenticated;
grant select, insert, update, delete on table "player"."guild_rank_settings" to service_role;
revoke execute on function "player"."list_guild_rank_settings"(bigint) from public, anon, authenticated;
revoke execute on function "player"."replace_guild_rank_settings"(bigint, jsonb) from public, anon, authenticated;
revoke execute on function "player"."create_guild_atomic"(bigint, text, bigint, jsonb) from public, anon, authenticated;
revoke execute on function "player"."contribute_guild_money_atomic"(bigint, bigint, bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."contribute_guild_material_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer, integer, bigint) from public, anon, authenticated;
revoke execute on function "player"."develop_guild_skill_atomic"(bigint, bigint, integer, integer, integer, bigint, bigint, integer[]) from public, anon, authenticated;
revoke execute on function "player"."set_guild_name"(bigint, bigint, text) from public, anon, authenticated;
revoke execute on function "player"."guild_permission_authorized"(bigint, bigint, text) from public, anon, authenticated;
revoke execute on function "player"."replace_guild_rank_settings_authorized"(bigint, bigint, jsonb) from public, anon, authenticated;
revoke execute on function "player"."set_guild_notice"(bigint, bigint, text, text) from public, anon, authenticated;
grant execute on function "player"."list_guild_rank_settings"(bigint) to service_role;
grant execute on function "player"."replace_guild_rank_settings"(bigint, jsonb) to service_role;
grant execute on function "player"."create_guild_atomic"(bigint, text, bigint, jsonb) to service_role;
grant execute on function "player"."contribute_guild_money_atomic"(bigint, bigint, bigint, bigint) to service_role;
grant execute on function "player"."contribute_guild_material_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer, integer, bigint) to service_role;
grant execute on function "player"."develop_guild_skill_atomic"(bigint, bigint, integer, integer, integer, bigint, bigint, integer[]) to service_role;
grant execute on function "player"."set_guild_name"(bigint, bigint, text) to service_role;
grant execute on function "player"."guild_permission_authorized"(bigint, bigint, text) to service_role;
grant execute on function "player"."replace_guild_rank_settings_authorized"(bigint, bigint, jsonb) to service_role;
grant execute on function "player"."set_guild_notice"(bigint, bigint, text, text) to service_role;

create or replace function "player"."guild_warehouse_authorized"(p_guild_id bigint, p_actor_id bigint)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_rank integer;
    v_leader_id bigint;
    v_allowed boolean;
begin
    select "leader_id" into v_leader_id from "player"."guilds" where "id" = p_guild_id for update;
    if not found then
        return false;
    end if;
    select "rank" into v_rank from "player"."guild_members"
    where "guild_id" = p_guild_id and "character_id" = p_actor_id for update;
    if not found then
        return false;
    end if;
    if v_rank = 1 and v_leader_id = p_actor_id then
        return true;
    end if;
    if v_rank not between 2 and 6 then
        return false;
    end if;
    select "can_slot" into v_allowed from "player"."guild_rank_settings"
    where "guild_id" = p_guild_id and "rank" = v_rank for update;
    return coalesce(v_allowed, false);
end;
$function$;

create or replace function "player"."move_guild_warehouse_atomic"(
    p_guild_id bigint, p_actor_id bigint, p_source_id bigint, p_source_index integer,
    p_target_index integer, p_template_id integer, p_max_stack integer, p_page_count integer
)
returns table(source_id bigint, target_id bigint, source_count integer, target_count integer, mode text)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_source "player"."guild_warehouse"%rowtype;
    v_target "player"."guild_warehouse"%rowtype;
    v_page_count integer;
    v_combined bigint;
begin
    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found or not "player"."guild_warehouse_authorized"(p_guild_id, p_actor_id) then
        raise exception 'GUILD_WAREHOUSE_UNAUTHORIZED';
    end if;
    v_page_count := coalesce(("player"."load_latest_guild_log_details"(p_guild_id, 9002)->>'pageCount')::integer, 1);
    if v_page_count <> p_page_count or v_page_count not between 1 and 4 or p_source_index < 0 or p_target_index < 0
       or p_source_index >= v_page_count * 30 or p_target_index >= v_page_count * 30
       or p_source_index = p_target_index or p_max_stack <= 0 then
        raise exception 'INVALID_GUILD_WAREHOUSE_MOVE';
    end if;
    perform 1 from "player"."guild_warehouse"
    where "guild_id" = p_guild_id and "slot_index" in (p_source_index, p_target_index)
    order by "id" for update;
    select * into v_source from "player"."guild_warehouse"
    where "guild_id" = p_guild_id and "id" = p_source_id and "slot_index" = p_source_index;
    if not found or v_source."template_id" <> p_template_id or v_source."stack_count" <= 0 or v_source."stack_count" > p_max_stack then
        raise exception 'INVALID_GUILD_WAREHOUSE_MOVE';
    end if;
    select * into v_target from "player"."guild_warehouse"
    where "guild_id" = p_guild_id and "slot_index" = p_target_index;
    if not found then
        update "player"."guild_warehouse" set "slot_index" = p_target_index where "id" = v_source."id";
        return query select v_source."id", 0::bigint, v_source."stack_count", 0, 'move'::text;
        return;
    end if;
    if v_target."template_id" <> v_source."template_id" or v_target."stack_count" <= 0 or v_target."stack_count" >= p_max_stack then
        raise exception 'INVALID_GUILD_WAREHOUSE_MOVE';
    end if;
    v_combined := v_source."stack_count"::bigint + v_target."stack_count";
    if v_combined <= p_max_stack then
        update "player"."guild_warehouse" set "stack_count" = v_combined::integer where "id" = v_target."id";
        delete from "player"."guild_warehouse" where "id" = v_source."id";
        return query select v_source."id", v_target."id", 0, v_combined::integer, 'merge'::text;
        return;
    end if;
    update "player"."guild_warehouse" set "stack_count" = p_max_stack where "id" = v_target."id";
    update "player"."guild_warehouse" set "stack_count" = (v_combined - p_max_stack)::integer where "id" = v_source."id";
    return query select v_source."id", v_target."id", (v_combined - p_max_stack)::integer, p_max_stack, 'partial'::text;
end;
$function$;

create or replace function "player"."drop_guild_warehouse_atomic"(p_guild_id bigint, p_actor_id bigint, p_slot_id bigint)
returns table(template_id integer, stack_count integer)
language plpgsql
security invoker
set search_path to ''
as $function$
begin
    perform 1 from "player"."guilds" where "id" = p_guild_id for update;
    if not found or not "player"."guild_warehouse_authorized"(p_guild_id, p_actor_id) then
        raise exception 'GUILD_WAREHOUSE_UNAUTHORIZED';
    end if;
    return query delete from "player"."guild_warehouse" as w
    where w."guild_id" = p_guild_id and w."id" = p_slot_id
    returning w."template_id", w."stack_count";
end;
$function$;

revoke execute on function "player"."guild_warehouse_authorized"(bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."move_guild_warehouse_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer) from public, anon, authenticated;
revoke execute on function "player"."drop_guild_warehouse_atomic"(bigint, bigint, bigint) from public, anon, authenticated;
grant execute on function "player"."guild_warehouse_authorized"(bigint, bigint) to service_role;
grant execute on function "player"."move_guild_warehouse_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, integer) to service_role;
grant execute on function "player"."drop_guild_warehouse_atomic"(bigint, bigint, bigint) to service_role;

create table if not exists "player"."guild_buildings" (
    "id" bigint generated by default as identity primary key,
    "guild_id" bigint not null references "player"."guilds"("id") on delete cascade,
    "extend_id" integer not null,
    "template_id" integer,
    "pending_template_id" integer,
    "mode" integer,
    "actor_id" bigint references "player"."characters"("id") on delete set null,
    "revision" bigint not null default 1,
    "created_at" timestamp with time zone not null default now(),
    "updated_at" timestamp with time zone not null default now(),
    constraint "guild_buildings_guild_extend_key" unique ("guild_id", "extend_id"),
    constraint "guild_buildings_state_check" check (
        ("template_id" is not null or "pending_template_id" is not null)
        and (("pending_template_id" is null and "mode" is null) or ("pending_template_id" is not null and "mode" in (1, 2)))
    )
);

alter table "player"."guild_buildings" enable row level security;
revoke all on table "player"."guild_buildings" from public, anon, authenticated;
grant select, insert, update, delete on table "player"."guild_buildings" to service_role;
grant usage, select on sequence "player"."guild_buildings_id_seq" to service_role;

create or replace function "player"."list_guild_buildings"(p_guild_id bigint)
returns table(id bigint, guild_id bigint, extend_id integer, template_id integer, pending_template_id integer, mode integer, actor_id bigint, revision bigint, created_at timestamp with time zone, updated_at timestamp with time zone)
language sql
security invoker
set search_path to ''
as $function$
    select b."id", b."guild_id", b."extend_id", coalesce(b."template_id", 0), coalesce(b."pending_template_id", 0), coalesce(b."mode", 0), coalesce(b."actor_id", 0), b."revision", b."created_at", b."updated_at"
    from "player"."guild_buildings" as b
    where b."guild_id" = p_guild_id
    order by b."extend_id";
$function$;

create or replace function "player"."start_guild_building_atomic"(
    p_guild_id bigint, p_actor_id bigint, p_source_id bigint, p_extend_id integer,
    p_target_template_id integer, p_current_template_id integer, p_mode integer,
    p_expected_revision bigint, p_prerequisite_ids integer[], p_family_ids integer[], p_num_limit integer
)
returns table(id bigint, guild_id bigint, extend_id integer, template_id integer, pending_template_id integer, mode integer, actor_id bigint, revision bigint, created_at timestamp with time zone, updated_at timestamp with time zone)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_building "player"."guild_buildings"%rowtype;
    v_required integer;
    v_count integer;
begin
    perform 1 from "player"."guilds" as g
    where g."id" = p_guild_id and g."leader_id" = p_actor_id for update;
    if not found or not exists (
        select 1 from "player"."guild_members" as m
        where m."guild_id" = p_guild_id and m."character_id" = p_actor_id and m."rank" = 1
    ) or p_mode not in (1, 2) or p_target_template_id <= 0 or p_extend_id <= 0
       or p_num_limit <= 0 or p_family_ids is null or cardinality(p_family_ids) = 0
       or (select count(distinct family_id) from unnest(p_family_ids) as family(family_id)) <> cardinality(p_family_ids)
       or not (p_target_template_id = any(p_family_ids)) then
        raise exception 'INVALID_GUILD_BUILDING_START';
    end if;
    perform 1 from "player"."guild_buildings" as b where b."guild_id" = p_guild_id order by b."id" for update;
    foreach v_required in array coalesce(p_prerequisite_ids, array[]::integer[]) loop
        if v_required not in (1, 10) and not exists (
            select 1 from "player"."guild_buildings" as b
            where b."guild_id" = p_guild_id and b."template_id" = v_required
        ) then
            raise exception 'GUILD_BUILDING_PREREQUISITE';
        end if;
    end loop;
    select count(*) into v_count from "player"."guild_buildings" as b
    where b."guild_id" = p_guild_id and b."extend_id" <> p_extend_id
      and (b."template_id" = any(p_family_ids) or b."pending_template_id" = any(p_family_ids));
    if v_count >= p_num_limit then
        raise exception 'GUILD_BUILDING_LIMIT';
    end if;
    if p_mode = 2 then
        if p_source_id <> p_extend_id or p_current_template_id <> 0 or p_expected_revision <> 0
           or exists (select 1 from "player"."guild_buildings" as b where b."guild_id" = p_guild_id and b."extend_id" = p_extend_id) then
            raise exception 'INVALID_GUILD_BUILDING_CREATE';
        end if;
        insert into "player"."guild_buildings" ("guild_id", "extend_id", "pending_template_id", "mode", "actor_id")
        values (p_guild_id, p_extend_id, p_target_template_id, p_mode, p_actor_id)
        returning * into v_building;
    else
        select * into v_building from "player"."guild_buildings" as b
        where b."guild_id" = p_guild_id and b."id" = p_source_id and b."extend_id" = p_extend_id for update;
        if not found or v_building."template_id" <> p_current_template_id or v_building."pending_template_id" is not null
           or v_building."revision" <> p_expected_revision then
            raise exception 'INVALID_GUILD_BUILDING_UPGRADE';
        end if;
        update "player"."guild_buildings" as b
        set "pending_template_id" = p_target_template_id, "mode" = p_mode, "actor_id" = p_actor_id,
            "revision" = b."revision" + 1, "updated_at" = now()
        where b."id" = v_building."id" returning * into v_building;
    end if;
    return query select v_building."id", v_building."guild_id", v_building."extend_id", coalesce(v_building."template_id", 0), coalesce(v_building."pending_template_id", 0), coalesce(v_building."mode", 0), coalesce(v_building."actor_id", 0), v_building."revision", v_building."created_at", v_building."updated_at";
end;
$function$;

create or replace function "player"."finish_guild_building_atomic"(
    p_guild_id bigint, p_actor_id bigint, p_building_id bigint, p_target_template_id integer,
    p_expected_revision bigint, p_money_cost bigint, p_experience_cost bigint,
    p_general_cost integer, p_rare_cost integer, p_special_cost integer
)
returns table(id bigint, guild_id bigint, extend_id integer, template_id integer, pending_template_id integer, mode integer, actor_id bigint, revision bigint, created_at timestamp with time zone, updated_at timestamp with time zone)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_guild "player"."guilds"%rowtype;
    v_building "player"."guild_buildings"%rowtype;
    v_material_ids integer[] := array[2026, 2027, 2039];
    v_material_costs integer[] := array[p_general_cost, p_rare_cost, p_special_cost];
    v_index integer;
    v_remaining integer;
    v_slot "player"."guild_warehouse"%rowtype;
    v_take integer;
begin
    if p_money_cost < 0 or p_experience_cost < 0 or p_general_cost < 0 or p_rare_cost < 0 or p_special_cost < 0 then
        raise exception 'INVALID_GUILD_BUILDING_COST';
    end if;
    select * into v_guild from "player"."guilds" as g
    where g."id" = p_guild_id and g."leader_id" = p_actor_id for update;
    if not found or not exists (
        select 1 from "player"."guild_members" as m
        where m."guild_id" = p_guild_id and m."character_id" = p_actor_id and m."rank" = 1
    ) then
        raise exception 'GUILD_BUILDING_UNAUTHORIZED';
    end if;
    select * into v_building from "player"."guild_buildings" as b
    where b."guild_id" = p_guild_id and b."id" = p_building_id for update;
    if not found or v_building."pending_template_id" <> p_target_template_id or v_building."revision" <> p_expected_revision then
        raise exception 'INVALID_GUILD_BUILDING_FINISH';
    end if;
    if v_guild."funds" < p_money_cost or v_guild."experience" < p_experience_cost then
        raise exception 'INSUFFICIENT_GUILD_BUILDING_RESOURCES';
    end if;
    for v_index in 1..3 loop
        v_remaining := v_material_costs[v_index];
        if coalesce((select sum(w."stack_count") from "player"."guild_warehouse" as w where w."guild_id" = p_guild_id and w."template_id" = v_material_ids[v_index]), 0) < v_remaining then
            raise exception 'INSUFFICIENT_GUILD_BUILDING_MATERIALS';
        end if;
        for v_slot in select * from "player"."guild_warehouse" as w
            where w."guild_id" = p_guild_id and w."template_id" = v_material_ids[v_index]
            order by w."id" for update
        loop
            exit when v_remaining = 0;
            v_take := least(v_remaining, v_slot."stack_count");
            if v_take = v_slot."stack_count" then
                delete from "player"."guild_warehouse" as w where w."id" = v_slot."id";
            else
                update "player"."guild_warehouse" as w set "stack_count" = w."stack_count" - v_take where w."id" = v_slot."id";
            end if;
            v_remaining := v_remaining - v_take;
        end loop;
    end loop;
    update "player"."guilds" as g
    set "funds" = g."funds" - p_money_cost, "experience" = g."experience" - p_experience_cost
    where g."id" = p_guild_id;
    update "player"."guild_buildings" as b
    set "template_id" = p_target_template_id, "pending_template_id" = null, "mode" = null, "actor_id" = null,
        "revision" = b."revision" + 1, "updated_at" = now()
    where b."id" = p_building_id returning * into v_building;
    return query select v_building."id", v_building."guild_id", v_building."extend_id", coalesce(v_building."template_id", 0), 0, 0, 0::bigint, v_building."revision", v_building."created_at", v_building."updated_at";
end;
$function$;

create or replace function "player"."reset_guild_building_atomic"(p_guild_id bigint, p_actor_id bigint, p_building_id bigint)
returns table(id bigint, guild_id bigint, extend_id integer, template_id integer, pending_template_id integer, mode integer, actor_id bigint, revision bigint, created_at timestamp with time zone, updated_at timestamp with time zone)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_building "player"."guild_buildings"%rowtype;
begin
    perform 1 from "player"."guilds" as g where g."id" = p_guild_id and g."leader_id" = p_actor_id for update;
    if not found or not exists (
        select 1 from "player"."guild_members" as m
        where m."guild_id" = p_guild_id and m."character_id" = p_actor_id and m."rank" = 1
    ) then
        raise exception 'GUILD_BUILDING_UNAUTHORIZED';
    end if;
    select * into v_building from "player"."guild_buildings" as b
    where b."guild_id" = p_guild_id and b."id" = p_building_id for update;
    if not found or v_building."pending_template_id" is null then
        raise exception 'INVALID_GUILD_BUILDING_RESET';
    end if;
    if v_building."template_id" is null then
        delete from "player"."guild_buildings" as b where b."id" = v_building."id";
    else
        update "player"."guild_buildings" as b
        set "pending_template_id" = null, "mode" = null, "actor_id" = null,
            "revision" = b."revision" + 1, "updated_at" = now()
        where b."id" = v_building."id" returning * into v_building;
    end if;
    return query select v_building."id", v_building."guild_id", v_building."extend_id", coalesce(v_building."template_id", 0), 0, 0, 0::bigint, v_building."revision", v_building."created_at", v_building."updated_at";
end;
$function$;

revoke execute on function "player"."list_guild_buildings"(bigint) from public, anon, authenticated;
revoke execute on function "player"."start_guild_building_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, bigint, integer[], integer[], integer) from public, anon, authenticated;
revoke execute on function "player"."finish_guild_building_atomic"(bigint, bigint, bigint, integer, bigint, bigint, bigint, integer, integer, integer) from public, anon, authenticated;
revoke execute on function "player"."reset_guild_building_atomic"(bigint, bigint, bigint) from public, anon, authenticated;
grant execute on function "player"."list_guild_buildings"(bigint) to service_role;
grant execute on function "player"."start_guild_building_atomic"(bigint, bigint, bigint, integer, integer, integer, integer, bigint, integer[], integer[], integer) to service_role;
grant execute on function "player"."finish_guild_building_atomic"(bigint, bigint, bigint, integer, bigint, bigint, bigint, integer, integer, integer) to service_role;
grant execute on function "player"."reset_guild_building_atomic"(bigint, bigint, bigint) to service_role;

create or replace function "player"."guild_application_state_json"(p_application_id bigint)
returns jsonb
language sql
security invoker
set search_path to ''
as $function$
    select jsonb_build_object(
        'id', a."id", 'guild_id', a."guild_id", 'character_id', a."character_id",
        'character_name', c."name", 'character_class', c."class_id", 'character_level', p."level", 'character_exp', p."experience",
        'message', a."message", 'status', a."status", 'created_at', a."created_at", 'processed_at', a."processed_at", 'processed_by', a."processed_by"
    )
    from "player"."guild_applications" as a
    join "player"."characters" as c on c."id" = a."character_id"
    join "player"."character_progression" as p on p."character_id" = c."id"
    where a."id" = p_application_id;
$function$;

create or replace function "player"."guild_member_state_json"(p_member_id bigint)
returns jsonb
language sql
security invoker
set search_path to ''
as $function$
    select jsonb_build_object(
        'id', m."id", 'guild_id', m."guild_id", 'character_id', m."character_id",
        'character_name', c."name", 'character_level', p."level", 'character_class', c."class_id", 'character_exp', p."experience",
        'rank', m."rank", 'duty', m."duty", 'contribution_normal', m."contribution_normal",
        'contribution_donate', m."contribution_donate", 'contribution_total', m."contribution_total",
        'contribution_weekly', m."contribution_weekly", 'joined_at', m."joined_at", 'last_online', m."last_online"
    )
    from "player"."guild_members" as m
    join "player"."characters" as c on c."id" = m."character_id"
    join "player"."character_progression" as p on p."character_id" = c."id"
    where m."id" = p_member_id;
$function$;

create or replace function "player"."guild_character_state_json"(p_character_id bigint)
returns jsonb
language sql
security invoker
set search_path to ''
as $function$
    select jsonb_build_object(
        'character_id', c."id", 'guild_id', c."guild_id", 'money', w."money", 'money_bind', w."money_bind", 'gold', w."gold",
        'guild_contrib', coalesce((select cc."amount" from "player"."character_currencies" as cc where cc."character_id" = c."id" and cc."currency_type" = 20 and cc."season" = 0), 0),
        'donate_contrib', coalesce((select cc."amount" from "player"."character_currencies" as cc where cc."character_id" = c."id" and cc."currency_type" = 21 and cc."season" = 0), 0),
        'guild_restore_normal', c."guild_restore_contrib", 'guild_restore_donate', c."guild_restore_donate"
    )
    from "player"."characters" as c
    join "player"."character_wallet" as w on w."character_id" = c."id"
    where c."id" = p_character_id;
$function$;

create or replace function "player"."apply_guild_atomic"(p_guild_id bigint, p_character_id bigint)
returns jsonb
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_application_id bigint;
    v_level integer;
    v_join_level integer;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_character_id::text, 0));
    select g."join_level_req" into v_join_level from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found then raise exception 'GUILD_NOT_FOUND'; end if;
    select p."level" into v_level from "player"."characters" as c
    join "player"."character_progression" as p on p."character_id" = c."id"
    where c."id" = p_character_id and c."guild_id" is null for update of c, p;
    if not found or v_level < v_join_level
       or exists (select 1 from "player"."guild_members" as m where m."character_id" = p_character_id)
       or exists (select 1 from "player"."guild_applications" as a where a."character_id" = p_character_id) then
        raise exception 'INVALID_GUILD_APPLICATION';
    end if;
    insert into "player"."guild_applications" ("guild_id", "character_id", "message", "status")
    values (p_guild_id, p_character_id, '', 0) returning "id" into v_application_id;
    return "player"."guild_application_state_json"(v_application_id);
end;
$function$;

create or replace function "player"."approve_guild_application_atomic"(p_application_id bigint, p_approver_id bigint)
returns jsonb
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_application "player"."guild_applications"%rowtype;
    v_application_json jsonb;
    v_guild "player"."guilds"%rowtype;
    v_member_id bigint;
    v_normal bigint;
    v_donate bigint;
    v_rows integer;
begin
    select * into v_application from "player"."guild_applications" as a where a."id" = p_application_id;
    if not found then raise exception 'GUILD_APPLICATION_NOT_FOUND'; end if;
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || v_application."character_id"::text, 0));
    select * into v_guild from "player"."guilds" as g where g."id" = v_application."guild_id" for update;
    select * into v_application from "player"."guild_applications" as a where a."id" = p_application_id for update;
    if not found or not "player"."guild_permission_authorized"(v_guild."id", p_approver_id, 'add') then raise exception 'GUILD_UNAUTHORIZED'; end if;
    if v_guild."population" >= v_guild."max_population" then raise exception 'GUILD_FULL'; end if;
    if v_guild."population" <> (select count(*) from "player"."guild_members" as m where m."guild_id" = v_guild."id") then raise exception 'MEMBERSHIP_DRIFT'; end if;
    perform 1 from "player"."characters" as c where c."id" = v_application."character_id" and c."guild_id" is null for update;
    if not found or exists (select 1 from "player"."guild_members" as m where m."character_id" = v_application."character_id") then
        raise exception 'ALREADY_IN_GUILD';
    end if;
    perform 1 from "player"."character_wallet" as w where w."character_id" = v_application."character_id" for update;
    if not found then raise exception 'CHARACTER_WALLET_NOT_FOUND'; end if;
    v_application_json := "player"."guild_application_state_json"(p_application_id);
    if v_application_json is null then raise exception 'INVALID_GUILD_APPLICATION_STATE'; end if;
    select c."guild_restore_contrib", c."guild_restore_donate" into v_normal, v_donate
    from "player"."characters" as c where c."id" = v_application."character_id";
    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (v_application."character_id", 20, v_normal, 0), (v_application."character_id", 21, v_donate, 0)
    on conflict ("character_id", "currency_type", "season") do update set "amount" = "player"."character_currencies"."amount" + excluded."amount";
    select coalesce((select cc."amount" from "player"."character_currencies" as cc where cc."character_id" = v_application."character_id" and cc."currency_type" = 20 and cc."season" = 0), 0),
           coalesce((select cc."amount" from "player"."character_currencies" as cc where cc."character_id" = v_application."character_id" and cc."currency_type" = 21 and cc."season" = 0), 0)
    into v_normal, v_donate;
    update "player"."characters" set "guild_id" = v_guild."id", "guild_restore_contrib" = 0, "guild_restore_donate" = 0 where "id" = v_application."character_id";
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    insert into "player"."guild_members" ("guild_id", "character_id", "rank", "contribution_normal", "contribution_donate", "contribution_total")
    values (v_guild."id", v_application."character_id", 6, v_normal, v_donate, v_normal + v_donate) returning "id" into v_member_id;
    update "player"."guilds" set "population" = "population" + 1 where "id" = v_guild."id";
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    delete from "player"."guild_applications" where "character_id" = v_application."character_id";
    get diagnostics v_rows = row_count;
    if v_rows < 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    return jsonb_build_object('application', v_application_json, 'member', "player"."guild_member_state_json"(v_member_id), 'character', "player"."guild_character_state_json"(v_application."character_id"));
end;
$function$;

create or replace function "player"."refuse_guild_application_atomic"(p_application_id bigint, p_approver_id bigint)
returns jsonb
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_application "player"."guild_applications"%rowtype;
    v_result jsonb;
begin
    select * into v_application from "player"."guild_applications" as a where a."id" = p_application_id;
    if not found then raise exception 'GUILD_APPLICATION_NOT_FOUND'; end if;
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || v_application."character_id"::text, 0));
    perform 1 from "player"."guilds" as g where g."id" = v_application."guild_id" for update;
    select * into v_application from "player"."guild_applications" as a where a."id" = p_application_id for update;
    if not found or not "player"."guild_permission_authorized"(v_application."guild_id", p_approver_id, 'add') then
        raise exception 'GUILD_APPLICATION_REJECTED';
    end if;
    v_result := "player"."guild_application_state_json"(p_application_id);
    delete from "player"."guild_applications" where "id" = p_application_id;
    return v_result;
end;
$function$;

create or replace function "player"."cancel_guild_application_atomic"(p_application_id bigint, p_character_id bigint)
returns jsonb
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_result jsonb;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_character_id::text, 0));
    perform 1 from "player"."guild_applications" as a where a."id" = p_application_id and a."character_id" = p_character_id for update;
    if not found then raise exception 'GUILD_APPLICATION_NOT_FOUND'; end if;
    v_result := "player"."guild_application_state_json"(p_application_id);
    delete from "player"."guild_applications" where "id" = p_application_id and "character_id" = p_character_id;
    return v_result;
end;
$function$;

create or replace function "player"."exit_guild_member_atomic"(p_guild_id bigint, p_actor_id bigint, p_target_id bigint, p_kick boolean)
returns jsonb
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_guild "player"."guilds"%rowtype;
    v_member "player"."guild_members"%rowtype;
    v_member_json jsonb;
    v_normal bigint;
    v_donate bigint;
    v_restore_normal bigint;
    v_restore_donate bigint;
    v_rows integer;
begin
    perform pg_advisory_xact_lock(hashtextextended('guild-character:' || p_target_id::text, 0));
    select * into v_guild from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found then raise exception 'GUILD_NOT_FOUND'; end if;
    if p_target_id = v_guild."leader_id" then raise exception 'CANNOT_KICK_LEADER'; end if;
    if (p_kick and not "player"."guild_permission_authorized"(p_guild_id, p_actor_id, 'del'))
       or (not p_kick and p_actor_id <> p_target_id) then
        raise exception 'GUILD_EXIT_UNAUTHORIZED';
    end if;
    select * into v_member from "player"."guild_members" as m
    where m."guild_id" = p_guild_id and m."character_id" = p_target_id for update;
    if not found or v_guild."population" <> (select count(*) from "player"."guild_members" as m where m."guild_id" = p_guild_id) then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;
    perform 1 from "player"."characters" as c where c."id" = p_target_id and c."guild_id" = p_guild_id for update;
    if not found then raise exception 'MEMBERSHIP_DRIFT'; end if;
    perform 1 from "player"."character_wallet" as w where w."character_id" = p_target_id for update;
    if not found then raise exception 'CHARACTER_WALLET_NOT_FOUND'; end if;
    insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
    values (p_target_id, 20, 0, 0), (p_target_id, 21, 0, 0)
    on conflict ("character_id", "currency_type", "season") do nothing;
    select cc."amount" into v_normal from "player"."character_currencies" as cc where cc."character_id" = p_target_id and cc."currency_type" = 20 and cc."season" = 0 for update;
    select cc."amount" into v_donate from "player"."character_currencies" as cc where cc."character_id" = p_target_id and cc."currency_type" = 21 and cc."season" = 0 for update;
    v_restore_normal := v_normal * 9 / 10;
    v_restore_donate := v_donate * 9 / 10;
    if v_restore_normal > 2147483647 or v_restore_donate > 2147483647 then
        raise exception 'GUILD_RESTORE_OVERFLOW';
    end if;
    v_member_json := "player"."guild_member_state_json"(v_member."id");
    if v_member_json is null then raise exception 'INVALID_GUILD_MEMBER_STATE'; end if;
    update "player"."character_currencies" set "amount" = 0 where "character_id" = p_target_id and "currency_type" in (20, 21) and "season" = 0;
    get diagnostics v_rows = row_count;
    if v_rows <> 2 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    update "player"."characters" set "guild_id" = null, "guild_restore_contrib" = v_restore_normal, "guild_restore_donate" = v_restore_donate where "id" = p_target_id;
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    delete from "player"."guild_members" where "id" = v_member."id";
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    update "player"."guilds" set "population" = "population" - 1 where "id" = p_guild_id and "population" > 0;
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    return jsonb_build_object('member', v_member_json, 'character', "player"."guild_character_state_json"(p_target_id));
end;
$function$;

create or replace function "player"."transfer_guild_leadership_atomic"(p_guild_id bigint, p_old_leader_id bigint, p_new_leader_id bigint)
returns table(guild_id bigint, old_leader_id bigint, new_leader_id bigint, new_leader_name text)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_name text;
    v_rows integer;
begin
    perform 1 from "player"."guilds" as g where g."id" = p_guild_id and g."leader_id" = p_old_leader_id for update;
    if not found or p_old_leader_id = p_new_leader_id then raise exception 'INVALID_GUILD_LEADERSHIP_TRANSFER'; end if;
    perform 1 from "player"."guild_members" as m where m."guild_id" = p_guild_id and m."character_id" in (p_old_leader_id, p_new_leader_id) order by m."character_id" for update;
    if (select count(*) from "player"."guild_members" as m where m."guild_id" = p_guild_id and m."character_id" in (p_old_leader_id, p_new_leader_id)) <> 2 then
        raise exception 'INVALID_GUILD_LEADERSHIP_TRANSFER';
    end if;
    update "player"."guild_members" as m
    set "rank" = case when m."character_id" = p_new_leader_id then 1 else 6 end
    where m."guild_id" = p_guild_id and m."character_id" in (p_old_leader_id, p_new_leader_id);
    get diagnostics v_rows = row_count;
    if v_rows <> 2 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    update "player"."guilds" as g set "leader_id" = p_new_leader_id where g."id" = p_guild_id and g."leader_id" = p_old_leader_id;
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    select c."name" into v_name from "player"."characters" as c where c."id" = p_new_leader_id;
    if not found then raise exception 'MEMBERSHIP_DRIFT'; end if;
    return query select p_guild_id, p_old_leader_id, p_new_leader_id, v_name;
end;
$function$;

create or replace function "player"."set_guild_member_rank_atomic"(p_guild_id bigint, p_actor_id bigint, p_target_id bigint, p_rank integer)
returns boolean
language plpgsql
security invoker
set search_path to ''
as $function$
begin
    perform 1 from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found or p_rank not between 2 and 6 or not "player"."guild_permission_authorized"(p_guild_id, p_actor_id, 'duty') then
        return false;
    end if;
    update "player"."guild_members" set "rank" = p_rank
    where "guild_id" = p_guild_id and "character_id" = p_target_id and "rank" <> 1;
    return found;
end;
$function$;

revoke execute on function "player"."guild_application_state_json"(bigint) from public, anon, authenticated;
revoke execute on function "player"."guild_member_state_json"(bigint) from public, anon, authenticated;
revoke execute on function "player"."guild_character_state_json"(bigint) from public, anon, authenticated;
revoke execute on function "player"."apply_guild_atomic"(bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."approve_guild_application_atomic"(bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."refuse_guild_application_atomic"(bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."cancel_guild_application_atomic"(bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."exit_guild_member_atomic"(bigint, bigint, bigint, boolean) from public, anon, authenticated;
revoke execute on function "player"."transfer_guild_leadership_atomic"(bigint, bigint, bigint) from public, anon, authenticated;
revoke execute on function "player"."set_guild_member_rank_atomic"(bigint, bigint, bigint, integer) from public, anon, authenticated;
grant execute on function "player"."guild_application_state_json"(bigint) to service_role;
grant execute on function "player"."guild_member_state_json"(bigint) to service_role;
grant execute on function "player"."guild_character_state_json"(bigint) to service_role;
grant execute on function "player"."apply_guild_atomic"(bigint, bigint) to service_role;
grant execute on function "player"."approve_guild_application_atomic"(bigint, bigint) to service_role;
grant execute on function "player"."refuse_guild_application_atomic"(bigint, bigint) to service_role;
grant execute on function "player"."cancel_guild_application_atomic"(bigint, bigint) to service_role;
grant execute on function "player"."exit_guild_member_atomic"(bigint, bigint, bigint, boolean) to service_role;
grant execute on function "player"."transfer_guild_leadership_atomic"(bigint, bigint, bigint) to service_role;
grant execute on function "player"."set_guild_member_rank_atomic"(bigint, bigint, bigint, integer) to service_role;

