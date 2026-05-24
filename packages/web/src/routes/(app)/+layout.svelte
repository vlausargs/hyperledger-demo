<script lang="ts">
  import { page } from '$app/stores';

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

    return items;
  });
</script>

<div class="flex h-screen bg-gray-100">
  <aside class="w-64 bg-white shadow-md flex flex-col">
    <div class="p-4 border-b">
      <h1 class="text-lg font-bold text-gray-800">Supply Chain</h1>
      <p class="text-sm text-gray-500">{orgLabel[data.user.org] || data.user.org}</p>
      <p class="text-xs text-gray-400">{data.user.username}</p>
    </div>
    <nav class="flex-1 p-4 space-y-1">
      {#each navItems as item}
        <a href={item.href}
          class="block px-3 py-2 rounded text-sm {$page.url.pathname.startsWith(item.href) ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-600 hover:bg-gray-50'}">
          {item.label}
        </a>
      {/each}
    </nav>
    <div class="p-4 border-t">
      <form method="POST" action="/logout">
        <button type="submit" class="text-sm text-gray-500 hover:text-gray-700">Sign out</button>
      </form>
    </div>
  </aside>

  <main class="flex-1 overflow-auto p-6">
    {@render children()}
  </main>
</div>
