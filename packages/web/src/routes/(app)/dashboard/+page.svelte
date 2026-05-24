<script lang="ts">
  let { data } = $props();
</script>

<h1 class="text-2xl font-bold text-gray-900 mb-6">Dashboard</h1>

<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
  <div class="bg-white rounded-lg shadow p-6">
    <h2 class="text-lg font-semibold text-gray-700 mb-4">Recent Products</h2>
    {#if data.products.products?.length}
      <ul class="space-y-2">
        {#each data.products.products as product}
          <li class="flex justify-between items-center text-sm">
            <span class="text-gray-800">{product.name}</span>
            <span class="px-2 py-1 text-xs rounded-full
              {product.status === 'ACTIVE' ? 'bg-green-100 text-green-700' :
               product.status === 'SHIPPED' ? 'bg-blue-100 text-blue-700' :
               product.status === 'DELIVERED' ? 'bg-purple-100 text-purple-700' :
               'bg-gray-100 text-gray-700'}">{product.status}</span>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="text-gray-500 text-sm">No products yet</p>
    {/if}
  </div>

  <div class="bg-white rounded-lg shadow p-6">
    <h2 class="text-lg font-semibold text-gray-700 mb-4">Recent Shipments</h2>
    {#if data.shipments.shipments?.length}
      <ul class="space-y-2">
        {#each data.shipments.shipments as shipment}
          <li class="flex justify-between items-center text-sm">
            <span class="text-gray-800">{shipment.name}</span>
            <span class="px-2 py-1 text-xs rounded-full
              {shipment.status === 'DRAFT' ? 'bg-gray-100 text-gray-700' :
               shipment.status === 'IN_TRANSIT' ? 'bg-yellow-100 text-yellow-700' :
               shipment.status === 'DELIVERED' ? 'bg-green-100 text-green-700' :
               'bg-gray-100 text-gray-700'}">{shipment.status}</span>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="text-gray-500 text-sm">No shipments yet</p>
    {/if}
  </div>
</div>
