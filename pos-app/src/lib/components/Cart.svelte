<script>
  import { cart, cartSubTotal, taxAmount, grandTotal, removeFromCart } from '$lib/stores.js';
</script>

<div class="cart">
  <h3>Cart ({$cart.length} items)</h3>

  {#if $cart.length === 0}
    <p class="empty">No items. Scan a product to add.</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>Product</th>
          <th>SKU</th>
          <th class="right">Price</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {#each $cart as item, i}
          <tr>
            <td>{item.productName}<br/><small class="id">{item.productId}</small></td>
            <td>{item.sku}</td>
            <td class="right">{item.unitPrice.toLocaleString('id-ID')}</td>
            <td><button class="remove" on:click={() => removeFromCart(i)}>✕</button></td>
          </tr>
        {/each}
      </tbody>
    </table>

    <div class="totals">
      <div><span>Subtotal</span><span>{$cartSubTotal.toLocaleString('id-ID')}</span></div>
      <div><span>Tax (11%)</span><span>{$taxAmount.toLocaleString('id-ID')}</span></div>
      <div class="grand"><span>Total</span><span>{$grandTotal.toLocaleString('id-ID')}</span></div>
    </div>
  {/if}
</div>

<style>
  .cart { background: #f9f9f9; border: 1px solid #ddd; border-radius: 8px; padding: 1rem; }
  h3 { margin-top: 0; }
  .empty { color: #999; text-align: center; }
  table { width: 100%; border-collapse: collapse; font-size: 0.9rem; margin-bottom: 0.75rem; }
  th { text-align: left; padding: 0.25rem; border-bottom: 2px solid #ddd; }
  td { padding: 0.4rem 0.25rem; border-bottom: 1px solid #eee; vertical-align: top; }
  .right { text-align: right; }
  .id { color: #999; font-size: 0.75rem; }
  .remove { background: none; border: none; color: #e74c3c; cursor: pointer; font-size: 1rem; }
  .totals { border-top: 2px solid #ddd; padding-top: 0.5rem; display: flex; flex-direction: column; gap: 0.25rem; }
  .totals div { display: flex; justify-content: space-between; font-size: 0.9rem; }
  .grand { font-weight: bold; font-size: 1.1rem; margin-top: 0.25rem; }
</style>
