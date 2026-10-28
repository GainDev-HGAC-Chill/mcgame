create table if not exists "player"."character_send_combine_purchases" (
    "character_id" bigint not null references "player"."characters"("id") on delete cascade,
    "activity_id" text not null,
    "event_end_ms" bigint not null,
    "buy_count" integer not null default 0,
    "updated_at" timestamp with time zone not null default now(),
    primary key ("character_id", "activity_id", "event_end_ms"),
    constraint "character_send_combine_purchases_count_check" check ("buy_count" >= 0),
    constraint "character_send_combine_purchases_event_check" check ("event_end_ms" > 0)
);

alter table "player"."character_send_combine_purchases" enable row level security;
revoke all on table "player"."character_send_combine_purchases" from public, anon, authenticated;
grant select, insert, update, delete on table "player"."character_send_combine_purchases" to service_role;

create or replace function "player"."purchase_send_combine_atomic"(
    p_character_id bigint,
    p_activity_id text,
    p_event_end_ms bigint,
    p_limit integer,
    p_cost bigint,
    p_awards jsonb
)
returns table(buy_count integer, gold bigint, items jsonb)
language plpgsql
security invoker
set search_path to ''
as $function$
declare
    v_wallet "player"."character_wallet"%rowtype;
    v_buy_count integer;
    v_award jsonb;
    v_template_id integer;
    v_item_type integer;
    v_count integer;
    v_bound boolean;
    v_color_code integer;
    v_slot_index integer;
    v_item "player"."character_items"%rowtype;
    v_items jsonb := '[]'::jsonb;
begin
    if p_character_id <= 0 or coalesce(p_activity_id, '') = '' or p_event_end_ms <= 0
       or p_limit <= 0 or p_cost <= 0 or jsonb_typeof(p_awards) <> 'array'
       or jsonb_array_length(p_awards) = 0 then
        raise exception 'INVALID_SEND_COMBINE_COMMAND';
    end if;

    if floor(extract(epoch from clock_timestamp()) * 1000)::bigint > p_event_end_ms then
        raise exception 'SEND_COMBINE_EXPIRED';
    end if;

    perform 1 from "player"."characters" where "id" = p_character_id for update;
    if not found then
        raise exception 'CHARACTER_NOT_FOUND';
    end if;

    select * into v_wallet
    from "player"."character_wallet"
    where "character_id" = p_character_id
    for update;
    if not found then
        raise exception 'CHARACTER_NOT_FOUND';
    end if;

    insert into "player"."character_send_combine_purchases" (
        "character_id", "activity_id", "event_end_ms", "buy_count"
    ) values (p_character_id, p_activity_id, p_event_end_ms, 0)
    on conflict do nothing;

    select p."buy_count" into v_buy_count
    from "player"."character_send_combine_purchases" as p
    where p."character_id" = p_character_id
      and p."activity_id" = p_activity_id
      and p."event_end_ms" = p_event_end_ms
    for update;

    if v_buy_count >= p_limit then
        raise exception 'SEND_COMBINE_LIMIT_REACHED';
    end if;
    if v_wallet."gold" < p_cost then
        raise exception 'INSUFFICIENT_FUNDS';
    end if;

    perform 1
    from "player"."character_items"
    where "character_id" = p_character_id and "slot_type" = 0
    order by "id"
    for update;

    for v_award in select value from jsonb_array_elements(p_awards)
    loop
        v_template_id := coalesce((v_award->>'templateId')::integer, 0);
        v_item_type := coalesce((v_award->>'itemType')::integer, 0);
        v_count := coalesce((v_award->>'count')::integer, 0);
        v_bound := coalesce((v_award->>'isBound')::boolean, false);
        v_color_code := coalesce((v_award->>'colorCode')::integer, 0);

        if v_template_id <= 0 or v_item_type not between 1 and 4 or v_count <= 0
           or not exists (
               select 1
               from "data"."data_tbl_item_template" as t
               where t."id" = v_template_id and t."stack_max" >= v_count
           ) then
            raise exception 'INVALID_SEND_COMBINE_AWARD';
        end if;

        select min(candidate.slot_index) into v_slot_index
        from generate_series(0, 209) as candidate(slot_index)
        where not exists (
            select 1
            from "player"."character_items" as occupied
            where occupied."character_id" = p_character_id
              and occupied."slot_type" = 0
              and occupied."slot_index" = candidate.slot_index
        );
        if v_slot_index is null then
            raise exception 'INVENTORY_FULL';
        end if;

        insert into "player"."character_items" (
            "character_id", "template_id", "item_type", "slot_type", "slot_index",
            "stack_count", "is_bound", "color_code", "properties"
        ) values (
            p_character_id, v_template_id, v_item_type, 0, v_slot_index,
            v_count, v_bound, v_color_code, '{}'::jsonb
        ) returning * into v_item;

        v_items := v_items || jsonb_build_array(jsonb_build_object(
            'id', v_item."id", 'characterId', v_item."character_id",
            'templateId', v_item."template_id", 'itemType', v_item."item_type",
            'slotType', v_item."slot_type", 'slotIndex', v_item."slot_index",
            'stackCount', v_item."stack_count", 'isBound', v_item."is_bound",
            'durability', v_item."durability", 'maxDurability', v_item."max_durability",
            'enchantLevel', v_item."enchant_level", 'starLevel', v_item."star_level",
            'colorCode', v_item."color_code", 'properties', v_item."properties",
            'createdAt', v_item."created_at", 'updatedAt', v_item."updated_at"
        ));
    end loop;

    update "player"."character_wallet" as w
    set "gold" = w."gold" - p_cost
    where w."character_id" = p_character_id
    returning w."gold" into v_wallet."gold";

    update "player"."character_send_combine_purchases" as p
    set "buy_count" = p."buy_count" + 1, "updated_at" = now()
    where p."character_id" = p_character_id
      and p."activity_id" = p_activity_id
      and p."event_end_ms" = p_event_end_ms
    returning p."buy_count" into v_buy_count;

    return query select v_buy_count, v_wallet."gold", v_items;
end;
$function$;

revoke all on function "player"."purchase_send_combine_atomic"(bigint, text, bigint, integer, bigint, jsonb) from public, anon, authenticated;
grant execute on function "player"."purchase_send_combine_atomic"(bigint, text, bigint, integer, bigint, jsonb) to service_role;
