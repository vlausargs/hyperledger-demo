<script>
  import { createEventDispatcher } from 'svelte';
  import { api } from '$lib/api.js';

  const dispatch = createEventDispatcher();

  let productId = '';
  let loading = false;
  let error = '';

  async function scan() {
    if (!productId.trim()) return;
    loading = true;
    error = '';
    try {
      const data = await api.verifyProduct(productId.trim());
      dispatch('verified', { id: productId.trim(), provenance: data });
      productId = '';
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function onKeydown(e) {
    if (e.key === 'Enter') scan();
  }
</script>

<div class="scan-input">
  <input
    bind:value={productId}
    on:keydown={onKeydown}
    placeholder="Scan or enter product ID..."
    disabled={loading}
  />
  <button on:click={scan} disabled={loading || !productId.trim()}>
    {loading ? '...' : 'Verify'}
  </button>
  {#if error}
    <span class="error">{error}</span>
  {/if}
</div>

<style>
  .scan-input { display: flex; gap: 0.5rem; align-items: center; flex-wrap: wrap; }
  input { flex: 1; padding: 0.5rem; font-size: 1rem; border: 1px solid #ccc; border-radius: 4px; }
  button {
    padding: 0.5rem 1rem;
    background: #2c3e50;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }
  button:disabled { opacity: 0.5; cursor: not-allowed; }
  .error { color: #e74c3c; font-size: 0.85rem; width: 100%; }
</style>
