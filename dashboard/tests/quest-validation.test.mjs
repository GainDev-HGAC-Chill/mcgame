import assert from "node:assert/strict"
import { test } from "node:test"
import { questSaveSchema, questLoopSaveSchema, questListQuerySchema } from "../src/lib/quests/validation.ts"
const base = { expectedRevision:"a".repeat(32),reason:"test reason",requestId:"11111111-1111-4111-8111-111111111111" }
const valid = patch => questSaveSchema.safeParse({...base,patch}).success
test("strict metadata delta without child replacement or null normalization", () => {
    assert.equal(valid({quest:{name:"Tên mới",info:null}}),true)
    for (const patch of [{quest:{id:5}}, {quest:{script_give_up:"x"}}, {quest:{award_exp:"5"}}, {requirements:[]}, {quest:{name:"x",qid:"5"}},{}]) assert.equal(valid(patch),false)
})
test("reward safety and lossless decimal strings", () => {
    assert.equal(valid({quest:{award_exp_re:"100000000000000",money_type:"2",money_num:"20"}}),true)
    for (const quest of [{award_exp_re:-1},{award_exp_re:"-1"},{money_num:"1.5"},{money_num:"01"},{at:"2"},{at:"3"},{money_type:"10"},{rn:"5"},{gender:"8"}]) assert.equal(valid({quest}),false)
})
test("type-aware requirements and award limitations", () => {
    const row = {item_id:"1",kind:"1",type:"29",num:"2",q:"-1"}
    assert.equal(valid({requirements:{insert:[row]}}),true)
    assert.equal(valid({requirements:{insert:[{...row,kind:"2"}]}}),false)
    assert.equal(valid({awards:{insert:[{...row,b:"1"}]}}),true)
    assert.equal(valid({awards:{insert:[{...row,b:"0"}]}}),false)
    assert.equal(valid({awards:{insert:[{...row,b:"1",kind:"2",type:"12"}]}}),false)
    assert.equal(valid({awards:{insert:[{...row,b:"1",kind:"2",type:"12",num:"1"}]}}),true)
})
test("immutable ownership and duplicate mutations rejected", () => {
    assert.equal(valid({requirements:{update:[{id:1,changes:{qid:"5"}}]}}),false)
    assert.equal(valid({requirements:{update:[{id:1,changes:{num:"2"}}],deleteIds:[1]}}),false)
    assert.equal(valid({requirements:{deleteIds:[1,1]}}),false)
    assert.equal(valid({requirements:{insert:[{id:1,item_id:"1",kind:"1",type:"29",num:"1",q:"0"}]}}),false)
})
test("shared loop opt-in, impact and revision required", () => {
    const body = {...base,confirmShared:true,expectedAffectedCount:38,patch:{name:"Shared"}}
    assert.equal(questLoopSaveSchema.safeParse(body).success,true)
    for (const edit of [{confirmShared:false},{expectedAffectedCount:-1},{expectedRevision:"stale"},{patch:{script_get:"x"}}]) assert.equal(questLoopSaveSchema.safeParse({...body,...edit}).success,false)
})
test("bounded list pagination and unknown parameters", () => {
    assert.equal(questListQuerySchema.parse({}).pageSize,25)
    for (const query of [{page:0},{pageSize:101},{id:-1},{table:"player.characters"}]) assert.equal(questListQuerySchema.safeParse(query).success,false)
})

test("live function/API negatives without committed catalog writes", {skip:!process.env.QUEST_FUNCTION_BASE_URL}, async () => {
    const baseUrl = process.env.QUEST_FUNCTION_BASE_URL
    const service = process.env.QUEST_FUNCTION_SERVICE_KEY
    const anon = process.env.QUEST_FUNCTION_ANON_KEY
    assert.ok(service && anon)
    const rpc = async (name, args, key = service) => {
        const response = await fetch(`${baseUrl}/rest/v1/rpc/${name}`,{method:"POST",headers:{apikey:key,Authorization:`Bearer ${key}`,"Content-Type":"application/json"},body:JSON.stringify(args)})
        return {status:response.status,data:await response.json()}
    }
    const before = await rpc("quest_admin_detail",{p_id:162})
    assert.equal(before.status,200)
    assert.equal(typeof before.data.quest.award_exp_re,"string")
    for (const kind of ["npc","monster","item","equipment","pet","quest","loop","class"]) {
        const response = await rpc("quest_admin_lookups",{p_kind:kind,p_search:"",p_page:1,p_page_size:1})
        assert.equal(response.status,200)
        assert.equal(response.data.items.length,1)
    }
    const denied = await rpc("quest_admin_detail",{p_id:162},anon)
    assert.ok(denied.status>=400)
    for (const patch of [{quest:{at:"2"}},{quest:{start_npc:"2147483647"}},{quest:{type:"7",sub_type:"-1"}},{requirements:{insert:[{item_id:"515",kind:"1",type:"29",num:"1",q:"10000"}]}},{requirements:{deleteIds:[2147483647]}},{quest:{name:"must rollback"},requirements:{insert:[{item_id:"2147483647",kind:"1",type:"29",num:"1",q:"0"}]}}]) {
        const response = await rpc("quest_admin_save",{p_id:162,p_expected_revision:before.data.revision,p_patch:patch,p_reason:"negative API validation",p_request_id:crypto.randomUUID(),p_actor_id:"11111111-1111-4111-8111-111111111111"})
        assert.ok(response.status>=400)
        assert.deepEqual((await rpc("quest_admin_detail",{p_id:162})).data,before.data)
    }
    if (process.env.QUEST_DASHBOARD_BASE_URL) {
        for (const path of ["/api/quests","/api/quests/162","/api/quests/lookups?kind=item","/api/quests/loops/1","/api/quests/access"]) {
            const response = await fetch(`${process.env.QUEST_DASHBOARD_BASE_URL}${path}`)
            assert.equal(response.status,401)
            assert.match(response.headers.get("cache-control"),/no-store/)
        }
        const response = await fetch(`${process.env.QUEST_DASHBOARD_BASE_URL}/api/quests/162`,{method:"PATCH",headers:{"Content-Type":"application/json"},body:"{}"})
        assert.equal(response.status,401)
    }
})
