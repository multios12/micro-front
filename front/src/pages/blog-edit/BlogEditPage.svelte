<script lang="ts">
  import { onMount } from "svelte";
  import AdminHeader from "../../components/AdminHeader.svelte";
  import ConfirmDialog from "../../components/ConfirmDialog.svelte";
  import FlashMessage from "../../components/FlashMessage.svelte";
  import FormActionButtons from "../../components/FormActionButtons.svelte";
  import ImageUploader from "../../components/ImageUploader.svelte";
  import MDInput from "../../components/MDInput/index.svelte";
  import Toast from "../../components/Toast.svelte";
  import {
    createBlogEditLabels,
    resolveBlogEditMode,
    resolveBlogEditTargetId,
  } from "./logic";
  import {
    createBlog,
    deleteBlog,
    deleteBlogImage,
    ApiError,
    fetchBlogDetail,
    fetchBlogImages,
    extractValidationFields,
    publish,
    uploadBlogImage,
    updateBlog,
  } from "../../lib/admin-api";
  import { refreshBlogCount } from "../../lib/blog-count";

  export let blogId = "new";
  $: mode = resolveBlogEditMode(blogId);
  $: labels = createBlogEditLabels(blogId);

  let deleteOpen = false;
  let loading = true;
  let saving = false;
  let publishing = false;
  let deleting = false;
  let message = "";
  let messageTone: "success" | "warning" | "error" = "success";
  let validationFields: Record<string, string> = {};
  let toastOpen = false;
  let toastTitle = "";
  let toastMessage = "";
  let toastTone: "success" | "warning" | "error" = "error";
  let resolvedBlogId: number | null = null;
  let articleId = "";
  let title = "";
  let category = "";
  let updatedAt = "";
  let content = "";
  let status: "public" | "private" = "private";
  let headerTitle = "";
  let pageBodyTitle = "";
  let cancelHref = "";
  let showPublishButton = false;
  let showDeleteButton = false;
  let imageNote = "";
  let deleteMessage = "";
  let lastLoadedBlogId = "";
  let blogImages: Array<{
    id: number;
    label: string;
    imageUrl: string;
    altText?: string;
  }> = [];

  const getTodayInputValue = () => {
    const now = new Date();
    const yyyy = String(now.getFullYear());
    const mm = String(now.getMonth() + 1).padStart(2, "0");
    const dd = String(now.getDate()).padStart(2, "0");
    return `${yyyy}-${mm}-${dd} 00:00:00`;
  };
  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: "smooth" });
  };
  const showErrorToast = (text: string) => {
    toastTitle = "エラー";
    toastMessage = text;
    toastTone = "error";
    toastOpen = true;
  };
  const getTabFieldError = (index: number, key: "tab_label" | "tab_url") =>
    validationFields[`tabs[${index}].${key}`] ?? "";
  const mapBlogImages = (
    items: Awaited<ReturnType<typeof fetchBlogImages>>["items"],
  ) =>
    items.map((image) => ({
      id: image.id,
      label: "URLをコピー",
      imageUrl: image.url,
      altText: image.alt_text,
    }));

  $: headerTitle = labels.headerTitle;
  $: pageBodyTitle = labels.pageBodyTitle;
  $: cancelHref = labels.cancelHref;
  $: showPublishButton = mode === "blog";
  $: showDeleteButton = mode !== "new";
  $: imageNote = labels.imageNote;
  $: deleteMessage = labels.deleteMessage;
  $: if (blogId !== lastLoadedBlogId) {
    lastLoadedBlogId = blogId;
    void loadBlog();
  }

  const loadBlog = async () => {
    scrollToTop();
    loading = true;
    validationFields = {};
    toastOpen = false;
    let targetId = Number.NaN;

    try {
      if (mode === "new") {
        resolvedBlogId = null;
        articleId = "new";
        title = "";
        category = "";
        updatedAt = getTodayInputValue();
        content = "";
        status = "private";
        blogImages = [];
        return;
      }

      targetId = resolveBlogEditTargetId(blogId);
      if (!targetId || Number.isNaN(targetId)) {
        throw new Error(
          mode === "about" ? "about 記事のIDが不正です" : "記事のIDが不正です",
        );
      }

      resolvedBlogId = targetId;
      const detail = await fetchBlogDetail(targetId);
      applyBlog(detail);
      try {
        const images = await fetchBlogImages(targetId);
        blogImages = mapBlogImages(images.items);
      } catch {
        blogImages = [];
      }
    } catch (err) {
      if (mode === "about" && err instanceof ApiError && err.status === 404) {
        resolvedBlogId = targetId;
        articleId = String(targetId);
        title = "";
        category = "";
        updatedAt = "";
        content = "";
        status = "private";
        blogImages = [];
        return;
      }
      showErrorToast(
        err instanceof Error ? err.message : "記事の読み込みに失敗しました",
      );
    } finally {
      loading = false;
    }
  };

  const applyBlog = (detail: Awaited<ReturnType<typeof fetchBlogDetail>>) => {
    articleId = String(detail.id);
    title = detail.title;
    category = detail.category;
    updatedAt = detail.updated_at;
    content = detail.content;
    status = detail.status;
  };

  const buildBlogPayload = (nextStatus: "public" | "private" = status) => ({
    title: mode === "about" ? "about" : title,
    content,
    category: mode === "about" ? "" : category,
    status: nextStatus,
    published_at: updatedAt,
  });

  const reloadBlogImages = async (blogId: number) => {
    try {
      const images = await fetchBlogImages(blogId);
      blogImages = mapBlogImages(images.items);
    } catch {
      blogImages = [];
    }
  };

  const getMarkdownImageUploadPath = () =>
    resolvedBlogId === null
      ? ""
      : `admin/api/blogs/${resolvedBlogId}/images`;

  const handleImageSelect = async (file: File) => {
    if (resolvedBlogId === null) {
      scrollToTop();
      showErrorToast("画像を保存するためには、一度記事を保存してください。");
      return;
    }

    saving = true;
    message = "";
    validationFields = {};

    try {
      await uploadBlogImage(resolvedBlogId, file);
      await reloadBlogImages(resolvedBlogId);
      toastTitle = "追加完了";
      toastMessage = "画像を追加しました。";
      toastTone = "success";
      toastOpen = true;
    } catch (err) {
      scrollToTop();
      showErrorToast(
        err instanceof Error ? err.message : "画像の追加に失敗しました",
      );
      validationFields = extractValidationFields(err) ?? {};
    } finally {
      saving = false;
    }
  };

  const handleImageDelete = async (imageId: number) => {
    if (resolvedBlogId === null) {
      scrollToTop();
      showErrorToast("画像を保存するためには、一度記事を保存してください。");
      return;
    }

    scrollToTop();
    message = "";
    validationFields = {};

    try {
      await deleteBlogImage(resolvedBlogId, imageId);
      await reloadBlogImages(resolvedBlogId);
      toastTitle = "削除完了";
      toastMessage = "画像を削除しました。";
      toastTone = "success";
      toastOpen = true;
    } catch (err) {
      scrollToTop();
      showErrorToast(
        err instanceof Error ? err.message : "画像の削除に失敗しました",
      );
    }
  };

  const copyTextToClipboard = async (text: string) => {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return;
    }

    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "true");
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    const copied = document.execCommand("copy");
    document.body.removeChild(textarea);
    if (!copied) {
      throw new Error("コピーできませんでした");
    }
  };

  const handleImageCopy = async (item: {
    imageUrl: string;
  }) => {
    if (resolvedBlogId === null) {
      scrollToTop();
      showErrorToast("画像を保存するためには、一度記事を保存してください。");
      return;
    }

    try {
      const url = new URL(item.imageUrl, location.origin).toString();
      await copyTextToClipboard(url);
      toastTitle = "コピー完了";
      toastMessage = "URLをコピーしました。";
      toastTone = "success";
      toastOpen = true;
    } catch (err) {
      scrollToTop();
      showErrorToast(
        err instanceof Error ? err.message : "URLのコピーに失敗しました",
      );
    }
  };

  const saveBlog = async () => {
    scrollToTop();
    saving = true;
    message = "";
    validationFields = {};

    try {
      const saved =
        mode === "new"
          ? await createBlog(buildBlogPayload())
          : await updateBlog(resolvedBlogId as number, buildBlogPayload());

      applyBlog(saved);

      if (mode === "new") {
        resolvedBlogId = saved.id;
        blogId = String(saved.id);
        location.hash = `#/blog-edit/${saved.id}`;
        void refreshBlogCount();
      }

      messageTone = "success";
      toastTitle = "保存完了";
      toastMessage =
        mode === "new" ? "記事を作成しました。" : "記事を保存しました。";
      toastTone = "success";
      toastOpen = true;
    } catch (err) {
      showErrorToast(
        err instanceof Error ? err.message : "記事の保存に失敗しました",
      );
      validationFields = extractValidationFields(err) ?? {};
    } finally {
      saving = false;
    }
  };

  const publishBlog = async () => {
    if (resolvedBlogId === null) {
      return;
    }

    publishing = true;
    validationFields = {};

    try {
      const saved = await updateBlog(resolvedBlogId, buildBlogPayload("public"));
      applyBlog(saved);
      await publish("blog", resolvedBlogId);
      toastTitle = "公開開始";
      toastMessage = "記事を公開状態に更新し、公開用 HTML の再生成を開始しました。";
      toastTone = "success";
      toastOpen = true;
    } catch (err) {
      showErrorToast(
        err instanceof Error ? err.message : "公開処理に失敗しました",
      );
    } finally {
      publishing = false;
    }
  };

  const confirmDelete = async () => {
    if (resolvedBlogId === null) {
      return;
    }

    scrollToTop();
    deleting = true;
    message = "";
    validationFields = {};

    try {
      await deleteBlog(resolvedBlogId);
      deleteOpen = false;
      toastTitle = "削除完了";
      toastMessage = "記事を削除しました。";
      toastTone = "success";
      toastOpen = true;
      void refreshBlogCount();
      location.hash = mode === "about" ? "#/dashboard" : "#/blogs";
    } catch (err) {
      showErrorToast(
        err instanceof Error ? err.message : "記事の削除に失敗しました",
      );
      deleteOpen = false;
    } finally {
      deleting = false;
    }
  };

  const openDeleteDialog = () => {
    scrollToTop();
    deleteOpen = true;
  };

  onMount(() => {
    scrollToTop();
  });
</script>

<svelte:head>
  <title>{headerTitle} | micro-front</title>
</svelte:head>

<AdminHeader title={headerTitle}>
  <svelte:fragment slot="actions">
    <a class="admin-button" href={cancelHref}>
      {mode === "about" ? "ダッシュボードへ戻る" : "一覧に戻る"}
    </a>
  </svelte:fragment>
</AdminHeader>

{#if toastOpen}
  <Toast
    tone={toastTone}
    title={toastTitle}
    message={toastMessage}
    onClose={() => (toastOpen = false)}
  />
{/if}

{#if message}
  <FlashMessage
    tone={messageTone}
    title={messageTone === "success" ? "完了" : "エラー"}
    {message}
  />
{/if}

{#if loading}
  <FlashMessage
    tone="success"
    title="読み込み中"
    message="管理API から記事情報を取得しています。"
  />
{/if}

<section class="admin-editor">
  <div class="admin-panel-head">
    <h2>{pageBodyTitle}</h2>
  </div>

  <div class="admin-stack">
    <div class="admin-grid-2">
      <div class="admin-field">
        <span class="admin-label">ID</span>
        <div class="admin-static">{articleId || "..."}</div>
      </div>
      <div class="admin-field">
        <span class="admin-label">公開状態</span>
        <select
          class="admin-select"
          class:admin-select-error={Boolean(validationFields.status)}
          bind:value={status}
        >
          <option value="public">公開</option>
          <option value="private">非公開</option>
        </select>
        {#if validationFields.status}
          <p class="admin-error-message">{validationFields.status}</p>
        {/if}
      </div>
    </div>

    {#if mode !== "about"}
      <div class="admin-field">
        <label class="admin-label" for="blog-title">タイトル</label>
        <input
          id="blog-title"
          class="admin-input"
          class:admin-input-error={Boolean(validationFields.title)}
          type="text"
          bind:value={title}
        />
        {#if validationFields.title}
          <p class="admin-error-message">{validationFields.title}</p>
        {/if}
      </div>

      <div class="admin-grid-2">
        <div class="admin-field">
          <label class="admin-label" for="category">カテゴリ</label>
          <input
            id="category"
            class="admin-input"
            class:admin-input-error={Boolean(validationFields.category)}
            type="text"
            bind:value={category}
          />
          {#if validationFields.category}
            <p class="admin-error-message">{validationFields.category}</p>
          {/if}
        </div>
        <div class="admin-field">
          <label class="admin-label" for="updated-at">更新日</label>
          <input
            id="updated-at"
            class="admin-input"
            class:admin-input-error={Boolean(validationFields.published_at)}
            type="text"
            bind:value={updatedAt}
          />
          {#if validationFields.published_at}
            <p class="admin-error-message">{validationFields.published_at}</p>
          {/if}
        </div>
      </div>
    {/if}

    <div class="admin-field">
      <label class="admin-label" for="blog-content">本文</label>
      <MDInput
        id="blog-content"
        bind:value={content}
        error={validationFields.content ?? ""}
        imageUploadPath={getMarkdownImageUploadPath()}
        onImageUploaded={() => {
          if (resolvedBlogId !== null) {
            void reloadBlogImages(resolvedBlogId);
          }
        }}
      />
    </div>
  </div>

  <div class="blog-edit-images">
    <ImageUploader
      title="画像"
      note={imageNote}
      buttonLabel="画像を選択"
      items={blogImages}
      onCopyItem={(item) => handleImageCopy(item)}
      onSelectFile={handleImageSelect}
      onDeleteItem={(item) => handleImageDelete(item.id)}
    />
  </div>

  <FormActionButtons
    items={[
      { label: "キャンセル", href: cancelHref, variant: "ghost" },
      {
        label: saving ? "保存中..." : mode === "new" ? "作成する" : "保存する",
        variant: "primary",
        onClick: saveBlog,
      },
      ...(showPublishButton
        ? [
            {
              label: publishing ? "再生成中..." : "記事を公開",
              variant: "secondary" as const,
              onClick: publishBlog,
            },
          ]
        : []),
      ...(showDeleteButton
        ? [
            {
              label: deleting ? "削除中..." : "削除",
              variant: "danger" as const,
              onClick: openDeleteDialog,
            },
          ]
        : []),
    ]}
  />
</section>

{#if showDeleteButton}
  <ConfirmDialog
    open={deleteOpen}
    title="削除確認"
    message={deleteMessage}
    onCancel={() => (deleteOpen = false)}
    onConfirm={confirmDelete}
  />
{/if}

<style>
  .blog-edit-images {
    margin-top: 1.5rem;
  }
</style>
