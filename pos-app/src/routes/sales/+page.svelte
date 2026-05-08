<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';

  let sales = [];
  let loading = true;
  let error = '';
  let search = '';
  let bookmark = '';
  let hasMore = false;

  async function load(reset = false) {
    loading = true;
    error = '';
    try {
      if (search.trim()) {
        const result = await api.getSales({ customerId: search.trim() });
        sales = reset ? (result || []) : [...sales, ...(result || [])];
        hasMore = false;
      } else {
        const result = await api.getSales({ pageSize: 20, bookmark: reset ? '' : bookmark });
        const list = result.sales || [];
        sales = reset ? list : [...sales, ...list];
        bookmark = result.bookmark || '';
        hasMore = !!result.bookmark;
      }
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  onMount(() => load(true));

  function doSearch() { load(true); }
</script>

<h2>Sales History</h2>

<div class="search-row">
  <input bind:value={search} placeholder="Filter by Customer ID..." />
  <button on:click={doSearch}>Search</button>
  {#if search}<button on:click={() => { search = ''; load(true); }}>Clear</button>{/if}
</div>

{#if error}<p class="error">{error}</p>{/if}

{#if sales.length === 0 && !loading}
  <p>No sales found.</p>
{:else}
  <table>
    <thead>
      <tr>
        <th>Sale ID</th>
        <th>Customer</th>
        <th>Cashier</th>
        <th class="right">Total (IDR)</th>
        <th>Date</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each sales as s}
        <tr>
          <td>{s.id}</td>
          <td>{s.customerId}</td>
          <td>{s.cashierName}</td>
          <td class="right">{s.totalAmount?.toLocaleString('id-ID')}</td>
          <td>{new Date(s.createdAt).toLocaleString('id-ID')}</td>
          <td><a href="/sales/{s.id}">View</a></td>
        </tr>
      {/each}
    </tbody>
  </table>
  {#if hasMore && !loading}
    <button class="load-more" on:click={() => load(false)}>Load more</button>
  {/if}
{/if}
{#if loading}<p>Loading...</p>{/if}

<style>
  h2 { margin-bottom: 1rem; }
  .search-row { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
  input { flex: 1; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px; }
  button { padding: 0.5rem 1rem; background: #2c3e50; color: white; border: none; border-radius: 4px; cursor: pointer; }
  table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; }
  th { background: #1a1a2e; color: white; padding: 0.5rem 0.75rem; text-align: left; }
  td { padding: 0.5rem 0.75rem; border-bottom: 1px solid #eee; font-size: 0.9rem; }
  .right { text-align: right; }
  a { color: #2980b9; text-decoration: none; }
  .load-more { margin-top: 1rem; width: 100%; }
  .error { color: #e74c3c; }
</style>
