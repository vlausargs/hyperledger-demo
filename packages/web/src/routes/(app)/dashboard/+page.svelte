<script lang="ts">
  import * as Card from '$lib/components/ui/card/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';

  let { data } = $props();

  function productVariant(status: string) {
    if (status === 'ACTIVE') return 'default' as const;
    if (status === 'SHIPPED') return 'secondary' as const;
    if (status === 'DELIVERED') return 'outline' as const;
    return 'secondary' as const;
  }

  function shipmentVariant(status: string) {
    if (status === 'DRAFT') return 'secondary' as const;
    if (status === 'IN_TRANSIT') return 'default' as const;
    if (status === 'DELIVERED') return 'outline' as const;
    return 'secondary' as const;
  }
</script>

<h1 class="text-2xl font-bold text-foreground mb-6">Dashboard</h1>

<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
  <Card.Card>
    <Card.Header>
      <Card.Title>Recent Products</Card.Title>
    </Card.Header>
    <Card.Content>
      {#if data.products.products?.length}
        <ul class="space-y-2">
          {#each data.products.products as product}
            <li class="flex justify-between items-center text-sm">
              <span class="text-foreground">{product.name}</span>
              <Badge variant={productVariant(product.status)}>{product.status}</Badge>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="text-muted-foreground text-sm">No products yet</p>
      {/if}
    </Card.Content>
  </Card.Card>

  <Card.Card>
    <Card.Header>
      <Card.Title>Recent Shipments</Card.Title>
    </Card.Header>
    <Card.Content>
      {#if data.shipments.shipments?.length}
        <ul class="space-y-2">
          {#each data.shipments.shipments as shipment}
            <li class="flex justify-between items-center text-sm">
              <span class="text-foreground">{shipment.name}</span>
              <Badge variant={shipmentVariant(shipment.status)}>{shipment.status}</Badge>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="text-muted-foreground text-sm">No shipments yet</p>
      {/if}
    </Card.Content>
  </Card.Card>
</div>
