<script lang="ts">
  import { cn } from "$lib/utils.js";

  interface Props {
    class?: string;
    children?: import("svelte").Snippet;
  }

  let { class: className = "", children }: Props = $props();
</script>

<div class="relative w-full overflow-auto">
  <table class={cn("w-full caption-bottom text-sm", className)}>
    {@render children?.()}
  </table>
</div>
