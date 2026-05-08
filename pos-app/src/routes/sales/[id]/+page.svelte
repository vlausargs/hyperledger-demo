<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';

  let sale = null;
  let loading = true;
  let error = '';

  onMount(async () => {
    try {
      sale = await api.getSale($page.params.id);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });
</script>

<a href="/sales">← Back to Sales</a>
<h2>Sale Detail</h2>

{#if loading}<p>Loading...</p>
{:else if error}<p class="error">{error}</p>
{:else if sale}
  <div class="card">
    <div class="meta">
      <div><strong>Sale ID:</strong> {sale.id}</div>
      <div><strong>Tx ID:</strong> <span class="mono">{sale.txId}</span></div>
      <div><strong>Customer:</strong> {sale.customerId}</div>
      <div><strong>Cashier:</strong> {sale.cashierName} ({sale.cashierId})</div>
      <div><strong>Date:</strong> {new Date(sale.createdAt).toLocaleString('id-ID')}</div>
      <div><strong>Retailer MSP:</strong> {sale.retailerMsp}</div>
    </div>

    <h3>Items</h3>
    <table>
      <thead>
        <tr><th>Product ID</th><th>SKU</th><th>Name</th><th class="right">Price</th></tr>
      </thead>
      <tbody>
        {#each sale.items as item}
          <tr>
            <td>{item.productId}</td>
            <td>{item.sku}</td>
            <td>{item.productName}</td>
            <td class="right">{item.unitPrice.toLocaleString('id-ID')}</td>
          </tr>
        {/each}
      </tbody>
    </table>

    <div class="totals">
      <div><span>Subtotal</span><span>{sale.subTotal?.toLocaleString('id-ID')}</span></div>
      <div><span>Tax</span><span>{sale.taxAmount?.toLocaleString('id-ID')}</span></div>
      <div class="grand"><span>Total ({sale.currency})</span><span>{sale.totalAmount?.toLocaleString('id-ID')}</span></div>
    </div>

    {#if sale.notes}
      <p><strong>Notes:</strong> {sale.notes}</p>
    {/if}
  </div>
{/if}

<style>
  h2 { margin: 0.5rem 0 1rem; }
  .card { background: white; border-radius: 8px; padding: 1.5rem; max-width: 700px; }
  .meta { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; margin-bottom: 1rem; font-size: 0.9rem; }
  .mono { font-family: monospace; font-size: 0.8rem; word-break: break-all; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 1rem; }
  th { background: #f0f0f0; padding: 0.4rem 0.5rem; text-align: left; font-size: 0.85rem; }
  td { padding: 0.4rem 0.5rem; border-bottom: 1px solid #eee; font-size: 0.9rem; }
  .right { text-align: right; }
  .totals { border-top: 2px solid #ddd; padding-top: 0.5rem; display: flex; flex-direction: column; gap: 0.25rem; max-width: 280px; margin-left: auto; }
  .totals div { display: flex; justify-content: space-between; font-size: 0.9rem; }
  .grand { font-weight: bold; font-size: 1rem; }
  .error { color: #e74c3c; }
  a { color: #2980b9; text-decoration: none; }
</style>
