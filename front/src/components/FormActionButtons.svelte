<script lang="ts">
  type ButtonVariant = 'ghost' | 'primary' | 'secondary' | 'danger'

  export let items: readonly { label: string; href?: string; variant?: ButtonVariant; onClick?: () => void }[] = []

  const variantClass: Record<ButtonVariant, string> = {
    ghost: 'admin-button',
    primary: 'admin-button admin-button-primary',
    secondary: 'admin-button admin-button-secondary',
    danger: 'admin-button admin-button-danger',
  }
</script>

<div class="flex flex-wrap gap-2 border-t border-white/10 pt-4">
  {#each items as item}
    {#if item.onClick}
      <button class={variantClass[item.variant ?? 'ghost']} type="button" on:click={item.onClick}>
        {item.label}
      </button>
    {:else}
      <a class={variantClass[item.variant ?? 'ghost']} href={item.href}>
        {item.label}
      </a>
    {/if}
  {/each}
</div>
