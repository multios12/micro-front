<script lang="ts">
  import AdminNavigation from "./AdminNavigation.svelte";

  type NavKey = "dashboard" | "blogs" | "site" | "about";

  export let active: NavKey = "dashboard";
  export let blogCount = 0;
  export let siteTitle = "micro-front";

  const activeHash = () =>
    active === "about" ? "#/blog-edit/about" : `#/${active}`;
</script>

<div class="admin-shell">
  <div class="admin-layout">
    <aside class="admin-sidebar">
      <div class="admin-sidebar-brand">
        <a
          class="admin-sidebar-brand-link block text-lg font-semibold text-white"
          href="/"
        >
          {siteTitle}
        </a>
      </div>

      <div class="admin-sidebar-nav">
        <AdminNavigation {active} {blogCount} />
      </div>
    </aside>

    <main class="admin-main">
      <slot />
    </main>
  </div>

  <div class="admin-mobile-nav" id="mobile-nav">
    <a
      class="admin-mobile-nav-backdrop"
      href={activeHash()}
      aria-label="ナビゲーションを閉じる"
    ></a>
    <section class="admin-mobile-nav-panel">
      <div class="admin-mobile-header">
        <a class="admin-sidebar-brand-link text-lg font-semibold text-white" href="/">
          {siteTitle}
        </a>
        <a
          class="admin-button h-10 w-10 justify-center p-0 text-lg"
          href={activeHash()}
          aria-label="ナビゲーションを閉じる"
        >
          ×
        </a>
      </div>

      <nav class="flex-1">
        <AdminNavigation {active} {blogCount} />
      </nav>
    </section>
  </div>
</div>
