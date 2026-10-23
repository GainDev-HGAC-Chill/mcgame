begin;

update "data"."data_tbl_pet_soul"
set "name" = 'Vĩnh Sinh',
    "desc" = 'Tăng Miễn Tử ' || trim(trailing '.' from trim(trailing '0' from "prop_val"::text))
where "id" between 751 and 760
  and "prop_type" = 34;

commit;

-- Inverse for an operator-approved rollback:
-- update "data"."data_tbl_pet_soul" set "name"='永生', "desc"='Tăng免死 ' || trim(trailing '.' from trim(trailing '0' from "prop_val"::text)) where "id" between 751 and 760 and "prop_type"=34;
