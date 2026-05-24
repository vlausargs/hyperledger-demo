<script lang="ts">
  let productId = $state('');
  let result: any = $state(null);
  let error = $state('');
  let loading = $state(false);

  async function trace() {
    if (!productId.trim()) return;
    loading = true;
    error = '';
    try {
      const res = await fetch(`/api/trace/${productId}`);
      if (!res.ok) throw new Error((await res.json()).error);
      result = await res.json();
    } catch (e: any) {
      error = e.message;
      result = null;
    } finally {
      loading = false;
    }
  }
</script>

<h1 class="text-2xl font-bold text-gray-900 mb-6">Product Trace</h1>

<div class="max-w-2xl">
  <div class="flex gap-2 mb-6">
    <input bind:value={productId} placeholder="Enter Product ID"
      class="flex-1 rounded border-gray-300 shadow-sm px-3 py-2 border focus:border-blue-500 focus:ring-blue-500" />
    <button onclick={trace} disabled={loading}
      class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">
      {loading ? 'Tracing...' : 'Trace'}
    </button>
  </div>

  {#if error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-4">{error}</div>
  {/if}

  {#if result}
    <div class="bg-white rounded-lg shadow p-6 space-y-4">
      <h2 class="text-lg font-semibold">Product: {result.product?.name}</h2>
      <p class="text-sm text-gray-600">Status: {result.product?.status}</p>
      <p class="text-sm text-gray-600">Owner: {result.product?.currentOwner}</p>

      {#if result.history?.length}
        <h3 class="text-md font-semibold mt-4">History ({result.history.length} entries)</h3>
        <ul class="space-y-1 text-sm">
          {#each result.history as entry}
            <li class="text-gray-600">TX: {entry.txId?.slice(0, 12)}... @ {new Date(entry.timestamp).toLocaleString()}</li>
          {/each}
        </ul>
      {/if}

      {#if result.custody?.length}
        <h3 class="text-md font-semibold mt-4">Custody Chain ({result.custody.length} transfers)</h3>
        <ul class="space-y-1 text-sm">
          {#each result.custody as c}
            <li class="text-gray-600">{c.fromName} &rarr; {c.toName} ({c.status})</li>
          {/each}
        </ul>
      {/if}
    </div>
  {/if}
</div>
