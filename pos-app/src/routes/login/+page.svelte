<script>
  import { goto } from '$app/navigation';
  import { token, cashierId, cashierName } from '$lib/stores.js';
  import { api } from '$lib/api.js';

  let username = '';
  let password = '';
  let error = '';
  let loading = false;

  async function login() {
    if (!username || !password) return;
    loading = true;
    error = '';
    try {
      const data = await api.login(username, password);
      token.set(data.token);
      cashierId.set(username);
      cashierName.set(data.name || username);
      goto('/pos');
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-wrap">
  <div class="login-box">
    <h2>HLF POS Login</h2>
    <label>Username
      <input bind:value={username} type="text" autocomplete="username" />
    </label>
    <label>Password
      <input bind:value={password} type="password" autocomplete="current-password"
        on:keydown={(e) => e.key === 'Enter' && login()} />
    </label>
    {#if error}<p class="error">{error}</p>{/if}
    <button on:click={login} disabled={loading}>
      {loading ? 'Logging in...' : 'Login'}
    </button>
  </div>
</div>

<style>
  .login-wrap {
    display: flex; align-items: center; justify-content: center;
    min-height: 80vh;
  }
  .login-box {
    background: white; padding: 2rem; border-radius: 8px;
    box-shadow: 0 2px 12px rgba(0,0,0,0.1); min-width: 300px;
  }
  h2 { margin-top: 0; text-align: center; }
  label { display: flex; flex-direction: column; gap: 0.25rem; margin-bottom: 1rem; font-size: 0.9rem; }
  input { padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px; font-size: 1rem; }
  button {
    width: 100%; padding: 0.75rem;
    background: #1a1a2e; color: white;
    border: none; border-radius: 4px; cursor: pointer; font-size: 1rem;
  }
  button:disabled { opacity: 0.5; }
  .error { color: #e74c3c; font-size: 0.85rem; }
</style>
