import { defineConfig } from 'vitest/config';
import path from 'node:path';

// Standalone Vitest config — intentionally does NOT load the SvelteKit or
// vite-plugin-svelte plugins. The bundled Vite that ships with Vitest 2 is
// pinned to v5, which is incompatible with the project's vite-plugin-svelte
// v7 (which expects Vite 8). Adding component tests later requires either
// pinning the plugin to a v5-compatible release or running them under
// `vite-node` separately.
//
// We test plain TypeScript modules (`$lib` utilities, API client wrapper,
// auth helpers) which need only jsdom + path aliases.
export default defineConfig({
  resolve: {
    alias: {
      $lib: path.resolve(__dirname, 'src/lib'),
      $app: path.resolve(__dirname, 'src/lib/test-stubs/app')
    }
  },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.{test,spec}.{js,ts}'],
    exclude: ['node_modules', 'build', '.svelte-kit'],
    setupFiles: ['./vitest-setup.ts']
  }
});
