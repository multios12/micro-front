<script lang="ts">
  import { Info, LayoutDashboard, Newspaper, PanelTop } from "lucide-svelte";
  import Icon from "./MDInput/Icon.svelte";

  type NavKey = "dashboard" | "blogs" | "site" | "about";

  export let active: NavKey = "dashboard";
  export let blogCount = 0;
  let items: Array<{
    key: NavKey;
    label: string;
    href: string;
    badge?: string;
    icon: typeof LayoutDashboard;
  }> = [];

  $: items = [
    {
      key: "dashboard",
      label: "Dashboard",
      href: "#/dashboard",
      icon: LayoutDashboard,
    },
    {
      key: "blogs",
      label: "Blogs",
      href: "#/blogs",
      badge: String(blogCount),
      icon: Newspaper,
    },
    { key: "site", label: "Site", href: "#/site", icon: PanelTop },
    { key: "about", label: "About", href: "#/blog-edit/about", icon: Info },
  ];
</script>

<nav class="flex flex-col gap-2">
  <div class="admin-label">Navigation</div>

  {#each items as item}
    <a
      class={`flex items-center justify-between gap-2 rounded-2xl border px-4 py-4 font-medium transition ${
        active === item.key
          ? "border-amber-400/30 bg-amber-400/10 text-amber-50"
          : "border-transparent text-slate-200 hover:border-slate-500/50 hover:bg-white/5"
      }`}
      href={item.href}
    >
      <span class="flex items-center gap-2">
        <Icon icon={item.icon} />
        <span>{item.label}</span>
      </span>
      {#if item.badge}
        <span
          class="rounded-full bg-sky-400/15 px-2 py-2 text-xs font-semibold text-sky-200"
        >
          {item.badge}
        </span>
      {/if}
    </a>
  {/each}
</nav>
