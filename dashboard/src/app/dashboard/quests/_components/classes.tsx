"use client"

import { useQuestResource } from "@/hooks/use-quests"
import {
    classValues,
    serializeClasses,
    text,
    type Lookup,
    type Value,
} from "../_lib/editor"
import { ApiError } from "./fields"

export function ClassSelector({
    value,
    onChange,
}: {
    value: Value | undefined
    onChange: (value: string) => void
}) {
    const { data, error } = useQuestResource<{ items: Lookup[] }>(
        "/api/quests/lookups?kind=class&pageSize=100"
    )
    const selected = classValues(value)
    const known = data?.items ?? []
    const unknown = selected.filter(
        id => id !== "all" && !known.some(item => String(item.id) === id)
    )
    return (
        <div className="space-y-2">
            <p className="font-medium">Nghề nghiệp</p>
            <p className="text-xs text-muted-foreground">
                Giá trị lưu: {value === null ? "NULL" : text(value)}. Giữ nguyên
                thứ tự / chuỗi cũ nếu không thay đổi lựa chọn.
            </p>
            <label className="flex gap-2">
                <input
                    type="checkbox"
                    checked={value === "all"}
                    onChange={event =>
                        onChange(
                            event.target.checked
                                ? "all"
                                : serializeClasses(
                                      known.map(item => String(item.id))
                                  )
                        )
                    }
                />
                Tất cả
            </label>
            <div className="flex flex-wrap gap-4">
                {[
                    ...known.map(item => ({
                        id: String(item.id),
                        name: item.name,
                    })),
                    ...unknown.map(id => ({ id, name: "Nghề lịch sử" })),
                ].map(item => (
                    <label key={item.id} className="flex gap-2 text-sm">
                        <input
                            type="checkbox"
                            checked={selected.includes(item.id)}
                            onChange={event =>
                                onChange(
                                    serializeClasses(
                                        event.target.checked
                                            ? [
                                                  ...selected.filter(
                                                      id => id !== "all"
                                                  ),
                                                  item.id,
                                              ]
                                            : selected.filter(
                                                  id => id !== item.id
                                              )
                                    )
                                )
                            }
                        />
                        {item.name} (#{item.id})
                    </label>
                ))}
            </div>
            {error && <ApiError error={error} />}
        </div>
    )
}
