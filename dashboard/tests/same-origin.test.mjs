import assert from "node:assert/strict"
import { test } from "node:test"

import {
    requireSameOrigin,
    SameOriginError,
} from "../src/lib/server/same-origin.ts"

function request(origin, fetchSite = "same-origin") {
    return new Request("http://127.0.0.1:3000/api/box-items/awards", {
        method: "POST",
        headers: {
            ...(origin ? { Origin: origin } : {}),
            ...(fetchSite ? { "Sec-Fetch-Site": fetchSite } : {}),
        },
    })
}

test("same-origin guard accepts only the request URL origin", () => {
    assert.doesNotThrow(() =>
        requireSameOrigin(request("http://127.0.0.1:3000"))
    )
    for (const candidate of [
        request(null),
        request("http://evil.example"),
        request("http://127.0.0.1:3000", "cross-site"),
        request("javascript:alert(1)"),
    ]) {
        assert.throws(() => requireSameOrigin(candidate), SameOriginError)
    }
})
