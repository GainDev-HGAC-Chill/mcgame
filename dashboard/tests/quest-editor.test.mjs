import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import test from "node:test"
import {
    aggregatePatch,
    changedFields,
    draftRows,
    rowPatch,
    serializeClasses,
} from "../src/app/dashboard/quests/_lib/editor.ts"

test("untouched nulls, unknown values and exact class serialization produce no patch", () => {
    const before = {
        id: 1,
        req_class: "|3|1|",
        is_rebirth: null,
        name: "",
        type: "97",
        award_exp_re: "9007199254740993",
        qtest: " <script>legacy</script> ",
    }
    assert.deepEqual(changedFields(before, { ...before }), {})
    assert.deepEqual(changedFields(before, { ...before, name: "Tên mới" }), {
        name: "Tên mới",
    })
})
test("child delta retains ids and unknown data; changes are nested, omission is not deletion", () => {
    const before = [
        {
            id: 5,
            qid: "42",
            kind: null,
            type: "99",
            num: null,
            item_id: "0",
            q: "23",
        },
        {
            id: 7,
            qid: "42",
            kind: "2",
            type: "12",
            num: "2",
            item_id: "10",
            q: "-1",
        },
    ]
    assert.deepEqual(rowPatch(before, draftRows(before)), {
        update: [],
        insert: [],
        deleteIds: [],
    })
    const after = draftRows(before)
    after[1].num = "3"
    assert.deepEqual(rowPatch(before, after), {
        update: [{ id: 7, changes: { num: "3" } }],
        insert: [],
        deleteIds: [],
    })
    assert.deepEqual(rowPatch(before, [after[0]]).deleteIds, [7])
})
test("new rows omit temporary keys and ownership; server allocates ids", () => {
    assert.deepEqual(
        rowPatch(
            [],
            [
                {
                    _key: "draft",
                    qid: "42",
                    kind: "1",
                    type: "29",
                    item_id: "5",
                    q: "-1",
                    num: "1",
                },
            ]
        ),
        {
            update: [],
            deleteIds: [],
            insert: [
                { kind: "1", type: "29", item_id: "5", q: "-1", num: "1" },
            ],
        }
    )
})
test("aggregate includes only dirty groups and preserves untouched legacy rows", () => {
    const detail = {
        questId: 42,
        revision: "token",
        quest: { id: 42, name: null, req_class: null },
        requirements: [{ id: 5, type: "99", kind: null, num: null }],
        awards: [],
        prerequisites: [],
        warnings: [],
        canWrite: true,
    }
    const rows = {
        requirements: draftRows(detail.requirements),
        awards: [],
        prerequisites: [],
    }
    assert.deepEqual(aggregatePatch(detail, { ...detail.quest }, rows), {})
    assert.deepEqual(
        aggregatePatch(detail, { ...detail.quest, name: "Nhiệm vụ" }, rows),
        { quest: { name: "Nhiệm vụ" } }
    )
})
test("class selections serialize pipes in selection order", () => {
    assert.equal(serializeClasses(["3", "1"]), "|3|1|")
    assert.equal(serializeClasses([]), "all")
})

test("active quests remain fully editable while revision and audit guards stay enabled", () => {
    const page = readFileSync(
        new URL("../src/app/dashboard/quests/[id]/page.tsx", import.meta.url),
        "utf8"
    )
    const migration = readFileSync(
        new URL(
            "../../supabase/migrations/20261016073610_allow_edit_active_quests.sql",
            import.meta.url
        ),
        "utf8"
    )
    assert.doesNotMatch(page, /unsafeActivePatch/)
    assert.match(page, /const structuralDisabled = disabled/)
    assert.doesNotMatch(migration, /ACTIVE_QUEST_IN_USE/)
    assert.match(migration, /Quest revision conflict/)
    assert.match(migration, /insert into data\.quest_admin_audit/)
})
