<script lang="ts">
  import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";

  interface Props {
    icon: IconDefinition;
    label?: string;
  }

  let { icon, label = "" }: Props = $props();
  const width = $derived(icon.icon[0]);
  const height = $derived(icon.icon[1]);
  const pathData = $derived(icon.icon[4]);
</script>

<svg
  class="md-icon"
  viewBox={`0 0 ${width} ${height}`}
  aria-hidden={label === ""}
  aria-label={label || undefined}
  role={label === "" ? undefined : "img"}
>
  {#if Array.isArray(pathData)}
    {#each pathData as path}
      <path d={path}></path>
    {/each}
  {:else}
    <path d={pathData}></path>
  {/if}
</svg>

<style>
  .md-icon {
    display: inline-block;
    width: 1em;
    height: 1em;
    fill: currentColor;
    flex: none;
  }
</style>
