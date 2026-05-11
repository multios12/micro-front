<script lang="ts">
  import { FileLock2 } from "lucide-svelte";
  import Icon from "./MDInput/Icon.svelte";
  import { formatPublishedDate } from "../lib/date-format";

  type BlogRow = {
    id: number;
    title: string;
    summary: string;
    category: string;
    status: "public" | "private";
    publishedAt: string;
    updatedAt?: string;
    href: string;
  };

  export let rows: readonly BlogRow[] = [];
  export let includeUpdatedAt = false;
  export let compactSummary = false;

  const rowClass = (status: BlogRow["status"]) =>
    status === "private"
      ? "border-rose-400/18 bg-rose-950/26 shadow-[0_24px_64px_rgba(127,29,29,0.14)]"
      : "border-white/10 bg-slate-800/90 shadow-[0_24px_64px_rgba(0,0,0,0.24)]";

  const openRow = (href: string) => {
    location.hash = href;
  };

  const truncateSummary = (summary: string) => {
    const text = summary.trim();
    const chars = Array.from(text);

    return chars.length > 20 ? `${chars.slice(0, 20).join("")}...` : text;
  };
</script>

<div class="grid gap-4 lg:hidden">
  {#each rows as row}
    <button
      class={`rounded-[18px] border p-4 ${rowClass(row.status)}`}
      type="button"
      on:click={() => openRow(row.href)}
    >
      <div class="grid gap-1">
        <div class="admin-label">タイトル</div>
        <div class="text-lg font-semibold text-white">{row.title}</div>
      </div>
      <div class="mt-3 grid gap-1">
        <div class="admin-label">カテゴリ</div>
        <div class="text-sm text-slate-200">{row.category}</div>
      </div>
      <div class="mt-4 flex items-center justify-between gap-3">
        <span class="admin-label">公開状態</span>
        {#if row.status === "private"}
          <span
            class="inline-flex items-center gap-1.5 rounded-full border border-rose-400/30 bg-rose-500/10 px-3 py-1 text-sm font-semibold text-rose-300 shadow-[0_0_0_1px_rgba(251,113,133,0.12)]"
          >
            <Icon icon={FileLock2} />
            <span>非公開</span>
          </span>
        {:else}
          <span class="text-sm font-medium text-slate-300">公開中</span>
        {/if}
        <span class="text-sm text-slate-300"
          >{formatPublishedDate(row.publishedAt)}</span
        >
      </div>
      {#if includeUpdatedAt && row.updatedAt}
        <div class="mt-3 text-right text-xs text-slate-500">
          {row.updatedAt}
        </div>
      {/if}
      {#if !compactSummary}
        <p class="mt-3 text-sm leading-6 text-slate-300">
          {truncateSummary(row.summary)}
        </p>
      {/if}
    </button>
  {/each}
</div>

<div class="hidden lg:block">
  <table class="admin-table">
    <thead>
      <tr>
        <th>非公開状態</th>
        <th>タイトル</th>
        <th>カテゴリ</th>
        <th>公開日</th>
        {#if includeUpdatedAt}
          <th>更新日</th>
        {/if}
      </tr>
    </thead>
    <tbody>
      {#each rows as row}
        <!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role -->
        <tr
          class={`admin-table-row-link ${row.status === "private" ? "admin-table-row-private" : ""}`}
          tabindex="0"
          role="link"
          on:click={() => openRow(row.href)}
          on:keydown={(event) =>
            (event.key === "Enter" || event.key === " ") && openRow(row.href)}
        >
          <td>
            {#if row.status === "private"}
              <span
                class="inline-flex items-center gap-1.5 rounded-full border border-rose-400/30 bg-rose-500/10 px-3 py-1 text-sm font-semibold text-rose-300 shadow-[0_0_0_1px_rgba(251,113,133,0.12)]"
              >
                <Icon icon={FileLock2} />
              </span>
            {/if}
          </td>
          <td>
            <strong class="block text-slate-100">{row.title}</strong>
            <span class="mt-1 block text-sm text-slate-400"
              >{truncateSummary(row.summary)}</span
            >
          </td>
          <td>{row.category}</td>
          <td>{formatPublishedDate(row.publishedAt)}</td>
          {#if includeUpdatedAt}
            <td>{row.updatedAt}</td>
          {/if}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  :global(.admin-table-row-private) {
    background: linear-gradient(
      180deg,
      rgba(127, 29, 29, 0.16),
      rgba(69, 10, 10, 0.12)
    );
  }

  :global(.admin-table-row-private:hover),
  :global(.admin-table-row-private:focus-visible) {
    background: linear-gradient(
      180deg,
      rgba(127, 29, 29, 0.22),
      rgba(69, 10, 10, 0.18)
    );
  }
</style>
