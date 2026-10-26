  create table "player"."character_achievement_reward_grants" (
    "character_id" bigint not null,
    "achievement_id" integer not null,
    "claim_token" uuid not null,
    "reward_kind" smallint not null,
    "template_id" integer not null,
    "quantity" integer not null,
    "status" text not null,
    "receipt" jsonb,
    "last_error" text not null default ''::text,
    "lease_expires_at" timestamp with time zone,
    "created_at" timestamp with time zone not null default CURRENT_TIMESTAMP,
    "updated_at" timestamp with time zone not null default CURRENT_TIMESTAMP,
    "granted_at" timestamp with time zone,
    "finalized_at" timestamp with time zone
      );


alter table "player"."character_achievement_reward_grants" enable row level security;
CREATE UNIQUE INDEX character_achievement_reward_grants_claim_token_key ON player.character_achievement_reward_grants USING btree (claim_token);

CREATE UNIQUE INDEX character_achievement_reward_grants_pkey ON player.character_achievement_reward_grants USING btree (character_id, achievement_id);
CREATE UNIQUE INDEX idx_character_items_achievement_reward_key ON player.character_items USING btree (character_id, ((properties ->> 'achievement_reward_key'::text))) WHERE (properties ? 'achievement_reward_key'::text);

alter table "player"."character_achievement_reward_grants" add constraint "character_achievement_reward_grants_pkey" PRIMARY KEY using index "character_achievement_reward_grants_pkey";
alter table "player"."character_achievement_reward_grants" add constraint "character_achievement_reward_grants_character_id_fkey" FOREIGN KEY (character_id) REFERENCES player.characters(id) ON DELETE CASCADE not valid;

alter table "player"."character_achievement_reward_grants" validate constraint "character_achievement_reward_grants_character_id_fkey";

alter table "player"."character_achievement_reward_grants" add constraint "character_achievement_reward_grants_claim_token_key" UNIQUE using index "character_achievement_reward_grants_claim_token_key";

alter table "player"."character_achievement_reward_grants" add constraint "character_achievement_reward_grants_quantity_check" CHECK ((quantity > 0)) not valid;

alter table "player"."character_achievement_reward_grants" validate constraint "character_achievement_reward_grants_quantity_check";

alter table "player"."character_achievement_reward_grants" add constraint "character_achievement_reward_grants_reward_kind_check" CHECK (((reward_kind >= 1) AND (reward_kind <= 3))) not valid;

alter table "player"."character_achievement_reward_grants" validate constraint "character_achievement_reward_grants_reward_kind_check";

alter table "player"."character_achievement_reward_grants" add constraint "character_achievement_reward_grants_status_check" CHECK ((status = ANY (ARRAY['reserved'::text, 'granting'::text, 'granted'::text, 'finalized'::text, 'failed'::text]))) not valid;

alter table "player"."character_achievement_reward_grants" validate constraint "character_achievement_reward_grants_status_check";

alter table "player"."character_achievement_reward_grants" add constraint "character_achievement_reward_grants_template_id_check" CHECK ((template_id > 0)) not valid;

alter table "player"."character_achievement_reward_grants" validate constraint "character_achievement_reward_grants_template_id_check";
set check_function_bodies = off;
CREATE OR REPLACE FUNCTION player.finalize_character_achievement_reward_grant(p_character_id bigint, p_achievement_id bigint, p_finalized_at timestamp with time zone)
 RETURNS SETOF player.character_achievements
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare
    v_status text;
begin
    perform pg_advisory_xact_lock(p_character_id);

    select g.status into v_status
    from player.character_achievement_reward_grants g
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
    for update;

    if v_status = 'finalized' then
        return query
        select a.*
        from player.character_achievements a
        where a.character_id = p_character_id
          and a.achievement_id = p_achievement_id
          and a.is_claimed = true;
        return;
    end if;

    if v_status <> 'granted' then
        return;
    end if;

    return query
    update player.character_achievements a
    set is_claimed = true,
        claimed_at = p_finalized_at
    where a.character_id = p_character_id
      and a.achievement_id = p_achievement_id
      and a.is_completed = true
      and a.is_claimed = false
    returning a.*;

    if found then
        update player.character_achievement_reward_grants g
        set status = 'finalized',
            finalized_at = p_finalized_at,
            updated_at = p_finalized_at
        where g.character_id = p_character_id
          and g.achievement_id = p_achievement_id;
    end if;
end;
$function$
;
CREATE OR REPLACE FUNCTION player.list_granted_character_achievement_rewards(p_character_id bigint, p_limit integer DEFAULT 100)
 RETURNS SETOF player.character_achievement_reward_grants
 LANGUAGE sql
 STABLE
 SET search_path TO ''
AS $function$
    select g.*
    from player.character_achievement_reward_grants g
    where g.character_id = p_character_id
      and g.status = 'granted'
    order by g.granted_at, g.achievement_id
    limit greatest(1, least(coalesce(p_limit, 100), 1000));
$function$
;

CREATE OR REPLACE FUNCTION player.mark_character_achievement_reward_failed(p_character_id bigint, p_achievement_id bigint, p_claim_token uuid, p_reason text, p_failed_at timestamp with time zone)
 RETURNS boolean
 LANGUAGE sql
 SET search_path TO ''
AS $function$
    with updated as (
        update player.character_achievement_reward_grants g
        set status = 'failed',
            last_error = left(coalesce(p_reason, 'reward grant failed'), 1000),
            lease_expires_at = null,
            updated_at = p_failed_at
        where g.character_id = p_character_id
          and g.achievement_id = p_achievement_id
          and g.claim_token = p_claim_token
          and g.status = 'granting'
        returning 1
    )
    select exists(select 1 from updated);
$function$
;

CREATE OR REPLACE FUNCTION player.mark_character_achievement_reward_granted(p_character_id bigint, p_achievement_id bigint, p_claim_token uuid, p_receipt jsonb, p_granted_at timestamp with time zone)
 RETURNS SETOF player.character_achievement_reward_grants
 LANGUAGE sql
 SET search_path TO ''
AS $function$
    update player.character_achievement_reward_grants g
    set status = 'granted',
        receipt = p_receipt,
        lease_expires_at = null,
        updated_at = p_granted_at,
        granted_at = coalesce(g.granted_at, p_granted_at)
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
      and g.claim_token = p_claim_token
      and g.status = 'granting'
    returning g.*;
$function$
;

CREATE OR REPLACE FUNCTION player.prepare_character_achievement_reward_grant(p_character_id bigint, p_achievement_id bigint, p_reward_kind integer, p_template_id integer, p_quantity integer, p_claim_token uuid, p_prepared_at timestamp with time zone)
 RETURNS SETOF player.character_achievement_reward_grants
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare
    v_grant player.character_achievement_reward_grants%rowtype;
begin
    perform pg_advisory_xact_lock(p_character_id);

    select * into v_grant
    from player.character_achievement_reward_grants g
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
    for update;

    if not found then
        if not exists (
            select 1
            from player.character_achievements a
            where a.character_id = p_character_id
              and a.achievement_id = p_achievement_id
              and a.is_completed = true
              and a.is_claimed = false
        ) then
            return;
        end if;

        insert into player.character_achievement_reward_grants (
            character_id, achievement_id, claim_token, reward_kind,
            template_id, quantity, status, created_at, updated_at
        ) values (
            p_character_id, p_achievement_id, p_claim_token, p_reward_kind,
            p_template_id, p_quantity, 'reserved', p_prepared_at, p_prepared_at
        ) returning * into v_grant;
    end if;

    if v_grant.reward_kind <> p_reward_kind
       or v_grant.template_id <> p_template_id
       or v_grant.quantity <> p_quantity then
        raise exception 'achievement reward definition drift for character %, achievement %', p_character_id, p_achievement_id;
    end if;

    if v_grant.status in ('granted', 'finalized') then
        return next v_grant;
        return;
    end if;

    if v_grant.status = 'granting'
       and v_grant.lease_expires_at is not null
       and v_grant.lease_expires_at > p_prepared_at then
        return next v_grant;
        return;
    end if;

    update player.character_achievement_reward_grants g
    set claim_token = p_claim_token,
        status = 'granting',
        last_error = '',
        lease_expires_at = p_prepared_at + interval '5 minutes',
        updated_at = p_prepared_at
    where g.character_id = p_character_id
      and g.achievement_id = p_achievement_id
    returning * into v_grant;

    return next v_grant;
end;
$function$
;
grant insert on table "player"."character_achievement_reward_grants" to "service_role";

grant select on table "player"."character_achievement_reward_grants" to "service_role";

grant update on table "player"."character_achievement_reward_grants" to "service_role";

revoke execute on function player.prepare_character_achievement_reward_grant(bigint, bigint, integer, integer, integer, uuid, timestamptz) from public, anon, authenticated;
grant execute on function player.prepare_character_achievement_reward_grant(bigint, bigint, integer, integer, integer, uuid, timestamptz) to service_role;
revoke execute on function player.mark_character_achievement_reward_granted(bigint, bigint, uuid, jsonb, timestamptz) from public, anon, authenticated;
grant execute on function player.mark_character_achievement_reward_granted(bigint, bigint, uuid, jsonb, timestamptz) to service_role;
revoke execute on function player.mark_character_achievement_reward_failed(bigint, bigint, uuid, text, timestamptz) from public, anon, authenticated;
grant execute on function player.mark_character_achievement_reward_failed(bigint, bigint, uuid, text, timestamptz) to service_role;
revoke execute on function player.finalize_character_achievement_reward_grant(bigint, bigint, timestamptz) from public, anon, authenticated;
grant execute on function player.finalize_character_achievement_reward_grant(bigint, bigint, timestamptz) to service_role;
revoke execute on function player.list_granted_character_achievement_rewards(bigint, integer) from public, anon, authenticated;
grant execute on function player.list_granted_character_achievement_rewards(bigint, integer) to service_role;
