import assert from "node:assert/strict"
import { test } from "node:test"

import {
    requireSameOrigin,
    SameOriginError,
} from "../src/lib/server/same-origin.ts"

function request(
    origin,
    fetchSite = "same-origin",
    requestOrigin = "http://127.0.0.1:3000"
) {
    return new Request(`${requestOrigin}/api/box-items/awards`, {
        method: "POST",
        headers: {
            ...(origin ? { Origin: origin } : {}),
            ...(fetchSite ? { "Sec-Fetch-Site": fetchSite } : {}),
        },
    })
}

test("same-origin guard accepts only the request URL origin", () => {
    for (const origin of [
        "http://localhost:3000",
        "http://127.0.0.1:3000",
        "http://26.31.111.123:3000",
    ]) {
        assert.doesNotThrow(() =>
            requireSameOrigin(request(origin, "same-origin", origin))
        )
    }
    for (const candidate of [
        request(null),
        request("http://evil.example"),
        request("http://127.0.0.1:3000", "cross-site"),
        request("javascript:alert(1)"),
    ]) {
        assert.throws(() => requireSameOrigin(candidate), SameOriginError)
    }
})
