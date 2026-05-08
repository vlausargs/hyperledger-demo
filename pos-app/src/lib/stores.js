import { writable, derived, get } from 'svelte/store';

function persisted(key, initial) {
  const stored = typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null;
  const store = writable(stored ? JSON.parse(stored) : initial);
  store.subscribe((val) => {
    if (typeof localStorage !== 'undefined') localStorage.setItem(key, JSON.stringify(val));
  });
  return store;
}

export const token = persisted('token', null);
export const cashierId = persisted('cashierId', '');
export const cashierName = persisted('cashierName', '');

export const cart = writable([]);
export const customerId = writable('');

export const cartSubTotal = derived(cart, ($cart) =>
  $cart.reduce((sum, item) => sum + item.unitPrice, 0)
);

export const taxAmount = derived(cartSubTotal, ($sub) =>
  Math.round($sub * 0.11 * 100) / 100
);

export const grandTotal = derived(
  [cartSubTotal, taxAmount],
  ([$sub, $tax]) => Math.round(($sub + $tax) * 100) / 100
);

export function addToCart(item) {
  cart.update((items) => [...items, item]);
}

export function removeFromCart(index) {
  cart.update((items) => items.filter((_, i) => i !== index));
}

export function clearCart() {
  cart.set([]);
  customerId.set('');
}

export function isLoggedIn() {
  return !!get(token);
}
