export const EQUIPMENT_QUALITY_OPTIONS = Array.from(
    { length: 26 },
    (_, quality) => ({
        quality,
        label: equipmentQualityLabel(quality),
    })
)

export function equipmentQualityLabel(quality: number): string {
    if (quality === 0) return "0 — Trắng, không tiền tố"
    if (quality <= 5) return `${quality} — Trắng`
    if (quality <= 10) return `${quality} — Xanh lục`
    if (quality <= 15) return `${quality} — Xanh dương`
    if (quality <= 20) return `${quality} — Tím`
    return `${quality} — Cam`
}
