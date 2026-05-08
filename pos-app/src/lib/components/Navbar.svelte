<script>
  import { token, cashierName, clearCart } from '$lib/stores.js';
  import { goto } from '$app/navigation';

  function logout() {
    token.set(null);
    clearCart();
    goto('/login');
  }
</script>

<nav>
  <div class="brand">HLF POS</div>
  <div class="links">
    <a href="/pos">Checkout</a>
    <a href="/inventory">Inventory</a>
    <a href="/sales">Sales</a>
  </div>
  <div class="user">
    <span>{$cashierName || 'Cashier'}</span>
    <button on:click={logout}>Logout</button>
  </div>
</nav>

<style>
  nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 1rem;
    background: #1a1a2e;
    color: white;
  }
  .brand { font-weight: bold; font-size: 1.2rem; }
  .links { display: flex; gap: 1rem; }
  .links a { color: #ccc; text-decoration: none; }
  .links a:hover { color: white; }
  .user { display: flex; align-items: center; gap: 0.5rem; }
  button {
    background: #e74c3c;
    color: white;
    border: none;
    padding: 0.25rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
  }
</style>
