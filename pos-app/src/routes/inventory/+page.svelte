<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';

  let items = [];
  let loading = true;
  let error = '';

  onMount(async () => {
    try {
      items = await api.getInventory('Org3MSP');
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });
</script>

<h2>Inventory (Org3 / Retailer)</h2>

{#if loading}<p>Loading...</p>
{:else if error}<p class="error">{error}</p>
{:else if items.length === 0}<p>No DELIVERED products in inventory.</p>
{:else}
  <table>
    <thead>
      <tr>
        <th>SKU</th>
        <th>Product Name</th>
        <th class="right">Available</th>
        <th>Product IDs</th>
      </tr>
    </thead>
    <tbody>
      {#each items as item}
      <tr>
        <td>{item.sku}</td>
        <td>{item.name}</td>
        <td class="right">{item.count}</td>
        <td class="ids">{item.productIds.join(', ')}</td>
      </tr>
      {/each}
    </tbody>
  </table>
{/if}

<style>
  h2 { margin-bottom: 1rem; }
  table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; }
  th { background: #1a1a2e; color: white; padding: 0.5rem 0.75rem; text-align: left; }
  td { padding: 0.5rem 0.75rem; border-bottom: 1px solid #eee; }
  tr:last-child td { border-bottom: none; }
  .right { text-align: right; }
  .ids { font-size: 0.75rem; color: #666; max-width: 300px; word-break: break-all; }
  .error { color: #e74c3c; }
</style>
