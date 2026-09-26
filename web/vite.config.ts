import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => ({
  server: {
    proxy: {
      "/api":
        loadEnv(mode, ".", "STATECRAFT_").STATECRAFT_API_URL ||
        "http://127.0.0.1:8081",
    },
  },
}));
