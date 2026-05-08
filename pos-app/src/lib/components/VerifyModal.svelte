<script>
  import { createEventDispatcher } from 'svelte';
  import { addToCart } from '$lib/stores.js';

  export let productId = '';
  export let provenance = null;
  export let open = false;

  const dispatch = createEventDispatcher();

  let unitPrice = '';

  $: product = provenance?.product;
  $: isRecalled = product?.status === 'RECALLED' || !!product?.recallId;
  $: isAvailable = product?.status === 'DELIVERED';

  function addItem() {
    const price = parseFloat(unitPrice);
    if (isNaN(price) || price <= 0) return;
    addToCart({
      productId,
      sku: product.sku,
      productName: product.name,
      unitPrice: price
    });
    dispatch('close');
  }

  function close() { dispatch('close'); }
</script>

{#if open && product}
  <div class="overlay" on:click|self={close}>
    <div class="modal">
      <h3>Product Verification</h3>

      {#if isRecalled}
        <div class="alert error">RECALLED — Cannot sell. Recall ID: {product.recallId}</div>
      {:else if !isAvailable}
        <div class="alert warn">Status: {product.status} — Not available for sale</div>
      {:else}
        <div class="alert ok">Verified — Available for sale</div>
      {/if}

      <table>
        <tr><th>ID</th><td>{product.id}</td></tr>
        <tr><th>SKU</th><td>{product.sku}</td></tr>
        <tr><th>Name</th><td>{product.name}</td></tr>
        <tr><th>Batch</th><td>{product.batchId}</td></tr>
        <tr><th>Manufacturer</th><td>{product.manufacturerName}</td></tr>
        <tr><th>Owner</th><td>{product.currentOwnerMSP}</td></tr>
        <tr><th>Status</th><td>{product.status}</td></tr>
      </table>

      {#if isAvailable && !isRecalled}
        <div class="price-row">
          <label>Retail Price (IDR)
            <input type="number" bind:value={unitPrice} min="0" step="100" placeholder="0" />
          </label>
          <button on:click={addItem} disabled={!unitPrice || parseFloat(unitPrice) <= 0}>
            Add to Cart
          </button>
        </div>
      {/if}

      <button class="close-btn" on:click={close}>Close</button>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed; inset: 0;
    background: rgba(0,0,0,0.5);
    display: flex; align-items: center; justify-content: center;
    z-index: 100;
  }
  .modal {
    background: white; border-radius: 8px; padding: 1.5rem;
    min-width: 360px; max-width: 520px; width: 90%;
  }
  h3 { margin-top: 0; }
  .alert { padding: 0.5rem; border-radius: 4px; margin-bottom: 1rem; font-weight: bold; }
  .alert.error { background: #fde; color: #c00; }
  .alert.warn { background: #ffe; color: #880; }
  .alert.ok { background: #dfd; color: #060; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 1rem; font-size: 0.9rem; }
  th { text-align: left; padding: 0.25rem 0.5rem; color: #555; width: 40%; }
  td { padding: 0.25rem 0.5rem; }
  .price-row { display: flex; gap: 0.5rem; align-items: flex-end; margin-bottom: 0.75rem; }
  label { flex: 1; display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.9rem; }
  input[type=number] { padding: 0.4rem; border: 1px solid #ccc; border-radius: 4px; font-size: 1rem; }
  button { padding: 0.5rem 1rem; background: #27ae60; color: white; border: none; border-radius: 4px; cursor: pointer; }
  button:disabled { opacity: 0.5; cursor: not-allowed; }
  .close-btn { background: #95a5a6; width: 100%; margin-top: 0.5rem; }
</style>
