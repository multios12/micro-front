<script lang="ts">
  export let title = ''
</script>

<header class="admin-topbar">
  <div class="admin-topbar-title">
    <h1>{title}</h1>
  </div>

  <div class="admin-topbar-actions">
    <slot name="actions" />
  </div>

  <a
    class="admin-menu-toggle lg:hidden"
    href="#mobile-nav"
    aria-label="ナビゲーションを開く"
  >
    <span></span>
    <span></span>
    <span></span>
  </a>
</header>
