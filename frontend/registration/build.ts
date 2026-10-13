import { copyFile, mkdir } from "node:fs/promises";

await mkdir("dist/registration", { recursive: true });
await copyFile("register.html", "dist/register.html");
await copyFile("style.css", "dist/registration/style.css");
