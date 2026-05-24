<script lang="ts">
  import { cn } from "$lib/utils.js";

  interface Props {
    class?: string;
    children?: import("svelte").Snippet;
  }

  let { class: className = "", children }: Props = $props();
</script>

<caption class={cn("mt-4 text-sm text-muted-foreground", className)}>
  {@render children?.()}
</caption>
