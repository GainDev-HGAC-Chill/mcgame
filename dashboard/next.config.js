/** @type {import('next').NextConfig} */
const nextConfig = {
    reactStrictMode: true,
    output: "standalone",
    async headers() {
        return [
            {
                source: "/",
                headers: [{ key: "Cache-Control", value: "private, no-store" }],
            },
            {
                source: "/dashboard/:path*",
                headers: [{ key: "Cache-Control", value: "private, no-store" }],
            },
        ]
    },
}

module.exports = nextConfig
