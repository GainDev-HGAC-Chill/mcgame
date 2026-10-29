alter table "player"."marriages" add column if not exists "wedding_line" integer;
alter table "player"."marriages" add column if not exists "wedding_flag" smallint;
alter table "player"."marriages" add column if not exists "wedding_booked_at" timestamp with time zone;

alter table "player"."marriages" drop constraint if exists "marriages_partner1_id_key";
alter table "player"."marriages" drop constraint if exists "marriages_partner2_id_key";

create unique index if not exists "idx_marriages_active_wedding_slot"
on "player"."marriages" ("wedding_line", "wedding_date")
where "divorced_at" is null and "wedding_date" is not null;

create or replace function "player"."create_marriage"(
    p_partner1_id bigint,
    p_partner2_id bigint,
    p_ring_type integer,
    p_intimacy integer,
    p_married_at timestamp with time zone
)
returns table(id bigint, married_at timestamp with time zone)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_marriage_id bigint;
    v_married_at timestamp with time zone;
begin
    if p_partner1_id <= 0 or p_partner2_id <= 0 or p_partner1_id = p_partner2_id or p_ring_type not between 1 and 3 then
        raise exception 'INVALID_MARRIAGE_COMMAND';
    end if;

    perform 1
    from "player"."characters" as c
    where c."id" in (p_partner1_id, p_partner2_id)
    order by c."id"
    for update;

    if (select count(*) from "player"."characters" as c where c."id" in (p_partner1_id, p_partner2_id)) <> 2 then
        raise exception 'CHARACTER_NOT_FOUND';
    end if;

    if exists (
        select 1
        from "player"."marriages" as m
        where m."divorced_at" is null
          and (m."partner1_id" in (p_partner1_id, p_partner2_id)
               or m."partner2_id" in (p_partner1_id, p_partner2_id))
    ) then
        raise exception 'MARRIAGE_ACTIVE';
    end if;

    insert into "player"."marriages" ("partner1_id", "partner2_id", "ring_type", "intimacy", "married_at")
    values (p_partner1_id, p_partner2_id, p_ring_type, p_intimacy, p_married_at)
    returning "marriages"."id", "marriages"."married_at" into v_marriage_id, v_married_at;

    insert into "player"."marriage_audit_log" ("marriage_id", "event", "actor_id", "partner1_id", "partner2_id", "payload")
    values (v_marriage_id, 'married', null, p_partner1_id, p_partner2_id,
            jsonb_build_object('ring_type', p_ring_type, 'intimacy', p_intimacy));

    return query select v_marriage_id, v_married_at;
exception
    when unique_violation then
        raise exception 'MARRIAGE_ACTIVE';
end;
$function$;

create or replace function "player"."divorce_with_card"(p_character_id bigint)
returns table(marriage_id bigint, partner1_id bigint, partner2_id bigint, item_id bigint, remaining integer)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_marriage "player"."marriages"%rowtype;
    v_item "player"."character_items"%rowtype;
    v_remaining integer;
begin
    select m.* into v_marriage
    from "player"."marriages" as m
    where m."divorced_at" is null
      and (m."partner1_id" = p_character_id or m."partner2_id" = p_character_id)
    limit 1;

    if not found then
        raise exception 'NOT_MARRIED';
    end if;

    perform 1
    from "player"."characters" as c
    where c."id" in (v_marriage."partner1_id", v_marriage."partner2_id")
    order by c."id"
    for update;

    select m.* into v_marriage
    from "player"."marriages" as m
    where m."id" = v_marriage."id" and m."divorced_at" is null
    for update;

    if not found then
        raise exception 'NOT_MARRIED';
    end if;

    select i.* into v_item
    from "player"."character_items" as i
    where i."character_id" = p_character_id
      and i."template_id" = 122
      and i."slot_type" = 0
      and i."stack_count" > 0
    order by i."id"
    limit 1
    for update;

    if not found then
        raise exception 'DIVORCE_CARD_REQUIRED';
    end if;

    v_remaining := v_item."stack_count" - 1;
    if v_remaining = 0 then
        delete from "player"."character_items" as i where i."id" = v_item."id";
    else
        update "player"."character_items" as i
        set "stack_count" = v_remaining, "updated_at" = current_timestamp
        where i."id" = v_item."id";
    end if;

    update "player"."marriages" as m
    set "divorced_at" = current_timestamp,
        "wedding_date" = null,
        "wedding_venue" = null,
        "wedding_line" = null,
        "wedding_flag" = null,
        "wedding_booked_at" = null
    where m."id" = v_marriage."id";

    insert into "player"."marriage_audit_log" ("marriage_id", "event", "actor_id", "partner1_id", "partner2_id", "payload")
    values (v_marriage."id", 'divorced', p_character_id, v_marriage."partner1_id", v_marriage."partner2_id",
            jsonb_build_object('card_item_id', v_item."id"));

    return query select v_marriage."id", v_marriage."partner1_id", v_marriage."partner2_id", v_item."id", v_remaining;
end;
$function$;

create or replace function "player"."book_wedding_hall"(
    p_character_id bigint,
    p_wedding_date timestamp with time zone,
    p_line integer,
    p_flag integer,
    p_venue integer,
    p_cost bigint
)
returns table(money bigint)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_marriage "player"."marriages"%rowtype;
    v_expected_cost bigint;
    v_expected_venue integer;
    v_money bigint;
begin
    if p_line <= 0 or p_flag not in (0, 1) or p_cost <= 0 then
        raise exception 'INVALID_WEDDING_COMMAND';
    end if;

    select m.* into v_marriage
    from "player"."marriages" as m
    where m."divorced_at" is null
      and (m."partner1_id" = p_character_id or m."partner2_id" = p_character_id)
    order by m."id"
    limit 1
    for update;

    if not found then
        raise exception 'NOT_MARRIED';
    end if;
    if v_marriage."wedding_date" is not null then
        raise exception 'WEDDING_ALREADY_BOOKED';
    end if;

    v_expected_cost := case v_marriage."ring_type" when 1 then 2990000 when 2 then 3990000 when 3 then 5000000 else null end;
    v_expected_venue := case v_marriage."ring_type" when 1 then 64 when 2 then 63 when 3 then 65 else null end;
    if p_cost is distinct from v_expected_cost or p_venue is distinct from v_expected_venue then
        raise exception 'INVALID_WEDDING_COMMAND';
    end if;
    if (p_wedding_date at time zone 'Asia/Ho_Chi_Minh')::date <> (current_timestamp at time zone 'Asia/Ho_Chi_Minh')::date
       or p_wedding_date < current_timestamp + interval '10 minutes' then
        raise exception 'INVALID_WEDDING_COMMAND';
    end if;

    perform pg_catalog.pg_advisory_xact_lock(77, p_line);
    if exists (
        select 1 from "player"."marriages" as m
        where m."divorced_at" is null
          and m."wedding_line" = p_line
          and m."wedding_date" < p_wedding_date + interval '60 minutes'
          and m."wedding_date" + interval '60 minutes' > p_wedding_date
    ) then
        raise exception 'WEDDING_SLOT_TAKEN';
    end if;

    update "player"."character_wallet" as w
    set "money" = w."money" - p_cost
    where w."character_id" = p_character_id and w."money" >= p_cost
    returning w."money" into v_money;
    if not found then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    update "player"."marriages" as m
    set "wedding_date" = p_wedding_date,
        "wedding_venue" = p_venue,
        "wedding_line" = p_line,
        "wedding_flag" = p_flag,
        "wedding_booked_at" = current_timestamp
    where m."id" = v_marriage."id";

    return query select v_money;
exception
    when unique_violation then
        raise exception 'WEDDING_SLOT_TAKEN';
end;
$function$;

create or replace function "player"."cancel_wedding_hall"(p_character_id bigint)
returns boolean
language sql
security invoker
set search_path to ''
as $function$
    with target as (
        select m."id" as "marriage_id"
        from "player"."marriages" as m
        where m."divorced_at" is null
          and (m."partner1_id" = p_character_id or m."partner2_id" = p_character_id)
        limit 1
    ), updated as (
        update "player"."marriages" as m
        set "wedding_date" = null,
            "wedding_venue" = null,
            "wedding_line" = null,
            "wedding_flag" = null,
            "wedding_booked_at" = null
        from target
        where m."id" = target."marriage_id" and m."divorced_at" is null and m."wedding_date" is not null
        returning 1
    )
    select exists(select 1 from updated);
$function$;

create or replace function "player"."list_wedding_bookings"(p_from timestamp with time zone, p_to timestamp with time zone)
returns table(
    marriage_id bigint,
    partner1_id bigint,
    partner1_name text,
    partner2_id bigint,
    partner2_name text,
    ring_type integer,
    wedding_date timestamp with time zone,
    line integer,
    flag integer
)
language sql
stable
security invoker
set search_path to ''
as $function$
    select m."id",
           case when c1."gender" = 0 then c1."id" when c2."gender" = 0 then c2."id" else c1."id" end,
           case when c1."gender" = 0 then c1."name" when c2."gender" = 0 then c2."name" else c1."name" end,
           case when c1."gender" = 0 then c2."id" when c2."gender" = 0 then c1."id" else c2."id" end,
           case when c1."gender" = 0 then c2."name" when c2."gender" = 0 then c1."name" else c2."name" end,
           m."ring_type", m."wedding_date", m."wedding_line", m."wedding_flag"::integer
    from "player"."marriages" as m
    join "player"."characters" as c1 on c1."id" = m."partner1_id"
    join "player"."characters" as c2 on c2."id" = m."partner2_id"
    where m."divorced_at" is null
      and m."wedding_date" >= p_from
      and m."wedding_date" < p_to
    order by m."wedding_date", m."wedding_line", m."id";
$function$;

create or replace function "player"."get_wedding_booking_by_groom"(p_groom_id bigint)
returns table(
    marriage_id bigint,
    partner1_id bigint,
    partner1_name text,
    partner2_id bigint,
    partner2_name text,
    ring_type integer,
    wedding_date timestamp with time zone,
    line integer,
    flag integer
)
language sql
stable
security invoker
set search_path to ''
as $function$
    select *
    from "player"."list_wedding_bookings"(current_timestamp - interval '60 minutes', current_timestamp + interval '1 day') as b
    where b."partner1_id" = p_groom_id
    limit 1;
$function$;

revoke execute on function "player"."create_marriage"(bigint, bigint, integer, integer, timestamp with time zone) from public, anon, authenticated;
revoke execute on function "player"."divorce_with_card"(bigint) from public, anon, authenticated;
revoke execute on function "player"."book_wedding_hall"(bigint, timestamp with time zone, integer, integer, integer, bigint) from public, anon, authenticated;
revoke execute on function "player"."cancel_wedding_hall"(bigint) from public, anon, authenticated;
revoke execute on function "player"."list_wedding_bookings"(timestamp with time zone, timestamp with time zone) from public, anon, authenticated;
revoke execute on function "player"."get_wedding_booking_by_groom"(bigint) from public, anon, authenticated;
grant execute on function "player"."create_marriage"(bigint, bigint, integer, integer, timestamp with time zone) to service_role;
grant execute on function "player"."divorce_with_card"(bigint) to service_role;
grant execute on function "player"."book_wedding_hall"(bigint, timestamp with time zone, integer, integer, integer, bigint) to service_role;
grant execute on function "player"."cancel_wedding_hall"(bigint) to service_role;
grant execute on function "player"."list_wedding_bookings"(timestamp with time zone, timestamp with time zone) to service_role;
grant execute on function "player"."get_wedding_booking_by_groom"(bigint) to service_role;
