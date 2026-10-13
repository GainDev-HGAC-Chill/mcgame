import "server-only"
import { z } from "zod"
import { createAdminClient } from "@/lib/supabase/admin"
import { applyQuestAuthHeaders, QuestAuthError, questAuthErrorResponse, requireQuestPermission, type QuestAdminIdentity } from "@/lib/server/quest-admin-auth"
import type { QuestActivation, QuestDetail, QuestLoopDetail } from "@/types/quests"

export class QuestApiError extends Error {
    constructor(readonly status: number, readonly code: string, message: string) { super(message) }
}
export async function questRpc<T>(name: string, args: Record<string, unknown>): Promise<T> {
    const { data, error } = await createAdminClient().rpc(name, args)
    if (error) {
        const status = error.code === "40001" ? 409 : error.code === "P0002" ? 404 : ["22023","22P02","22003","23502","23505"].includes(error.code) ? 422 : 503
        throw new QuestApiError(status, status === 409 ? "quest_conflict" : status === 404 ? "quest_not_found" : status === 422 ? "quest_validation" : "quest_unavailable", status === 503 ? "Dịch vụ dữ liệu chưa sẵn sàng." : error.message)
    }
    if (data === null) throw new QuestApiError(404,"quest_not_found","Không tìm thấy dữ liệu.")
    return data as T
}
export async function readQuestBody(request: Request): Promise<unknown> {
    if (!/^application\/json(?:\s*;|$)/i.test(request.headers.get("content-type") ?? "")) throw new QuestApiError(415,"quest_content_type","Cần application/json.")
    const limit = 128 * 1024
    const declared = request.headers.get("content-length")
    if (declared && Number(declared) > limit) throw new QuestApiError(413,"quest_body_limit","Nội dung quá lớn.")
    const reader = request.body?.getReader()
    if (!reader) throw new QuestApiError(422,"quest_json","Thiếu nội dung JSON.")
    const chunks: Uint8Array[] = []; let size = 0
    try {
        while (true) {
            const { value, done } = await reader.read()
            if (done) break
            size += value.byteLength
            if (size > limit) { await reader.cancel(); throw new QuestApiError(413,"quest_body_limit","Nội dung quá lớn.") }
            chunks.push(value)
        }
        const bytes = new Uint8Array(size); let offset = 0
        chunks.forEach(chunk => { bytes.set(chunk,offset); offset += chunk.byteLength })
        return JSON.parse(new TextDecoder("utf-8",{fatal:true}).decode(bytes)) as unknown
    } catch (error) {
        if (error instanceof QuestApiError) throw error
        throw new QuestApiError(422,"quest_json","JSON không hợp lệ.")
    } finally { reader.releaseLock() }
}
export function questActivation(saved = false): QuestActivation {
    return {databaseSaved:saved,serverReloadRequired:true,clientRefreshRequired:true,mode:"database-only",flashCatalogBlocked:true,clientCatalogPublicationRequired:true}
}
const activationWarning = "Chỉ lưu cơ sở dữ liệu. Flash dùng danh mục nhúng: khởi động lại máy chủ và đăng nhập lại KHÔNG bảo đảm đồng bộ. Cần xuất bản danh mục client đã kiểm chứng; kích hoạt nhất quán hiện bị chặn."
export function decorateLoop(detail: QuestLoopDetail, canWrite: boolean, saved = false): QuestLoopDetail {
    return {...detail,canWrite,activation:questActivation(saved),warnings:[activationWarning,"Cấu hình vòng lặp dùng chung; thay đổi tác động mọi nhiệm vụ liên quan. Không sửa tiến độ người chơi.","Chặn chỉnh ngữ nghĩa khi có vòng/nhiệm vụ con đang dùng đã lưu; không thấy tiến độ online/cache chưa ghi. Không tự kích hoạt."]}
}
export function decorateQuest(detail: QuestDetail, canWrite: boolean, saved = false): QuestDetail {
    const warnings = [...(detail.warnings ?? []),activationWarning,"Bảo vệ đang dùng chỉ dựa trên tiến độ đã lưu: phiên online/cache write-behind có thể chưa xuất hiện trong DB. Không kích hoạt/restart khi chưa kiểm chứng bảo toàn tiến độ.","Phần thưởng lịch sử được gửi theo đường best-effort, không bảo đảm giao dịch nguyên tử.","Đồ/thú thưởng hiện luôn khóa; thú thưởng chỉ cấp một mẫu (q > 0: mức tăng trưởng q/10; q <= 0: ngẫu nhiên theo mẫu)."]
    if (detail.quest.at !== "1") warnings.push("Chế độ chọn một/lọc môn phái chưa được backend hỗ trợ; giữ nguyên dữ liệu lịch sử.")
    if (detail.quest.rn !== "0") warnings.push("Điều kiện tiền tệ chưa được backend hỗ trợ.")
    if (detail.quest.a_script_text || detail.quest.script_give_up || detail.quest.qtest) warnings.push("Script/QA lịch sử chỉ đọc; không tự chuẩn hóa hoặc thực thi.")
    if (detail.quest.award_exp_re === "-1") warnings.push("EXP -1 là sentinel lịch sử, được giữ nguyên khi không chỉnh sửa.")
    if (Number(detail.quest.min_level)>Number(detail.quest.max_level)) warnings.push("Khoảng cấp lịch sử đảo ngược (có thể cố ý vô hiệu hóa).")
    if (detail.activeUse.activeQuests || detail.activeUse.activeLoops) warnings.push("Có người chơi đang dùng nhiệm vụ; thay đổi không đặt lại tiến độ hoặc bồi thường.")
    if (!["-1","0","1","2","3","4"].includes(detail.quest.money_type ?? "")) warnings.push("Loại tiền thưởng lịch sử chưa có ánh xạ Flash/backend thống nhất; không cho chỉnh phần thưởng tiền này.")
    detail.requirements.forEach(row => { if (!(["1:19","1:29","2:12","3:12"].includes(`${row.kind}:${row.type}`))) warnings.push(`Yêu cầu ${row.id}: loại lịch sử không hỗ trợ.`) })
    detail.awards.forEach(row => { if (!(["1:19","1:29","2:12"].includes(`${row.kind}:${row.type}`)) || row.b!=="1" || (row.type==="12" && row.num!=="1")) warnings.push(`Thưởng ${row.id}: giá trị lịch sử không được runtime tôn trọng đầy đủ.`) })
    return {...detail,canWrite,warnings,loopRef:detail.loopRef ? decorateLoop(detail.loopRef,canWrite) : null,activation:questActivation(saved)}
}
export async function questRoute(request: Request, permission: "read" | "write", handler: (identity: QuestAdminIdentity) => Promise<unknown>): Promise<Response> {
    let identity: QuestAdminIdentity | undefined
    try {
        identity = await requireQuestPermission(request,permission)
        return applyQuestAuthHeaders(Response.json(await handler(identity)),identity)
    } catch (error) {
        if (error instanceof QuestAuthError) return questAuthErrorResponse(error)
        const response = error instanceof z.ZodError
            ? Response.json({error:{code:"quest_validation",message:"Dữ liệu không hợp lệ.",issues:error.issues}},{status:422})
            : error instanceof QuestApiError
                ? Response.json({error:{code:error.code,message:error.message}},{status:error.status})
                : Response.json({error:{code:"quest_unavailable",message:"Dịch vụ chưa sẵn sàng."}},{status:503})
        response.headers.set("Cache-Control","private, no-store")
        return identity ? applyQuestAuthHeaders(response,identity) : response
    }
}
export function parseQuestPathId(value: string): number {
    if (!/^[1-9][0-9]{0,9}$/.test(value) || Number(value)>2147483647) throw new QuestApiError(422,"quest_id","ID không hợp lệ.")
    return Number(value)
}
