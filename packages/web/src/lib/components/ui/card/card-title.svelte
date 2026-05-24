<script lang="ts">
  import { cn } from "$lib/utils.js";

  interface Props {
    class?: string;
    children?: import("svelte").Snippet;
  }

  let { class: className = "", children }: Props = $props();
</script>

<div class={cn("font-semibold leading-none tracking-tight", className)}>
  {@render children?.()}
</div>
