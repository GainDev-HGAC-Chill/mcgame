export class SameOriginError extends Error {}

export function requireSameOrigin(request: Pick<Request, "url" | "headers">) {
    const origin = request.headers.get("Origin")
    const fetchSite = request.headers.get("Sec-Fetch-Site")
    let valid = false
    try {
        const parsed = new URL(origin ?? "")
        const allowed = (process.env.DASHBOARD_ALLOWED_ORIGINS ?? "")
            .split(",")
            .map(value => value.trim())
            .filter(Boolean)
        valid =
            parsed.origin === origin &&
            (parsed.protocol === "http:" || parsed.protocol === "https:") &&
            (origin === new URL(request.url).origin ||
                allowed.includes(origin!))
    } catch {
        valid = false
    }
    if (!valid || (fetchSite !== null && fetchSite !== "same-origin")) {
        throw new SameOriginError("Yêu cầu phải đến từ cùng nguồn.")
    }
}
