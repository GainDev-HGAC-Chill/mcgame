/** RFC 4122 UUID v4 for browsers on HTTP as well as HTTPS. */
export function createClientUUID(): string {
    const cryptoAPI = globalThis.crypto
    if (!cryptoAPI || typeof cryptoAPI.getRandomValues !== "function") {
        throw new Error("Trình duyệt không hỗ trợ tạo UUID an toàn.")
    }
    if (typeof cryptoAPI.randomUUID === "function") {
        return cryptoAPI.randomUUID()
    }

    const bytes = new Uint8Array(16)
    cryptoAPI.getRandomValues(bytes)
    bytes[6] = (bytes[6] & 0x0f) | 0x40
    bytes[8] = (bytes[8] & 0x3f) | 0x80
    const hex = Array.from(bytes, value => value.toString(16).padStart(2, "0")).join("")
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}
