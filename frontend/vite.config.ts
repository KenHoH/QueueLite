import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { existsSync, realpathSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL(".", import.meta.url));
const needsWindowsTildeFix = process.platform === "win32" && root.includes("~");

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    // Vite's Windows short-name guard also rejects literal tildes in folder names.
    // Preserve a frontend-only boundary when replacing that guard locally.
    ...(needsWindowsTildeFix
      ? [
          {
            name: "frontend-only-windows-paths",
            configureServer(server: import("vite").ViteDevServer) {
              const canonicalRoot = realpathSync(root);
              server.middlewares.use((req, res, next) => {
                try {
                  const url = decodeURIComponent(
                    (req.url || "/").split("?")[0],
                  ).replaceAll("\\", "/");
                  const requested = url.startsWith("/@fs/")
                    ? url.slice(5)
                    : path.resolve(root, `.${url}`);
                  const canonical = existsSync(requested)
                    ? realpathSync(requested)
                    : path.resolve(requested);
                  const relative = path.relative(canonicalRoot, canonical);
                  const hidden = relative
                    .split(/[\\/]/)
                    .some((part) => part.startsWith(".") && part !== ".vite");
                  if (
                    relative.startsWith("..") ||
                    path.isAbsolute(relative) ||
                    hidden ||
                    /\.(pem|crt)$/i.test(relative)
                  ) {
                    res.statusCode = 403;
                    res.end("Only public frontend files are served.");
                    return;
                  }
                  next();
                } catch {
                  res.statusCode = 400;
                  res.end("Invalid request path.");
                }
              });
            },
          },
        ]
      : []),
  ],
  server: { host: "127.0.0.1", fs: { strict: !needsWindowsTildeFix } },
  preview: { host: "127.0.0.1" },
});
