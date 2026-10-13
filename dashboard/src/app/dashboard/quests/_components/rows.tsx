"use client"

import { Button } from "@/components/ui/button"
import { EnumField, Field, Picker } from "./fields"
import { text, type Collection, type DraftRow } from "../_lib/editor"

const pairs = {
    requirements: [
        ["1:29", "Thu thập vật phẩm"],
        ["1:19", "Thu thập trang bị"],
        ["2:12", "Diệt quái"],
        ["3:12", "Nộp thú cưng"],
    ],
    awards: [
        ["1:29", "Vật phẩm"],
        ["1:19", "Trang bị"],
        ["2:12", "Thú cưng"],
    ],
    prerequisites: [
        ["1:29", "Mang theo vật phẩm"],
        ["2:45", "Hoàn thành nhiệm vụ"],
    ],
} as const
export function RowsEditor({
    collection,
    rows,
    onChange,
}: {
    collection: Collection
    rows: DraftRow[]
    onChange: (rows: DraftRow[]) => void
}) {
    function edit(key: string, fields: Partial<DraftRow>) {
        onChange(
            rows.map(row => (row._key === key ? { ...row, ...fields } : row))
        )
    }
    return (
        <div className="space-y-4">
            {rows.map(row => {
                const pair = `${row.kind}:${row.type}`
                const supported =
                    pairs[collection].some(([value]) => value === pair) ||
                    (collection === "prerequisites" && pair === "2:-1")
                const pet =
                    collection === "awards" ? pair === "2:12" : pair === "3:12"
                const catalog =
                    text(row.type) === "29"
                        ? "item"
                        : text(row.type) === "19"
                          ? "equipment"
                          : collection === "prerequisites"
                            ? "quest"
                            : pet
                              ? "pet"
                              : "monster"
                const q = Number(row.q)
                const quality =
                    row.q == null
                        ? "NULL lịch sử — giữ nguyên, không đổi thành 0"
                        : collection === "awards"
                          ? pet
                              ? `Mã chất lượng cấp thú q=${text(row.q)}; không phải bộ lọc nộp thú`
                              : q <= 0
                                ? "Chất lượng cấp mặc định của danh mục; không phải bộ lọc thu thập"
                                : `Màu tham chiếu từ q: ${Math.floor((q + 4) / 5)}; cấp trang bị / vật phẩm có đường diễn giải riêng`
                          : pet
                            ? q < 0
                                ? "Không lọc ngưỡng tăng trưởng"
                                : `Ngưỡng tăng trưởng thú: ${q / 10}`
                            : q < 0
                              ? "Không lọc phẩm chất"
                              : `Màu suy ra: ${Math.floor((q + 4) / 5)} (so khớp chính xác)`
                return (
                    <section
                        key={row._key}
                        className="space-y-4 rounded-lg border p-4"
                    >
                        <div className="flex items-center justify-between">
                            <h3 className="font-medium">
                                {row.id ? `Dòng #${row.id}` : "Dòng mới"}
                            </h3>
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() =>
                                    onChange(
                                        rows.filter(
                                            item => item._key !== row._key
                                        )
                                    )
                                }
                            >
                                Xóa dòng
                            </Button>
                        </div>
                        {!supported && (
                            <p className="text-sm text-amber-600">
                                Loại lịch sử chưa được hỗ trợ. Giữ nguyên dữ
                                liệu; chỉ đổi loại khi chủ động thay thế toàn bộ
                                ý nghĩa dòng.
                            </p>
                        )}
                        {collection === "awards" &&
                            (text(row.b) !== "1" ||
                                (pet && text(row.num) !== "1")) && (
                                <p className="text-sm text-amber-600">
                                    Dòng thưởng cũ: runtime luôn khóa, chỉ cấp
                                    một thú mỗi dòng. Nếu sửa dòng phải chọn
                                    khóa = 1 và số thú = 1.
                                </p>
                            )}
                        <EnumField
                            label="Loại dòng / bảng đích"
                            value={pair}
                            options={pairs[collection]}
                            onChange={value => {
                                const [kind, type] = value.split(":")
                                edit(row._key, {
                                    kind,
                                    type,
                                    item_id: "0",
                                    ...(collection === "awards"
                                        ? {
                                              b: "1",
                                              ...(type === "12"
                                                  ? { num: "1" }
                                                  : {}),
                                          }
                                        : {}),
                                })
                            }}
                        />
                        {supported && (
                            <div className="grid gap-4 md:grid-cols-2">
                                <Picker
                                    label="Đối tượng"
                                    catalog={catalog}
                                    value={row.item_id}
                                    onChange={value =>
                                        edit(row._key, { item_id: value })
                                    }
                                />
                                {collection === "awards" && pet ? (
                                    <div className="space-y-2 text-sm">
                                        Số thú mỗi dòng: 1 (runtime không hỗ trợ
                                        nhiều thú trong một dòng). Có thể thêm
                                        các dòng riêng; kiểm tra sức chứa túi
                                        thú.
                                        {text(row.num) !== "1" && (
                                            <>
                                                <p>
                                                    Số lượng cũ đang lưu:{" "}
                                                    {text(row.num)}
                                                </p>
                                                <Button
                                                    variant="outline"
                                                    onClick={() =>
                                                        edit(row._key, {
                                                            num: "1",
                                                        })
                                                    }
                                                >
                                                    Chủ động đổi số thú về 1
                                                </Button>
                                            </>
                                        )}
                                    </div>
                                ) : (
                                    <Field
                                        label="Số lượng"
                                        numeric
                                        value={row.num}
                                        onChange={value =>
                                            edit(row._key, { num: value })
                                        }
                                    />
                                )}
                                {collection !== "prerequisites" &&
                                    !(
                                        collection === "requirements" &&
                                        pair === "2:12"
                                    ) && (
                                        <div className="space-y-2">
                                            <Field
                                                label="q — mã phẩm chất được lưu (không phải mã màu)"
                                                numeric
                                                value={row.q}
                                                onChange={value =>
                                                    edit(row._key, { q: value })
                                                }
                                            />
                                            <p className="text-xs text-muted-foreground">
                                                {quality}
                                                {collection === "awards" &&
                                                    " · diễn giải cấp thưởng có thể khác yêu cầu thu thập"}
                                            </p>
                                        </div>
                                    )}
                                {collection === "awards" && (
                                    <EnumField
                                        label="Khóa vật phẩm (runtime luôn khóa)"
                                        value={row.b}
                                        options={[[1, "Khóa"]]}
                                        onChange={value =>
                                            edit(row._key, { b: value })
                                        }
                                    />
                                )}
                            </div>
                        )}
                        <details className="text-xs">
                            <summary>Dữ liệu gốc / trường lịch sử</summary>
                            <pre className="overflow-auto whitespace-pre-wrap">
                                {JSON.stringify(
                                    Object.fromEntries(
                                        Object.entries(row).filter(
                                            ([key]) => key !== "_key"
                                        )
                                    ),
                                    null,
                                    2
                                )}
                            </pre>
                        </details>
                    </section>
                )
            })}
            {!rows.length && (
                <p className="text-muted-foreground">Chưa có dòng nào.</p>
            )}
            <Button
                variant="outline"
                onClick={() =>
                    onChange([
                        ...rows,
                        {
                            _key: crypto.randomUUID(),
                            kind: "1",
                            type: "29",
                            item_id: "0",
                            num: "1",
                            ...(collection !== "prerequisites"
                                ? { q: "-1" }
                                : {}),
                            ...(collection === "awards" ? { b: "1" } : {}),
                        },
                    ])
                }
            >
                Thêm dòng
            </Button>
        </div>
    )
}
