"use client"

import { useEffect, useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { gameConfigService } from "@/services/game-config.service"
import { fetchAPI } from "@/services/api"
import {
    INITIAL_CHARACTER_DEFAULTS,
    NEW_CHARACTER_DEFAULTS_KEY,
    parseNewCharacterDefaults,
    type NewCharacterDefaults,
} from "@/lib/new-character-defaults"

type CatalogOption = { id: number; name: string; subtitle: string }

export function NewCharacterDefaultsEditor() {
    const [config, setConfig] = useState<NewCharacterDefaults>(
        INITIAL_CHARACTER_DEFAULTS
    )
    const [loading, setLoading] = useState(true)
    const [ready, setReady] = useState(false)
    const [busy, setBusy] = useState(false)
    const [kind, setKind] = useState("29")
    const [search, setSearch] = useState("")
    const [options, setOptions] = useState<CatalogOption[]>([])
    const [searching, setSearching] = useState(false)
    const [confirmed, setConfirmed] = useState(false)

    async function load() {
        setLoading(true)
        setReady(false)
        try {
            const entries = await gameConfigService.list("server")
            const entry = entries.find(
                item => item.key === NEW_CHARACTER_DEFAULTS_KEY
            )
            setConfig(
                entry
                    ? parseNewCharacterDefaults(entry.value)
                    : structuredClone(INITIAL_CHARACTER_DEFAULTS)
            )
            setConfirmed(false)
            setReady(true)
        } catch (error) {
            toast.error(
                error instanceof Error
                    ? error.message
                    : "Không tải được gói quà khởi tạo."
            )
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        void load()
    }, [])

    function change(next: NewCharacterDefaults) {
        setConfig(next)
        setConfirmed(false)
    }

    async function lookup() {
        setSearching(true)
        try {
            const result = await fetchAPI<{
                ok: boolean
                items: CatalogOption[]
                message?: string
            }>(
                `/box-items/award-options?type=${kind}&petOnly=1&search=${encodeURIComponent(search)}&limit=20`
            )
            if (!result.ok)
                throw new Error(
                    result.message || "Không tra cứu được danh mục."
                )
            setOptions(result.items)
        } catch (error) {
            toast.error(
                error instanceof Error
                    ? error.message
                    : "Không tra cứu được danh mục."
            )
        } finally {
            setSearching(false)
        }
    }

    function add(option: CatalogOption) {
        if (kind === "12") {
            if (config.pets.reduce((sum, pet) => sum + pet.quantity, 0) >= 6) {
                toast.error("Gói quà đã đủ 6 ô thú cưng ban đầu.")
                return
            }
            change({
                ...config,
                pets: [
                    ...config.pets,
                    {
                        templateId: option.id,
                        quantity: 1,
                        bound: true,
                        growth: 1.5,
                    },
                ],
            })
        } else {
            if (config.items.length >= 90) {
                toast.error(
                    "Gói quà đã đủ 90 mục vật phẩm. Mỗi loại túi chỉ có 30 ô; máy chủ còn kiểm tra số ô thực tế theo số lượng và khả năng xếp chồng."
                )
                return
            }
            change({
                ...config,
                items: [
                    ...config.items,
                    {
                        templateTableId: Number(kind) as 19 | 29,
                        templateId: option.id,
                        quantity: 1,
                        bound: true,
                        color: 0,
                    },
                ],
            })
        }
        toast.success(`Đã thêm ${option.name} vào bản nháp, chưa lưu.`)
    }

    async function save() {
        if (!ready || !confirmed) return
        setBusy(true)
        try {
            const value = parseNewCharacterDefaults(config)
            const entry = await gameConfigService.upsert("server", {
                key: NEW_CHARACTER_DEFAULTS_KEY,
                value,
                description:
                    "Cấp, tiền và quà cho nhân vật mới; không áp dụng hồi tố",
            })
            setConfig(parseNewCharacterDefaults(entry.value))
            setConfirmed(false)
            toast.success("Đã lưu gói khởi tạo cho nhân vật mới.")
        } catch (error) {
            toast.error(
                error instanceof Error
                    ? error.message
                    : "Không lưu được gói khởi tạo."
            )
        } finally {
            setBusy(false)
        }
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Phần thưởng khi tạo nhân vật mới</CardTitle>
                <CardDescription>
                    Cấp, tiền và toàn bộ quà được áp dụng một lần khi tạo nhân
                    vật. Không phát quà cho nhân vật cũ hay khi đăng nhập lại.
                </CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
                {loading ? (
                    <p>Đang tải cấu hình…</p>
                ) : !ready ? (
                    <div className="space-y-3">
                        <p className="text-sm text-muted-foreground">
                            Không đọc được cấu hình hợp lệ. Tải lại trước; chỉ
                            soạn gói thay thế nếu bạn chủ động muốn sửa cấu
                            hình. Bản nháp không được tự động lưu.
                        </p>
                        <div className="flex flex-wrap gap-2">
                            <Button onClick={() => void load()}>
                                Thử tải lại cấu hình
                            </Button>
                            <Button
                                variant="outline"
                                onClick={() => {
                                    setConfig(
                                        structuredClone(
                                            INITIAL_CHARACTER_DEFAULTS
                                        )
                                    )
                                    setConfirmed(false)
                                    setReady(true)
                                }}
                            >
                                Soạn gói thay thế từ mặc định
                            </Button>
                        </div>
                    </div>
                ) : (
                    <fieldset disabled={busy} className="space-y-5">
                        <div className="grid gap-4 md:grid-cols-3">
                            {(
                                [
                                    ["level", "Cấp khởi tạo", 1, 160],
                                    [
                                        "gold",
                                        "Vàng",
                                        0,
                                        Number.MAX_SAFE_INTEGER,
                                    ],
                                    [
                                        "goldBind",
                                        "Kim phiếu (vàng khóa)",
                                        0,
                                        Number.MAX_SAFE_INTEGER,
                                    ],
                                ] as const
                            ).map(([key, label, min, max]) => (
                                <div key={key} className="space-y-2">
                                    <Label htmlFor={`starter-${key}`}>
                                        {label}
                                    </Label>
                                    <Input
                                        id={`starter-${key}`}
                                        type="number"
                                        min={min}
                                        max={max}
                                        step={1}
                                        value={config[key]}
                                        onChange={event =>
                                            change({
                                                ...config,
                                                [key]: Number(
                                                    event.target.value
                                                ),
                                            })
                                        }
                                    />
                                </div>
                            ))}
                        </div>
                        <div className="space-y-3 rounded-md border p-3">
                            <Label>
                                Chọn vật phẩm, trang bị hoặc thú cưng từ danh
                                mục
                            </Label>
                            <div className="flex flex-wrap gap-2">
                                <select
                                    aria-label="Loại quà"
                                    disabled={searching}
                                    className="rounded-md border bg-background px-3 py-2 text-sm"
                                    value={kind}
                                    onChange={event => {
                                        setKind(event.target.value)
                                        setOptions([])
                                    }}
                                >
                                    <option value="29">Vật phẩm</option>
                                    <option value="19">Trang bị</option>
                                    <option value="12">Thú cưng</option>
                                </select>
                                <Input
                                    className="min-w-40 flex-1"
                                    aria-label="Tên hoặc mã quà"
                                    placeholder="Tìm theo tên hoặc mã"
                                    value={search}
                                    onChange={event =>
                                        setSearch(event.target.value)
                                    }
                                />
                                <Button
                                    type="button"
                                    onClick={() => void lookup()}
                                    disabled={searching}
                                >
                                    {searching ? "Đang tìm…" : "Tra cứu"}
                                </Button>
                            </div>
                            {options.map(option => (
                                <div
                                    key={option.id}
                                    className="flex items-center justify-between gap-3 border-t py-2 text-sm"
                                >
                                    <span>
                                        {option.name}{" "}
                                        <span className="text-muted-foreground">
                                            (mã {option.id})
                                        </span>
                                    </span>
                                    <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        onClick={() => add(option)}
                                    >
                                        Thêm vào gói
                                    </Button>
                                </div>
                            ))}
                        </div>
                        <div className="space-y-3">
                            <h3 className="font-medium">
                                Vật phẩm và trang bị ({config.items.length} mục)
                            </h3>
                            <p className="text-sm text-muted-foreground">Trang bị dùng phẩm chất 0–5 (0 chọn theo mẫu); nguyên liệu dùng 0–4; vật phẩm khác dùng 0. Trạng thái khóa bắt buộc theo danh mục không thể bỏ.</p>
                            {config.items.length === 0 && (
                                <p className="text-sm text-muted-foreground">
                                    Chưa có quà vật phẩm bổ sung.
                                </p>
                            )}
                            {config.items.map((item, index) => (
                                <div
                                    key={index}
                                    className="grid items-end gap-3 rounded-md border p-3 sm:grid-cols-5"
                                >
                                    <p className="text-sm">
                                        {item.templateTableId === 19
                                            ? "Trang bị"
                                            : "Vật phẩm"}{" "}
                                        #{item.templateId}
                                    </p>
                                    <div>
                                        <Label>Số lượng</Label>
                                        <Input
                                            aria-label={`Số lượng vật phẩm ${index + 1}`}
                                            type="number"
                                            min={1}
                                            step={1}
                                            value={item.quantity}
                                            onChange={event =>
                                                change({
                                                    ...config,
                                                    items: config.items.map(
                                                        (row, i) =>
                                                            i === index
                                                                ? {
                                                                      ...row,
                                                                      quantity:
                                                                          Number(
                                                                              event
                                                                                  .target
                                                                                  .value
                                                                          ),
                                                                  }
                                                                : row
                                                    ),
                                                })
                                            }
                                        />
                                    </div>
                                    <div>
                                        <Label>Phẩm chất (0–5)</Label>
                                        <Input
                                            aria-label={`Phẩm chất vật phẩm ${index + 1}`}
                                            type="number"
                                            min={0}
                                            max={5}
                                            step={1}
                                            value={item.color}
                                            onChange={event =>
                                                change({
                                                    ...config,
                                                    items: config.items.map(
                                                        (row, i) =>
                                                            i === index
                                                                ? {
                                                                      ...row,
                                                                      color: Number(
                                                                          event
                                                                              .target
                                                                              .value
                                                                      ),
                                                                  }
                                                                : row
                                                    ),
                                                })
                                            }
                                        />
                                    </div>
                                    <label className="flex items-center gap-2 text-sm">
                                        <input
                                            type="checkbox"
                                            checked={item.bound}
                                            onChange={event =>
                                                change({
                                                    ...config,
                                                    items: config.items.map(
                                                        (row, i) =>
                                                            i === index
                                                                ? {
                                                                      ...row,
                                                                      bound: event
                                                                          .target
                                                                          .checked,
                                                                  }
                                                                : row
                                                    ),
                                                })
                                            }
                                        />
                                        Khóa
                                    </label>
                                    <Button
                                        variant="outline"
                                        type="button"
                                        onClick={() =>
                                            change({
                                                ...config,
                                                items: config.items.filter(
                                                    (_, i) => i !== index
                                                ),
                                            })
                                        }
                                    >
                                        Bỏ khỏi gói
                                    </Button>
                                </div>
                            ))}
                        </div>
                        <div className="space-y-3">
                            <h3 className="font-medium">
                                Thú cưng (
                                {config.pets.reduce(
                                    (sum, pet) => sum + pet.quantity,
                                    0
                                )}{" "}
                                / 6 ô)
                            </h3>
                            {config.pets.length === 0 && (
                                <p className="text-sm text-muted-foreground">
                                    Chưa có quà thú cưng bổ sung.
                                </p>
                            )}
                            {config.pets.map((pet, index) => (
                                <div
                                    key={index}
                                    className="grid items-end gap-3 rounded-md border p-3 sm:grid-cols-5"
                                >
                                    <p className="text-sm">
                                        Thú cưng #{pet.templateId}
                                    </p>
                                    <div>
                                        <Label>Số lượng</Label>
                                        <Input
                                            aria-label={`Số lượng thú cưng ${index + 1}`}
                                            type="number"
                                            min={1}
                                            max={6}
                                            step={1}
                                            value={pet.quantity}
                                            onChange={event =>
                                                change({
                                                    ...config,
                                                    pets: config.pets.map(
                                                        (row, i) =>
                                                            i === index
                                                                ? {
                                                                      ...row,
                                                                      quantity:
                                                                          Number(
                                                                              event
                                                                                  .target
                                                                                  .value
                                                                          ),
                                                                  }
                                                                : row
                                                    ),
                                                })
                                            }
                                        />
                                    </div>
                                    <div>
                                        <Label>Trưởng thành (vd. 1,9)</Label>
                                        <Input
                                            aria-label={`Trưởng thành thú cưng ${index + 1}`}
                                            type="number"
                                            min={0.1}
                                            max={10}
                                            step={0.1}
                                            value={pet.growth}
                                            onChange={event =>
                                                change({
                                                    ...config,
                                                    pets: config.pets.map(
                                                        (row, i) =>
                                                            i === index
                                                                ? {
                                                                      ...row,
                                                                      growth: Number(
                                                                          event
                                                                              .target
                                                                              .value
                                                                      ),
                                                                  }
                                                                : row
                                                    ),
                                                })
                                            }
                                        />
                                    </div>
                                    <label className="flex items-center gap-2 text-sm">
                                        <input
                                            type="checkbox"
                                            checked={pet.bound}
                                            onChange={event =>
                                                change({
                                                    ...config,
                                                    pets: config.pets.map(
                                                        (row, i) =>
                                                            i === index
                                                                ? {
                                                                      ...row,
                                                                      bound: event
                                                                          .target
                                                                          .checked,
                                                                  }
                                                                : row
                                                    ),
                                                })
                                            }
                                        />
                                        Khóa
                                    </label>
                                    <Button
                                        variant="outline"
                                        type="button"
                                        onClick={() =>
                                            change({
                                                ...config,
                                                pets: config.pets.filter(
                                                    (_, i) => i !== index
                                                ),
                                            })
                                        }
                                    >
                                        Bỏ khỏi gói
                                    </Button>
                                </div>
                            ))}
                        </div>
                        <p className="rounded-md border bg-muted/30 p-3 text-sm">
                            Túi ban đầu có giới hạn sức chứa. Máy chủ kiểm tra
                            toàn bộ gói trước khi tạo; cấu hình sai hoặc vượt
                            sức chứa sẽ từ chối, không bỏ quà hay phát một phần.
                        </p>
                        <label className="flex items-center gap-2 text-sm">
                            <input
                                type="checkbox"
                                checked={confirmed}
                                onChange={event =>
                                    setConfirmed(event.target.checked)
                                }
                            />
                            Tôi xác nhận gói này chỉ áp dụng cho các nhân vật
                            tạo sau khi lưu.
                        </label>
                        <Button
                            type="button"
                            onClick={() => void save()}
                            disabled={!confirmed || busy}
                        >
                            {busy ? "Đang lưu…" : "Lưu gói khởi tạo"}
                        </Button>
                    </fieldset>
                )}
            </CardContent>
        </Card>
    )
}
