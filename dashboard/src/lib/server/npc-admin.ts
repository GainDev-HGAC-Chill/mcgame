import "server-only"

import { z } from "zod"

import {
    applyQuestAuthHeaders,
    QuestAuthError,
    questAuthErrorResponse,
    requireQuestPermission,
    type QuestAdminIdentity,
} from "@/lib/server/quest-admin-auth"
import { createAdminClient } from "@/lib/supabase/admin"
import type { NpcActivation, NpcDetail, NpcLookupResponse } from "@/types/npcs"

export class NpcApiError extends Error {
    constructor(
        readonly status: number,
        readonly code: string,
        message: string
    ) {
        super(message)
    }
}

export async function npcRpc<T>(
    name: string,
    args: Record<string, unknown>
): Promise<T> {
    const { data, error } = await createAdminClient().rpc(name, args)
    if (error) {
        const status =
            error.code === "40001"
                ? 409
                : error.code === "P0002"
                  ? 404
                  : ["22023", "22P02", "22003", "23502", "23505"].includes(
                          error.code
                      )
                    ? 422
                    : 503
        throw new NpcApiError(
            status,
            status === 409
                ? "npc_conflict"
                : status === 404
                  ? "npc_not_found"
                  : status === 422
                    ? "npc_validation"
                    : "npc_unavailable",
            status === 503 ? "Dịch vụ dữ liệu chưa sẵn sàng." : error.message
        )
    }
    if (data === null)
        throw new NpcApiError(404, "npc_not_found", "Không tìm thấy NPC.")
    return data as T
}

export async function readNpcBody(request: Request): Promise<unknown> {
    if (
        !/^application\/json(?:\s*;|$)/i.test(
            request.headers.get("content-type") ?? ""
        )
    )
        throw new NpcApiError(415, "npc_content_type", "Cần application/json.")
    const limit = 128 * 1024
    const declared = request.headers.get("content-length")
    if (declared && Number(declared) > limit)
        throw new NpcApiError(413, "npc_body_limit", "Nội dung quá lớn.")
    const reader = request.body?.getReader()
    if (!reader) throw new NpcApiError(422, "npc_json", "Thiếu nội dung JSON.")
    const chunks: Uint8Array[] = []
    let size = 0
    try {
        while (true) {
            const { value, done } = await reader.read()
            if (done) break
            size += value.byteLength
            if (size > limit) {
                await reader.cancel()
                throw new NpcApiError(
                    413,
                    "npc_body_limit",
                    "Nội dung quá lớn."
                )
            }
            chunks.push(value)
        }
        const bytes = new Uint8Array(size)
        let offset = 0
        for (const chunk of chunks) {
            bytes.set(chunk, offset)
            offset += chunk.byteLength
        }
        return JSON.parse(
            new TextDecoder("utf-8", { fatal: true }).decode(bytes)
        ) as unknown
    } catch (error) {
        if (error instanceof NpcApiError) throw error
        throw new NpcApiError(422, "npc_json", "JSON không hợp lệ.")
    } finally {
        reader.releaseLock()
    }
}

export function npcActivation(saved = false): NpcActivation {
    return {
        databaseSaved: saved,
        serverRestartRequired: true,
        clientReconnectRequired: true,
        sceneReentryRequired: true,
        questCatalogBlocked: true,
    }
}

type CatalogRow = { id: number; name: string | null; icon_code?: number | null }

async function hydrateShopDisplay(detail: NpcDetail): Promise<NpcDetail> {
    if (!detail.shop?.slots.length) return detail
    const itemIDs = [
        ...new Set(
            detail.shop.slots
                .filter(row => row.type !== 19)
                .map(row => row.item_id)
        ),
    ]
    const equipmentIDs = [
        ...new Set(
            detail.shop.slots
                .filter(row => row.type === 19)
                .map(row => row.item_id)
        ),
    ]
    const client = createAdminClient().schema("data")
    const [items, equipment] = await Promise.all([
        itemIDs.length
            ? client
                  .from("data_tbl_item_template")
                  .select("id,name,icon_code")
                  .in("id", itemIDs)
            : Promise.resolve({ data: [], error: null }),
        equipmentIDs.length
            ? client
                  .from("data_tbl_equipt_template")
                  .select("id,name,icon_code")
                  .in("id", equipmentIDs)
            : Promise.resolve({ data: [], error: null }),
    ])
    if (items.error || equipment.error)
        throw new NpcApiError(
            503,
            "npc_catalog_unavailable",
            "Không tải được tên và hình catalog."
        )
    const catalog = new Map<string, CatalogRow>()
    for (const row of (items.data ?? []) as CatalogRow[])
        catalog.set(`29:${row.id}`, row)
    for (const row of (equipment.data ?? []) as CatalogRow[])
        catalog.set(`19:${row.id}`, row)
    return {
        ...detail,
        shop: {
            ...detail.shop,
            slots: detail.shop.slots.map(row => {
                const found = catalog.get(
                    `${row.type === 19 ? 19 : 29}:${row.item_id}`
                )
                return {
                    ...row,
                    displayName: found?.name ?? null,
                    iconCode: Number(found?.icon_code) || 0,
                }
            }),
        },
    }
}

export async function npcLookup(query: {
    kind: "shop" | "item" | "equipment" | "map"
    search: string
    page: number
    pageSize: number
}): Promise<NpcLookupResponse> {
    const table = {
        shop: "data_tbl_shop",
        item: "data_tbl_item_template",
        equipment: "data_tbl_equipt_template",
        map: "data_tbl_map",
    }[query.kind]
    const hasIcon = query.kind === "item" || query.kind === "equipment"
    let request = createAdminClient()
        .schema("data")
        .from(table)
        .select(hasIcon ? "id,name,icon_code" : "id,name", { count: "exact" })
        .order("id")
        .range(
            (query.page - 1) * query.pageSize,
            query.page * query.pageSize - 1
        )
    if (query.search) {
        request = /^[1-9][0-9]{0,9}$/.test(query.search)
            ? request.eq("id", Number(query.search))
            : request.ilike("name", `%${query.search}%`)
    }
    const { data, error, count } = await request
    if (error)
        throw new NpcApiError(
            503,
            "npc_catalog_unavailable",
            "Không tải được catalog tra cứu."
        )
    return {
        items: ((data ?? []) as unknown as CatalogRow[]).map(row => ({
            id: Number(row.id),
            name: row.name,
            table,
            iconCode: Number(row.icon_code) || 0,
        })),
        total: count ?? 0,
        page: query.page,
        pageSize: query.pageSize,
    }
}

export async function decorateNpc(
    detail: NpcDetail,
    canWrite: boolean,
    saved = false
): Promise<NpcDetail> {
    const hydrated = await hydrateShopDisplay(detail)
    const warnings = [
        ...(hydrated.warnings ?? []),
        "NPC và cửa hàng chỉ có hiệu lực sau khi máy chủ được khởi động lại có kiểm soát; client cần kết nối lại và vào lại bản đồ.",
        "Liên kết nhiệm vụ chỉ đọc tại đây. Cấu trúc danh mục nhiệm vụ Flash vẫn chưa có đường kích hoạt nhất quán.",
    ]
    return {
        ...hydrated,
        canWrite,
        warnings,
        activation: npcActivation(saved),
    }
}

export async function npcRoute(
    request: Request,
    permission: "read" | "write",
    handler: (identity: QuestAdminIdentity) => Promise<unknown>
): Promise<Response> {
    let identity: QuestAdminIdentity | undefined
    try {
        identity = await requireQuestPermission(request, permission)
        return applyQuestAuthHeaders(
            Response.json(await handler(identity)),
            identity
        )
    } catch (error) {
        if (error instanceof QuestAuthError)
            return questAuthErrorResponse(error)
        const response =
            error instanceof z.ZodError
                ? Response.json(
                      {
                          error: {
                              code: "npc_validation",
                              message: "Dữ liệu không hợp lệ.",
                              issues: error.issues,
                          },
                      },
                      { status: 422 }
                  )
                : error instanceof NpcApiError
                  ? Response.json(
                        { error: { code: error.code, message: error.message } },
                        { status: error.status }
                    )
                  : Response.json(
                        {
                            error: {
                                code: "npc_unavailable",
                                message: "Dịch vụ chưa sẵn sàng.",
                            },
                        },
                        { status: 503 }
                    )
        response.headers.set("Cache-Control", "private, no-store")
        return identity ? applyQuestAuthHeaders(response, identity) : response
    }
}

export function parseNpcPathId(value: string): number {
    if (!/^[1-9][0-9]{0,9}$/.test(value) || Number(value) > 2147483647)
        throw new NpcApiError(422, "npc_id", "ID NPC không hợp lệ.")
    return Number(value)
}
