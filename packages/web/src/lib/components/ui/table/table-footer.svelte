<script lang="ts">
  import { cn } from "$lib/utils.js";

  interface Props {
    class?: string;
    children?: import("svelte").Snippet;
  }

  let { class: className = "", children }: Props = $props();
</script>

<tfoot class={cn("border-t bg-muted/50 font-medium [&>tr]:last:border-b-0", className)}>
  {@render children?.()}
</tfoot>
