-- System Shop Admin for the existing 10 TBL_SHOP containers.
-- Service-role RPCs, optimistic revision, audit, and atomic mutations.
-- This migration never mutates an existing shop slot by itself.

CREATE TABLE IF NOT EXISTS data.system_shop_admin_audit (
    request_id uuid PRIMARY KEY,
    actor_id uuid NOT NULL,
    sid integer NOT NULL,
    slot_id integer NOT NULL,
    action text NOT NULL CHECK (action IN ('insert', 'update')),
    reason text NOT NULL,
    request_hash text NOT NULL,
    before_image jsonb NOT NULL,
    after_image jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE data.system_shop_admin_audit ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON TABLE data.system_shop_admin_audit FROM PUBLIC, anon, authenticated, service_role;
GRANT SELECT, INSERT ON TABLE data.system_shop_admin_audit TO service_role;

CREATE OR REPLACE FUNCTION public.system_shop_admin_snapshot(p_sid integer)
RETURNS jsonb
LANGUAGE sql STABLE SECURITY INVOKER SET search_path = ''
AS $$
    SELECT jsonb_build_object(
        'shop', jsonb_build_object('id', sh.id, 'name', sh.name, 'type', sh.type),
        'slots', COALESCE((
            SELECT jsonb_agg(
                to_jsonb(sl) || jsonb_build_object(
                    'displayName', CASE WHEN sl.type = 19 THEN eq.name ELSE it.name END,
                    'iconCode', CASE WHEN sl.type = 19 THEN eq.icon_code ELSE it.icon_code END
                ) ORDER BY sl.position, sl.id
            )
            FROM data.data_tbl_shop_slot sl
            LEFT JOIN data.data_tbl_item_template it ON sl.type = 29 AND it.id = sl.item_id
            LEFT JOIN data.data_tbl_equipt_template eq ON sl.type = 19 AND eq.id = sl.item_id
            WHERE sl.sid = sh.id
        ), '[]'::jsonb)
    )
    FROM data.data_tbl_shop sh
    WHERE sh.id = p_sid AND sh.type = 1
      AND sh.id = ANY(ARRAY[35,36,41,65,66,67,70,81,88,89]);
$$;

CREATE OR REPLACE FUNCTION public.system_shop_admin_detail(p_sid integer)
RETURNS jsonb
LANGUAGE sql STABLE SECURITY INVOKER SET search_path = ''
AS $$
    SELECT s || jsonb_build_object('revision', md5(s::text))
    FROM public.system_shop_admin_snapshot(p_sid) s
    WHERE s IS NOT NULL;
$$;

CREATE OR REPLACE FUNCTION public.system_shop_admin_save(
    p_sid integer,
    p_expected_revision text,
    p_action text,
    p_slot_id integer,
    p_slot jsonb,
    p_reason text,
    p_request_id uuid,
    p_actor_id uuid
)
RETURNS jsonb
LANGUAGE plpgsql SECURITY INVOKER SET search_path = ''
AS $$
DECLARE
    v_before jsonb;
    v_after jsonb;
    v_prior data.system_shop_admin_audit%ROWTYPE;
    v_hash text;
    v_slot_id integer;
    v_type integer;
    v_item integer;
    v_gold integer;
    v_money integer;
    v_point integer;
    v_gt integer;
    v_quality integer;
    v_amount integer;
    v_status integer;
    v_position integer;
    v_old data.data_tbl_shop_slot%ROWTYPE;
    v_key text;
BEGIN
    IF p_actor_id IS NULL OR p_request_id IS NULL OR p_reason IS NULL
      OR char_length(btrim(p_reason)) NOT BETWEEN 3 AND 1000
      OR p_expected_revision IS NULL OR p_expected_revision !~ '^[a-f0-9]{32}$'
      OR p_action NOT IN ('insert','update')
      OR p_slot IS NULL OR jsonb_typeof(p_slot) <> 'object'
      OR pg_column_size(p_slot) > 8192 THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Invalid system shop request';
    END IF;
    IF EXISTS (
        SELECT 1 FROM jsonb_object_keys(p_slot) x
        WHERE x NOT IN ('type','item_id','gold','money','point','gt','quality','amount','st','position')
    ) OR (SELECT count(*) FROM jsonb_object_keys(p_slot)) <> 10 THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Unknown/missing shop slot fields';
    END IF;
    -- Reject decimal numbers, strings, nulls, booleans, oversized integers.
    FOR v_key IN SELECT jsonb_object_keys(p_slot) LOOP
        IF jsonb_typeof(p_slot -> v_key) <> 'number'
           OR (p_slot ->> v_key)::numeric <> trunc((p_slot ->> v_key)::numeric)
           OR (p_slot ->> v_key)::numeric NOT BETWEEN -1 AND 2147483647 THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Invalid numeric shop slot field';
        END IF;
    END LOOP;
    v_type := (p_slot->>'type')::integer;
    v_item := (p_slot->>'item_id')::integer;
    v_gold := (p_slot->>'gold')::integer;
    v_money := (p_slot->>'money')::integer;
    v_point := (p_slot->>'point')::integer;
    v_gt := (p_slot->>'gt')::integer;
    v_quality := (p_slot->>'quality')::integer;
    v_amount := (p_slot->>'amount')::integer;
    v_status := (p_slot->>'st')::integer;
    v_position := (p_slot->>'position')::integer;
    IF v_type NOT IN (19,29) OR v_item < 1
      OR v_gold < 0 OR v_money < 0 OR v_point < 0
      OR v_gt NOT IN (0,1) OR v_quality < 0
      OR v_amount = 0 OR v_status NOT IN (0,1,2,4,5)
      OR v_position < 1
      OR ((v_gold > 0)::integer + (v_money > 0)::integer + (v_point > 0)::integer) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Unsupported currency, price, status or quantity';
    END IF;
    IF (v_type = 29 AND NOT EXISTS (SELECT 1 FROM data.data_tbl_item_template WHERE id=v_item))
       OR (v_type = 19 AND NOT EXISTS (SELECT 1 FROM data.data_tbl_equipt_template WHERE id=v_item)) THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Item template not found in chosen table';
    END IF;
    -- Prevent concurrent insertion / lost updates / ID collision.
    LOCK TABLE data.data_tbl_shop_slot IN SHARE ROW EXCLUSIVE MODE;
    v_hash := md5(jsonb_build_array(p_sid,p_expected_revision,p_action,p_slot_id,p_slot,p_reason,p_actor_id)::text);
    SELECT * INTO v_prior FROM data.system_shop_admin_audit WHERE request_id=p_request_id;
    IF FOUND THEN
        IF v_prior.request_hash <> v_hash OR v_prior.actor_id <> p_actor_id THEN
            RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='Request ID conflict';
        END IF;
        RETURN v_prior.after_image || jsonb_build_object('changedSlotId',v_prior.slot_id);
    END IF;
    v_before := public.system_shop_admin_detail(p_sid);
    IF v_before IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='P0002', MESSAGE='System shop not found';
    END IF;
    IF v_before->>'revision' <> p_expected_revision THEN
        RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='Shop revision conflict; reload before saving';
    END IF;
    IF p_action = 'insert' THEN
        IF p_slot_id IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Insert must not supply slot ID';
        END IF;
        SELECT COALESCE(MAX(id),0)+1 INTO v_slot_id FROM data.data_tbl_shop_slot;
        IF EXISTS (
            SELECT 1 FROM data.data_tbl_shop_slot
            WHERE sid=p_sid AND position=v_position AND st<>5 AND v_status<>5
        ) THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Duplicate visible shop position';
        END IF;
        INSERT INTO data.data_tbl_shop_slot(
            id,amount,gold,gt,item_id,money,p_num1,p_num2,p_type1,p_type2,
            point,position,price_all,quality,r,sale,sid,st,type
        ) VALUES (
            v_slot_id,v_amount,v_gold,v_gt,v_item,v_money,0,0,0,0,
            v_point,v_position,0,v_quality,1,0,p_sid,v_status,v_type
        );
    ELSE
        IF p_slot_id IS NULL OR p_slot_id < 1 THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Update requires slot ID';
        END IF;
        SELECT * INTO v_old FROM data.data_tbl_shop_slot
        WHERE id=p_slot_id AND sid=p_sid FOR UPDATE;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE='P0002', MESSAGE='Shop slot not found';
        END IF;
        IF v_position <> v_old.position THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Existing slot position is immutable in Flash';
        END IF;
        IF COALESCE(v_old.p_type1,0) <> 0 OR COALESCE(v_old.p_type2,0) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Multi-currency legacy slots are not editable here';
        END IF;
        IF EXISTS (
            SELECT 1 FROM data.data_tbl_shop_slot
            WHERE sid=p_sid AND position=v_position AND id<>p_slot_id AND st<>5 AND v_status<>5
        ) AND v_old.st = 5 THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='Cannot unhide into occupied position';
        END IF;
        v_slot_id := p_slot_id;
        UPDATE data.data_tbl_shop_slot SET
            amount=v_amount,gold=v_gold,gt=v_gt,item_id=v_item,money=v_money,
            point=v_point,quality=v_quality,st=v_status,type=v_type,
            -- Synchronize conventional bulk price only if it previously matched gold.
            price_all=CASE WHEN v_old.price_all = v_old.gold THEN v_gold ELSE v_old.price_all END
        WHERE id=v_slot_id AND sid=p_sid;
    END IF;
    v_after := public.system_shop_admin_detail(p_sid);
    IF (v_after->>'revision') = (v_before->>'revision') THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='No data changed';
    END IF;
    INSERT INTO data.system_shop_admin_audit(
        request_id,actor_id,sid,slot_id,action,reason,request_hash,before_image,after_image
    ) VALUES (
        p_request_id,p_actor_id,p_sid,v_slot_id,p_action,btrim(p_reason),v_hash,v_before,v_after
    );
    RETURN v_after || jsonb_build_object('changedSlotId',v_slot_id);
END;
$$;

REVOKE ALL ON FUNCTION public.system_shop_admin_snapshot(integer),
    public.system_shop_admin_detail(integer),
    public.system_shop_admin_save(integer,text,text,integer,jsonb,text,uuid,uuid)
FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.system_shop_admin_snapshot(integer),
    public.system_shop_admin_detail(integer),
    public.system_shop_admin_save(integer,text,text,integer,jsonb,text,uuid,uuid)
TO service_role;
NOTIFY pgrst, 'reload schema';
