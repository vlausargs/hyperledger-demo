<script lang="ts">
  import { cn } from "$lib/utils.js";

  interface Props {
    class?: string;
    children?: import("svelte").Snippet;
  }

  let { class: className = "", children }: Props = $props();
</script>

<div class={cn("rounded-xl border bg-card text-card-foreground shadow", className)}>
  {@render children?.()}
</div>
