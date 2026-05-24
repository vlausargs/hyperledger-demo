<script lang="ts">
  import { page } from '$app/stores';
  import { Button } from '$lib/components/ui/button/index.js';

  let { data, children } = $props();

  const orgLabel: Record<string, string> = {
    'Org1MSP': 'Manufacturer',
    'Org2MSP': 'Distributor',
    'Org3MSP': 'Retailer'
  };

  type NavItem = { href: string; label: string };

  const navItems: NavItem[] = $derived.by(() => {
    const items: NavItem[] = [
      { href: '/dashboard', label: 'Dashboard' },
    ];

    if (data.user.org === 'Org1MSP') {
      items.push({ href: '/manufacturer/products', label: 'Products' });
      items.push({ href: '/manufacturer/shipments', label: 'Shipments' });
    }
    if (data.user.org === 'Org2MSP') {
      items.push({ href: '/distributor/receiving', label: 'Receiving' });
      items.push({ href: '/distributor/inventory', label: 'Inventory' });
      items.push({ href: '/distributor/shipments', label: 'Shipments' });
    }
    if (data.user.org === 'Org3MSP') {
      items.push({ href: '/retailer/pos', label: 'POS' });
      items.push({ href: '/retailer/inventory', label: 'Inventory' });
      items.push({ href: '/retailer/sales', label: 'Sales' });
    }

    items.push({ href: '/trace', label: 'Trace' });
    items.push({ href: '/recalls', label: 'Recalls' });
    items.push({ href: '/admin', label: 'Admin' });

    return items;
  });
</script>

<div class="flex h-screen bg-muted/40">
  <aside class="w-64 bg-card shadow-md flex flex-col border-r">
    <div class="p-4 border-b">
      <h1 class="text-lg font-bold text-foreground">Supply Chain</h1>
      <p class="text-sm text-muted-foreground">{orgLabel[data.user.org] || data.user.org}</p>
      <p class="text-xs text-muted-foreground/70">{data.user.username}</p>
    </div>
    <nav class="flex-1 p-3 space-y-1">
      {#each navItems as item}
        <a href={item.href} class="block w-full">
          <Button
            variant={$page.url.pathname.startsWith(item.href) ? 'secondary' : 'ghost'}
            class="w-full justify-start text-sm"
          >
            {item.label}
          </Button>
        </a>
      {/each}
    </nav>
    <div class="p-4 border-t">
      <form method="POST" action="/logout">
        <Button type="submit" variant="ghost" class="w-full justify-start text-sm text-muted-foreground">
          Sign out
        </Button>
      </form>
    </div>
  </aside>

  <main class="flex-1 overflow-auto p-6">
    {@render children()}
  </main>
</div>
