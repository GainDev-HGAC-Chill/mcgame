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

create or replace function "player"."disband_guild_atomic"(p_guild_id bigint, p_leader_id bigint, p_expected_member_ids bigint[])
returns jsonb
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_guild "player"."guilds"%rowtype;
    v_member_ids bigint[];
    v_character_id bigint;
    v_normal bigint;
    v_donate bigint;
    v_restore_normal bigint;
    v_restore_donate bigint;
    v_characters jsonb := '[]'::jsonb;
    v_rows integer;
begin
    if p_guild_id <= 0 or p_leader_id <= 0 or cardinality(p_expected_member_ids) <= 0 then
        raise exception 'INVALID_GUILD_DISBAND';
    end if;
    select * into v_guild from "player"."guilds" as g where g."id" = p_guild_id for update;
    if not found then raise exception 'GUILD_NOT_FOUND'; end if;
    if v_guild."leader_id" <> p_leader_id then raise exception 'GUILD_UNAUTHORIZED'; end if;

    select array_agg(m."character_id" order by m."character_id") into v_member_ids
    from "player"."guild_members" as m where m."guild_id" = p_guild_id;
    if v_member_ids is distinct from (
        select array_agg(expected_id order by expected_id) from unnest(p_expected_member_ids) as expected(expected_id)
    ) or cardinality(v_member_ids) <> v_guild."population" then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;

    foreach v_character_id in array v_member_ids loop
        perform pg_advisory_xact_lock(hashtextextended('guild-character:' || v_character_id::text, 0));
    end loop;
    perform 1 from "player"."guild_members" as m where m."guild_id" = p_guild_id order by m."character_id" for update;
    perform 1 from "player"."characters" as c where c."id" = any(v_member_ids) and c."guild_id" = p_guild_id order by c."id" for update;
    if (select count(*) from "player"."characters" as c where c."id" = any(v_member_ids) and c."guild_id" = p_guild_id) <> cardinality(v_member_ids) then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;
    perform 1 from "player"."character_wallet" as w where w."character_id" = any(v_member_ids) order by w."character_id" for update;
    if (select count(*) from "player"."character_wallet" as w where w."character_id" = any(v_member_ids)) <> cardinality(v_member_ids) then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;

    foreach v_character_id in array v_member_ids loop
        insert into "player"."character_currencies" ("character_id", "currency_type", "amount", "season")
        values (v_character_id, 20, 0, 0), (v_character_id, 21, 0, 0)
        on conflict ("character_id", "currency_type", "season") do nothing;
    end loop;
    perform 1 from "player"."character_currencies" as cc
    where cc."character_id" = any(v_member_ids) and cc."currency_type" in (20, 21) and cc."season" = 0
    order by cc."character_id", cc."currency_type" for update;
    if (select count(*) from "player"."character_currencies" as cc where cc."character_id" = any(v_member_ids) and cc."currency_type" in (20, 21) and cc."season" = 0) <> cardinality(v_member_ids) * 2 then
        raise exception 'MEMBERSHIP_DRIFT';
    end if;

    foreach v_character_id in array v_member_ids loop
        select cc."amount" into v_normal from "player"."character_currencies" as cc
        where cc."character_id" = v_character_id and cc."currency_type" = 20 and cc."season" = 0;
        select cc."amount" into v_donate from "player"."character_currencies" as cc
        where cc."character_id" = v_character_id and cc."currency_type" = 21 and cc."season" = 0;
        v_restore_normal := v_normal * 9 / 10;
        v_restore_donate := v_donate * 9 / 10;
        if v_restore_normal > 2147483647 or v_restore_donate > 2147483647 then
            raise exception 'GUILD_RESTORE_OVERFLOW';
        end if;
        update "player"."character_currencies" set "amount" = 0
        where "character_id" = v_character_id and "currency_type" in (20, 21) and "season" = 0;
        get diagnostics v_rows = row_count;
        if v_rows <> 2 then raise exception 'MEMBERSHIP_DRIFT'; end if;
        update "player"."characters"
        set "guild_id" = null, "guild_restore_contrib" = v_restore_normal, "guild_restore_donate" = v_restore_donate
        where "id" = v_character_id and "guild_id" = p_guild_id;
        get diagnostics v_rows = row_count;
        if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
        v_characters := v_characters || jsonb_build_array("player"."guild_character_state_json"(v_character_id));
    end loop;

    delete from "player"."guilds" where "id" = p_guild_id and "leader_id" = p_leader_id;
    get diagnostics v_rows = row_count;
    if v_rows <> 1 then raise exception 'MEMBERSHIP_DRIFT'; end if;
    return jsonb_build_object('guild_id', p_guild_id, 'characters', v_characters);
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
revoke execute on function "player"."disband_guild_atomic"(bigint, bigint, bigint[]) from public, anon, authenticated;
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
grant execute on function "player"."disband_guild_atomic"(bigint, bigint, bigint[]) to service_role;
