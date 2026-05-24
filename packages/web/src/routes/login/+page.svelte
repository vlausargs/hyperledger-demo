<script lang="ts">
  import { enhance } from '$app/forms';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import * as Card from '$lib/components/ui/card/index.js';

  let error = $state('');
</script>

<div class="min-h-screen flex items-center justify-center bg-muted/40">
  <Card.Card class="w-full max-w-md">
    <Card.Header class="space-y-1 text-center">
      <Card.Title class="text-2xl font-bold">HLF Supply Chain</Card.Title>
      <Card.Description>Sign in to your account</Card.Description>
    </Card.Header>
    <Card.Content>
      {#if error}
        <div class="mb-4 rounded-md bg-destructive/10 border border-destructive/20 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      {/if}

      <form method="POST" use:enhance={() => {
        return async ({ result }) => {
          if (result.type === 'failure') {
            error = result.data?.error || 'Login failed';
          } else if (result.type === 'redirect') {
            window.location.href = result.location;
          }
        };
      }}>
        <div class="space-y-4">
          <div class="space-y-2">
            <Label for="username">Username</Label>
            <Input id="username" name="username" type="text" required placeholder="Enter your username" />
          </div>
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input id="password" name="password" type="password" required placeholder="Enter your password" />
          </div>
          <Button type="submit" class="w-full">Sign in</Button>
        </div>
      </form>
    </Card.Content>
  </Card.Card>
</div>
