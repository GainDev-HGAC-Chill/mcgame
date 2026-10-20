"use client"

import { createClientUUID } from "@/lib/client-uuid"

import { useEffect, useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { CatalogLookup } from "@/app/dashboard/npcs/_components/catalog-lookup"
import { npcRequest, useNpcResource } from "@/hooks/use-npcs"
import {
    SYSTEM_SHOPS, SLOT_STATES, emptyShopSlot,
    type ShopSlotForm, type ShopSlotRow, type SystemShopDetail,
} from "@/lib/system-shop/config"

function fromRow(row: ShopSlotRow): ShopSlotForm {
    return {
        type: row.type, item_id: row.item_id,
        gold: row.gold, money: row.money, point: row.point,
        gt: row.gt, quality: row.quality, amount: row.amount,
        st: row.st, position: row.position,
    }
}

function NumberBox({label,value,disabled=false,min=0,onChange}:{
    label:string;value:number;disabled?:boolean;min?:number;onChange:(n:number)=>void
}) {
    return (
        <label className="grid gap-1 text-sm">
            <span className="text-muted-foreground">{label}</span>
            <Input type="number" min={min} step={1} value={value}
                disabled={disabled} onChange={event=>onChange(Number(event.target.value))}/>
        </label>
    )
}

export default function SystemShopPage() {
    const [sid,setSid] = useState<number>(65)
    const resource = useNpcResource<SystemShopDetail>(`/api/system-shop?sid=${sid}`)
    const [detail,setDetail] = useState<SystemShopDetail|null>(null)
    const [selectedId,setSelectedId] = useState<number|null>(null)
    const [draft,setDraft] = useState<ShopSlotForm>(emptyShopSlot())
    const [mode,setMode] = useState<"insert"|"update">("insert")
    const [search,setSearch] = useState("")
    const [statusFilter,setStatusFilter] = useState("all")
    const [reason,setReason] = useState("Bổ sung vật phẩm Shop hệ thống")
    const [saving,setSaving] = useState(false)
    const [error,setError] = useState("")
    const [success,setSuccess] = useState("")
    const [lookupName,setLookupName] = useState<string|null>(null)

    useEffect(()=>{
        if (!resource.data) return
        setDetail(resource.data)
        setMode("insert")
        setSelectedId(null)
        setDraft(emptyShopSlot(resource.data.slots))
        setError("")
    },[resource.data])

    const current = detail?.slots.find(row=>row.id===selectedId) ?? null
    const specialCurrency = Boolean(current && (current.p_type1 || current.p_type2))
    const visibleSlots = useMemo(()=> (detail?.slots ?? []).filter(row=>{
        const q=search.trim().toLowerCase()
        return (!q || String(row.item_id)===q || String(row.id)===q
            || (row.displayName??"").toLowerCase().includes(q))
            && (statusFilter==="all" || String(row.st)===statusFilter)
    }),[detail,search,statusFilter])

    function update<K extends keyof ShopSlotForm>(key:K,value:ShopSlotForm[K]){
        setDraft(prev=>({...prev,[key]:value}))
    }
    function pick(row:ShopSlotRow){
        setMode("update")
        setSelectedId(row.id)
        setDraft(fromRow(row))
        setLookupName(row.displayName)
        setReason("Cập nhật mặt hàng Shop hệ thống")
        setError("")
        setSuccess("")
    }
    function addNew(){
        setMode("insert")
        setSelectedId(null)
        setDraft(emptyShopSlot(detail?.slots))
        setLookupName(null)
        setReason("Thêm mặt hàng Shop hệ thống")
        setError("")
        setSuccess("")
    }
    function changeSid(next:number){
        setSid(next);setDetail(null);setSelectedId(null)
        setError("");setSuccess("");setSearch("")
    }
    async function save(override?:Partial<ShopSlotForm>){
        if (!detail || !detail.canWrite || saving) return
        const payload = {...draft,...override}
        const priceKinds = Number(payload.gold>0)+Number(payload.money>0)+Number(payload.point>0)
        if(priceKinds!==1){setError("Chỉ chọn một giá dương: vàng, bạc hoặc điểm.");return}
        if(specialCurrency){setError("Ô này dùng tiền tệ đặc biệt; cần mở rộng trình chỉnh sửa trước khi thay đổi.");return}
        if(reason.trim().length<3){setError("Nhập lý do chỉnh sửa (ít nhất 3 ký tự).");return}
        if(mode==="update" && !selectedId){setError("Chưa chọn ô hàng.");return}
        if(override?.st===5 && !window.confirm("Ẩn mặt hàng khỏi Shop?")) return
        setSaving(true);setError("");setSuccess("")
        try{
            const next = await npcRequest<SystemShopDetail>("/api/system-shop",{
                method:"POST",body:JSON.stringify({
                    sid:detail.shop.id,
                    expectedRevision:detail.revision,
                    action:mode,
                    slotId:mode==="insert"?null:selectedId,
                    slot:payload,
                    reason:reason.trim(),
                    requestId:createClientUUID(),
                }),
            })
            setDetail(next)
            const updated = next.slots.find(row=>row.id===next.changedSlotId)
            if(updated) pick(updated)
            setSuccess("Đã lưu database và nhật ký. Restart server, đăng nhập lại game để kích hoạt.")
        }catch(e){
            setError(e instanceof Error ? e.message : "Không thể lưu dữ liệu.")
        }finally{
            setSaving(false)
        }
    }
    const priceMode = draft.gold>0 ? "gold" : draft.money>0 ? "money" : "point"
    const priceValue = priceMode==="gold"?draft.gold:priceMode==="money"?draft.money:draft.point
    const canEdit = Boolean(detail?.canWrite) && !saving && !specialCurrency
    return (
        <div className="space-y-5 p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                    <h1 className="text-2xl font-bold">Quản lý Shop hệ thống</h1>
                    <p className="text-sm text-muted-foreground">TBL_SHOP_SLOT · Shop chính trong giao diện game</p>
                </div>
                <Button variant="outline" onClick={()=>resource.reload()} disabled={saving}>Tải lại</Button>
            </div>
            <p className="rounded border p-3 text-sm">
                Lưu chỉ thay đổi database. Server phải được restart và game cần kết nối lại.
                Vị trí ô đã tồn tại không thể chỉnh an toàn do trình nạp Flash không ghi đè position.
                Không mở Dashboard ra mạng công cộng nếu chưa có xác thực quản trị độc lập.
            </p>
            <div className="flex flex-wrap items-end gap-3">
                <label className="grid gap-1 text-sm">
                    Chọn Shop
                    <select className="h-10 rounded-md border bg-background px-3" value={sid}
                        onChange={e=>changeSid(Number(e.target.value))} disabled={saving}>
                        {SYSTEM_SHOPS.map(s=><option key={s.sid} value={s.sid}>{s.label} (#{s.sid})</option>)}
                    </select>
                </label>
                <label className="grid gap-1 text-sm">
                    Tìm mặt hàng
                    <Input placeholder="Tên hoặc ItemID / SlotID" value={search} onChange={e=>setSearch(e.target.value)}/>
                </label>
                <label className="grid gap-1 text-sm">
                    Trạng thái
                    <select className="h-10 rounded-md border bg-background px-3" value={statusFilter} onChange={e=>setStatusFilter(e.target.value)}>
                        <option value="all">Tất cả</option>
                        {SLOT_STATES.map(x=><option key={x.id} value={String(x.id)}>{x.label}</option>)}
                        <option value="3">Limited (chỉ xem)</option>
                    </select>
                </label>
                <Button onClick={addNew} disabled={!detail?.canWrite}>+ Thêm item</Button>
            </div>
            {resource.loading && !detail && <p>Đang tải dữ liệu Shop…</p>}
            {resource.error && <p className="text-destructive">{resource.error.message}</p>}
            {error && <p role="alert" className="rounded border border-destructive p-3 text-sm text-destructive">{error}</p>}
            {success && <p className="rounded border p-3 text-sm">{success}</p>}
            {detail && (
                <div className="grid gap-5 xl:grid-cols-[minmax(0,1.35fr)_minmax(360px,1fr)]">
                    <section className="space-y-3 rounded-lg border p-4">
                        <h2 className="font-semibold">Mặt hàng: {visibleSlots.length}/{detail.slots.length} · {detail.shop.name}</h2>
                        <div className="max-h-[620px] overflow-y-auto">
                            <table className="w-full text-left text-sm">
                                <thead className="sticky top-0 bg-background"><tr className="border-b">
                                    <th className="p-2">Slot</th><th className="p-2">Item</th><th className="p-2">Giá</th><th className="p-2">Trạng thái</th><th className="p-2">Vị trí</th>
                                </tr></thead>
                                <tbody>{visibleSlots.map(row=><tr key={row.id}
                                    className={`cursor-pointer border-b hover:bg-muted/60 ${row.id===selectedId?"bg-muted":""}`}
                                    onClick={()=>pick(row)}>
                                    <td className="p-2">#{row.id}</td>
                                    <td className="p-2">{row.displayName || "Không có tên"}<span className="block text-xs text-muted-foreground">#{row.item_id} · {row.type===19?"Trang bị":"Item"}</span></td>
                                    <td className="p-2">{row.gold>0?`${row.gold} vàng`:row.money>0?`${row.money} bạc`:row.point>0?`${row.point} điểm`:row.p_type1>0?`Tiền #${row.p_type1}`:"—"}</td>
                                    <td className="p-2">{SLOT_STATES.find(s=>s.id===row.st)?.label ?? `Khác (${row.st})`}</td>
                                    <td className="p-2">{row.position}</td>
                                </tr>)}</tbody>
                            </table>
                            {!visibleSlots.length && <p className="p-3 text-muted-foreground">Không tìm thấy item.</p>}
                        </div>
                    </section>
                    <section className="space-y-4 rounded-lg border p-4">
                        <h2 className="font-semibold">{mode==="insert"?"Thêm mặt hàng mới":`Sửa ô hàng #${selectedId}`}</h2>
                        {specialCurrency && <p className="text-sm text-destructive">Slot hiện dùng p_type1/p_type2 (tiền tệ đặc biệt). Chế độ này chỉ đọc để tránh mất cấu hình giá cũ.</p>}
                        <label className="grid gap-1 text-sm">Loại danh mục
                            <select className="h-10 rounded-md border bg-background px-3" value={draft.type} disabled={!canEdit}
                                onChange={e=>{const type=Number(e.target.value) as 19|29;setDraft(prev=>({...prev,type,item_id:1}));setLookupName(null)}}>
                                <option value={29}>Vật phẩm (TBL 29)</option>
                                <option value={19}>Trang bị (TBL 19)</option>
                            </select>
                        </label>
                        <CatalogLookup kind={draft.type===19?"equipment":"item"} label="ItemID / Template"
                            value={draft.item_id} disabled={!canEdit}
                            onChange={v=>{update("item_id",v??1);setLookupName(null)}}
                            onSelect={item=>setLookupName(item.name)}/>
                        {lookupName && <p className="text-xs text-muted-foreground">Đang chọn: {lookupName}</p>}
                        <div className="grid grid-cols-2 gap-3">
                            <label className="grid gap-1 text-sm">Đơn vị giá
                                <select className="h-10 rounded-md border bg-background px-3" value={priceMode} disabled={!canEdit}
                                    onChange={e=>setDraft(prev=>({ ...prev,
                                        gold:e.target.value==="gold"?600:0,
                                        money:e.target.value==="money"?600:0,
                                        point:e.target.value==="point"?600:0,
                                    }))}>
                                    <option value="gold">Vàng</option><option value="money">Bạc</option><option value="point">Điểm</option>
                                </select>
                            </label>
                            <NumberBox label="Đơn giá" value={priceValue} disabled={!canEdit} min={1} onChange={v=>setDraft(prev=>({ ...prev,
                                gold:priceMode==="gold"?v:0,money:priceMode==="money"?v:0,point:priceMode==="point"?v:0,
                            }))}/>
                            <label className="grid gap-1 text-sm">Tiền khóa/thường
                                <select className="h-10 rounded-md border bg-background px-3" value={draft.gt} disabled={!canEdit || priceMode==="point"}
                                    onChange={e=>update("gt",Number(e.target.value) as 0|1)}>
                                    <option value={0}>Theo người chơi chọn</option>
                                    <option value={1}>Bắt buộc tiền thường</option>
                                </select>
                            </label>
                            <NumberBox label="Phẩm chất" value={draft.quality} disabled={!canEdit} onChange={v=>update("quality",v)}/>
                            <NumberBox label="Giới hạn mua (-1 = không giới hạn)" value={draft.amount} disabled={!canEdit} min={-1} onChange={v=>update("amount",v)}/>
                            <NumberBox label="Vị trí" value={draft.position} disabled={!canEdit || mode==="update"} min={1} onChange={v=>update("position",v)}/>
                        </div>
                        <label className="grid gap-1 text-sm">Trạng thái hiển thị
                            <select className="h-10 rounded-md border bg-background px-3" value={draft.st} disabled={!canEdit}
                                onChange={e=>update("st",Number(e.target.value) as ShopSlotForm["st"])}>
                                {SLOT_STATES.map(x=><option key={x.id} value={x.id}>{x.label}</option>)}
                            </select>
                        </label>
                        <label className="grid gap-1 text-sm">Lý do chỉnh sửa (ghi audit)
                            <Input value={reason} disabled={!canEdit} onChange={e=>setReason(e.target.value)} maxLength={1000}/>
                        </label>
                        <div className="flex flex-wrap gap-2">
                            <Button disabled={!canEdit} onClick={()=>void save()}>{saving?"Đang lưu…":mode==="insert"?"Thêm item":"Lưu thay đổi"}</Button>
                            {mode==="update" && draft.st!==5 && <Button variant="destructive" disabled={!canEdit} onClick={()=>void save({st:5})}>Ẩn item</Button>}
                        </div>
                    </section>
                </div>
            )}
        </div>
    )
}
