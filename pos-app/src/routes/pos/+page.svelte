<script>
  import { nanoid } from 'nanoid';
  import { cart, customerId, cartSubTotal, taxAmount, grandTotal, clearCart, cashierId, cashierName } from '$lib/stores.js';
  import { api } from '$lib/api.js';
  import ScanInput from '$lib/components/ScanInput.svelte';
  import VerifyModal from '$lib/components/VerifyModal.svelte';
  import Cart from '$lib/components/Cart.svelte';

  let modalOpen = false;
  let verifiedProductId = '';
  let provenance = null;

  let checkoutLoading = false;
  let checkoutError = '';
  let receipt = null;

  function onVerified(e) {
    verifiedProductId = e.detail.id;
    provenance = e.detail.provenance;
    modalOpen = true;
  }

  async function checkout() {
    if ($cart.length === 0) return;
    if (!$customerId.trim()) { checkoutError = 'Customer ID required'; return; }

    checkoutLoading = true;
    checkoutError = '';
    try {
      const saleId = 'SALE-' + nanoid(10).toUpperCase();
      const payload = {
        id: saleId,
        customerId: $customerId.trim(),
        cashierId: $cashierId,
        cashierName: $cashierName,
        items: $cart.map(i => ({
          productId: i.productId,
          sku: i.sku,
          productName: i.productName,
          unitPrice: i.unitPrice
        })),
        taxAmount: $taxAmount,
        currency: 'IDR'
      };
      const result = await api.createSale(payload);
      receipt = { id: result.id || saleId, ...result };
      clearCart();
    } catch (e) {
      checkoutError = e.message;
    } finally {
      checkoutLoading = false;
    }
  }
</script>

<h2>Point of Sale — Checkout</h2>

<ScanInput on:verified={onVerified} />

<VerifyModal
  bind:open={modalOpen}
  productId={verifiedProductId}
  {provenance}
  on:close={() => (modalOpen = false)}
/>

<div class="layout">
  <div class="left">
    <Cart />
    <div class="customer-row">
      <label>Customer ID
        <input bind:value={$customerId} placeholder="Enter customer ID" />
      </label>
    </div>
    {#if checkoutError}<p class="error">{checkoutError}</p>{/if}
    <button class="checkout-btn" on:click={checkout}
      disabled={checkoutLoading || $cart.length === 0}>
      {checkoutLoading ? 'Processing...' : `Checkout — Rp ${$grandTotal.toLocaleString('id-ID')}`}
    </button>
  </div>
</div>

{#if receipt}
  <div class="overlay" on:click|self={() => (receipt = null)}>
    <div class="receipt">
      <h3>Sale Complete</h3>
      <p>Sale ID: <strong>{receipt.id}</strong></p>
      <button on:click={() => (receipt = null)}>Close</button>
    </div>
  </div>
{/if}

<style>
  h2 { margin-bottom: 1rem; }
  .layout { margin-top: 1rem; }
  .left { max-width: 600px; display: flex; flex-direction: column; gap: 0.75rem; }
  .customer-row label { display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.9rem; }
  input { padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px; font-size: 1rem; }
  .checkout-btn {
    padding: 0.75rem; background: #27ae60; color: white;
    border: none; border-radius: 4px; cursor: pointer; font-size: 1rem; font-weight: bold;
  }
  .checkout-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .error { color: #e74c3c; font-size: 0.85rem; }
  .overlay {
    position: fixed; inset: 0; background: rgba(0,0,0,0.5);
    display: flex; align-items: center; justify-content: center; z-index: 200;
  }
  .receipt {
    background: white; border-radius: 8px; padding: 2rem; text-align: center; min-width: 280px;
  }
  .receipt button {
    padding: 0.5rem 1.5rem; background: #1a1a2e; color: white;
    border: none; border-radius: 4px; cursor: pointer; margin-top: 1rem;
  }
</style>
