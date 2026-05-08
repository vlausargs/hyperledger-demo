<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { token } from '$lib/stores.js';
  import Navbar from '$lib/components/Navbar.svelte';

  onMount(() => {
    if (!$token && $page.url.pathname !== '/login') {
      goto('/login');
    }
  });

  $: showNav = $page.url.pathname !== '/login';
</script>

{#if showNav}
  <Navbar />
{/if}
<main>
  <slot />
</main>

<style>
  :global(body) { margin: 0; font-family: system-ui, sans-serif; background: #f5f5f5; }
  main { padding: 1rem; max-width: 1100px; margin: 0 auto; }
</style>
