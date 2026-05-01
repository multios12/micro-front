<script lang="ts">
  import type { ComponentType, SvelteComponent } from "svelte";

  interface Props {
    icon: ComponentType<SvelteComponent>;
    label?: string;
  }

  let { icon: IconComponent, label = "" }: Props = $props();
</script>

<span class="md-icon" aria-hidden={label === ""}>
  <IconComponent
    size={16}
    aria-label={label || undefined}
    role={label === "" ? undefined : "img"}
  />
</span>

<style>
  .md-icon {
    display: inline-block;
    width: 1em;
    height: 1em;
    flex: none;
  }

  .md-icon :global(svg) {
    display: block;
    width: 100%;
    height: 100%;
  }
</style>
