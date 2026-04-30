<script lang="ts">
  import { onMount } from "svelte";

  import AdminHeader from "../../components/AdminHeader.svelte";
  import FlashMessage from "../../components/FlashMessage.svelte";
  import FormActionButtons from "../../components/FormActionButtons.svelte";
  import Toast from "../../components/Toast.svelte";
  import MarkdownEditor from "../../components/MarkdownEditor.svelte";
  import {
    fetchSiteSettings,
    createSitePreview,
    extractValidationFields,
    updateSiteSettings,
    type SiteApiResponse,
  } from "../../lib/admin-api";
  import { siteTitle as siteTitleStore } from "../../lib/site-title";
  import {
    addSiteTab,
    buildSiteUpdateRequest,
    normalizeSiteTabs,
    removeSiteTab,
    updateSiteTab,
    type SiteTab,
  } from "./logic";

  let loading = true;
  let saving = false;
  let previewing = false;
  let messageTone: "success" | "warning" | "error" = "success";
  let validationFields: Record<string, string> = {};
  let toastOpen = false;
  let toastTitle = "";
  let toastMessage = "";

  let siteTitle = "";
  let siteSubtitle = "";
  let siteDescription = "";
  let footInformation = "";
  let copyright = "";
  let tabs: SiteTab[] = [];

  const siteHeaderAction = {
    label: "ダッシュボードへ戻る",
    href: "#/dashboard",
  } as const;

  const loadSite = async () => {
    loading = true;
    validationFields = {};
    toastOpen = false;

    try {
      const site = await fetchSiteSettings();
      applySite(site);
    } catch (err) {
      messageTone = "error";
      toastTitle = "エラー";
      toastMessage =
        err instanceof Error
          ? err.message
          : "サイト設定の読み込みに失敗しました";
      toastOpen = true;
    } finally {
      loading = false;
    }
  };

  const applySite = (site: SiteApiResponse) => {
    siteTitle = site.site_title;
    siteSubtitle = site.site_subtitle;
    siteDescription = site.site_description;
    footInformation = site.foot_information;
    copyright = site.copyright;
    tabs = normalizeSiteTabs(site.tabs);
  };

  const addTab = () => {
    tabs = addSiteTab(tabs);
  };

  const removeTab = (index: number) => {
    tabs = removeSiteTab(tabs, index);
  };

  const updateTab = (index: number, key: keyof SiteTab, value: string) => {
    tabs = updateSiteTab(tabs, index, key, value);
  };
  const getTabFieldError = (index: number, key: "tab_label" | "tab_url") =>
    validationFields[`tabs[${index}].${key}`] ?? "";

  const saveSite = async () => {
    window.scrollTo({ top: 0, behavior: "smooth" });
    saving = true;
    validationFields = {};
    toastOpen = false;

    try {
      const saved = await updateSiteSettings(
        buildSiteUpdateRequest({
          siteTitle,
          siteSubtitle,
          siteDescription,
          tabs,
          footInformation,
          copyright,
        }),
      );
      applySite(saved);
      siteTitleStore.set(saved.site_title || "micro-front");
      messageTone = "success";
      toastTitle = "保存完了";
      toastMessage = "サイト設定を保存しました。";
      toastOpen = true;
    } catch (err) {
      if (err instanceof Error) {
        messageTone = "error";
        toastTitle = "エラー";
        toastMessage = err.message;
        toastOpen = true;
        validationFields = extractValidationFields(err) ?? {};
      } else {
        messageTone = "error";
        toastTitle = "エラー";
        toastMessage = "サイト設定の保存に失敗しました";
        toastOpen = true;
        validationFields = {};
      }
    } finally {
      saving = false;
    }
  };

  const previewSite = async () => {
    const previewWindow = window.open("about:blank", "_blank");
    if (!previewWindow) {
      messageTone = "error";
      toastTitle = "エラー";
      toastMessage =
        "ポップアップがブロックされました。ブラウザ設定を確認してください。";
      toastOpen = true;
      return;
    }

    previewWindow.opener = null;
    previewing = true;
    validationFields = {};
    toastOpen = false;

    try {
      const preview = await createSitePreview();
      previewWindow.location.href = preview.url;
      messageTone = "success";
      toastTitle = "プレビュー生成";
      toastMessage = "プレビューを新しいタブで開きました。";
      toastOpen = true;
    } catch (err) {
      previewWindow.close();
      messageTone = "error";
      toastTitle = "エラー";
      toastMessage =
        err instanceof Error ? err.message : "プレビューの生成に失敗しました";
      toastOpen = true;
    } finally {
      previewing = false;
    }
  };

  onMount(() => {
    void loadSite();
  });
</script>

<svelte:head>
  <title>Site | micro-front</title>
</svelte:head>

<AdminHeader title="Site">
  <svelte:fragment slot="actions">
    <button
      class="admin-button admin-button-secondary"
      type="button"
      on:click={previewSite}
    >
      {previewing ? "プレビュー生成中..." : "プレビューを表示"}
    </button>
    <a class="admin-button" href={siteHeaderAction.href}
      >{siteHeaderAction.label}</a
    >
  </svelte:fragment>
</AdminHeader>

{#if toastOpen}
  <Toast
    tone={messageTone}
    title={toastTitle}
    message={toastMessage}
    onClose={() => (toastOpen = false)}
  />
{/if}

{#if loading}
  <FlashMessage
    tone="success"
    title="読み込み中"
    message="管理API からサイト設定を取得しています。"
  />
{/if}

<section class="admin-editor">
  <div class="admin-stack">
    <div class="admin-field">
      <label class="admin-label" for="site-title">タイトル</label>
      <input
        id="site-title"
        class="admin-input"
        class:admin-input-error={Boolean(validationFields.site_title)}
        type="text"
        bind:value={siteTitle}
      />
      {#if validationFields.site_title}
        <p class="admin-error-message">{validationFields.site_title}</p>
      {/if}
    </div>

    <div class="admin-field">
      <label class="admin-label" for="site-subtitle">サブタイトル</label>
      <input
        id="site-subtitle"
        class="admin-input"
        class:admin-input-error={Boolean(validationFields.site_subtitle)}
        type="text"
        bind:value={siteSubtitle}
      />
      {#if validationFields.site_subtitle}
        <p class="admin-error-message">{validationFields.site_subtitle}</p>
      {/if}
    </div>

    <div class="admin-field">
      <label class="admin-label" for="foot-information">フッタ情報</label>
      <input
        id="foot-information"
        class="admin-input"
        class:admin-input-error={Boolean(validationFields.foot_information)}
        type="text"
        bind:value={footInformation}
      />
      {#if validationFields.foot_information}
        <p class="admin-error-message">{validationFields.foot_information}</p>
      {/if}
    </div>

    <div class="admin-field">
      <label class="admin-label" for="copyright">コピーライト</label>
      <input
        id="copyright"
        class="admin-input"
        class:admin-input-error={Boolean(validationFields.copyright)}
        type="text"
        bind:value={copyright}
      />
      {#if validationFields.copyright}
        <p class="admin-error-message">{validationFields.copyright}</p>
      {/if}
    </div>

    <MarkdownEditor
      id="site-description"
      label="サイト説明"
      bind:value={siteDescription}
      error={validationFields.site_description ?? ""}
    />

    <div class="admin-field">
      <label class="admin-label" for="foot-information">タブ</label>
    </div>
    <div class="admin-field">
      <button
        class="admin-button admin-button-secondary"
        type="button"
        on:click={addTab}
      >
        タブを追加
      </button>
    </div>

    {#each tabs as tab, index}
      <div class="grid gap-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
        <div class="admin-grid-2">
          <div class="admin-field">
            <label class="admin-label" for={`tab-label-${index}`}
              >タブラベル</label
            >
            <input
              id={`tab-label-${index}`}
              class="admin-input"
              class:admin-input-error={Boolean(
                getTabFieldError(index, "tab_label"),
              )}
              type="text"
              value={tab.tab_label}
              on:input={(event) =>
                updateTab(
                  index,
                  "tab_label",
                  (event.currentTarget as HTMLInputElement).value,
                )}
            />
            {#if getTabFieldError(index, "tab_label")}
              <p class="admin-error-message">
                {getTabFieldError(index, "tab_label")}
              </p>
            {/if}
          </div>
          <div class="admin-field">
            <label class="admin-label" for={`tab-url-${index}`}>タブURL</label>
            <input
              id={`tab-url-${index}`}
              class="admin-input"
              class:admin-input-error={Boolean(
                getTabFieldError(index, "tab_url"),
              )}
              type="text"
              value={tab.tab_url}
              on:input={(event) =>
                updateTab(
                  index,
                  "tab_url",
                  (event.currentTarget as HTMLInputElement).value,
                )}
            />
            {#if getTabFieldError(index, "tab_url")}
              <p class="admin-error-message">
                {getTabFieldError(index, "tab_url")}
              </p>
            {/if}
          </div>
        </div>
        <button
          class="admin-button admin-button-danger justify-self-start md:justify-self-auto"
          type="button"
          on:click={() => removeTab(index)}
        >
          削除
        </button>
      </div>
    {/each}
  </div>

  <FormActionButtons
    items={[
      { label: "キャンセル", href: siteHeaderAction.href, variant: "ghost" },
      {
        label: previewing ? "プレビュー生成中..." : "プレビューを表示",
        variant: "secondary",
        onClick: previewSite,
      },
      {
        label: saving ? "保存中..." : "保存する",
        variant: "primary",
        onClick: saveSite,
      },
    ]}
  />
</section>
