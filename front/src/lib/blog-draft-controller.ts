export type BlogDraftValues = {
  title: string
  category: string
  publishedAt: string
  content: string
  status: 'public' | 'private'
  titleImageTemplate: string
}

export type BlogDraft = {
  key: string
  savedAt: string
  baseUpdatedAt?: string
  values: BlogDraftValues
}

export type BlogDraftStatus = '' | 'saving' | 'saved' | 'error'

type BlogDraftControllerOptions = {
  getValues: () => BlogDraftValues
  onRestoreAvailable: (hasServerConflict: boolean) => void
  onStatusChange: (status: BlogDraftStatus) => void
}

const databaseName = 'micro-front'
const storeName = 'blog-drafts'
const databaseVersion = 1

/**
 * 下書き保存用の IndexedDB データベースを開き、利用可能な接続を返します。
 *
 * @returns 接続済みの IndexedDB データベース
 */
const openDatabase = (): Promise<IDBDatabase> =>
  new Promise((resolve, reject) => {
    const request = indexedDB.open(databaseName, databaseVersion)
    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(storeName)) {
        database.createObjectStore(storeName, { keyPath: 'key' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('下書き保存領域を開けませんでした'))
  })

/**
 * 下書きストアに対する IndexedDB リクエストを実行します。
 *
 * @typeParam T - IndexedDB リクエストが返す値の型
 * @param mode - トランザクションのアクセスモード
 * @param operation - 対象ストアで実行するリクエストを作成する関数
 * @returns リクエスト結果を解決する Promise
 */
const runRequest = <T>(
  mode: IDBTransactionMode,
  operation: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> =>
  new Promise((resolve, reject) => {
    void openDatabase()
      .then((database) => {
        const transaction = database.transaction(storeName, mode)
        const request = operation(transaction.objectStore(storeName))
        request.onsuccess = () => resolve(request.result)
        request.onerror = () => reject(request.error ?? new Error('下書きの操作に失敗しました'))
        transaction.oncomplete = () => database.close()
        transaction.onerror = () => database.close()
      })
      .catch(reject)
  })

/**
 * 記事編集フォームのローカル下書きを管理するコントローラを作成します。
 * IndexedDB の永続化、復元候補、入力の debounce 保存を一元管理します。
 *
 * @param options - フォーム値とUI状態を連携するためのコールバック
 * @returns 下書き保存を操作するコントローラ
 */
export const createBlogDraftController = (options: BlogDraftControllerOptions) => {
  let key = ''
  let baseUpdatedAt: string | undefined
  let pendingDraft: BlogDraft | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  let enabled = false
  let lastValuesSignature = ''
  let hasPendingChanges = false

  /** 下書きの保存状態をUIへ通知します。 */
  const setStatus = (status: BlogDraftStatus) => options.onStatusChange(status)

  /** 現在の編集対象とサーバー版の更新日時を設定して、復元候補を読み込みます。 */
  const prepare = async (nextKey: string, nextBaseUpdatedAt?: string) => {
    enabled = false
    key = nextKey
    baseUpdatedAt = nextBaseUpdatedAt
    pendingDraft = null
    if (timer) {
      clearTimeout(timer)
      timer = undefined
    }

    try {
      const draft = await runRequest<BlogDraft | undefined>('readonly', (store) => store.get(key))
      if (!draft) {
        return
      }
      pendingDraft = draft
      options.onRestoreAvailable(
        Boolean(draft.baseUpdatedAt && draft.baseUpdatedAt !== baseUpdatedAt),
      )
    } catch {
      setStatus('error')
    }
  }

  /** 現在のフォーム値を基準値として自動保存を有効化します。 */
  const enable = () => {
    lastValuesSignature = JSON.stringify(options.getValues())
    hasPendingChanges = false
    enabled = true
  }

  /** 自動保存を停止し、保留中のタイマーを解除します。 */
  const disable = () => {
    enabled = false
    if (timer) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  /** 変更済みのフォーム値を IndexedDB に保存します。 */
  const save = async () => {
    if (!enabled || !key) {
      return
    }
    try {
      await runRequest('readwrite', (store) => store.put({
        key,
        savedAt: new Date().toISOString(),
        baseUpdatedAt,
        values: options.getValues(),
      } satisfies BlogDraft))
      hasPendingChanges = false
      setStatus('saved')
    } catch {
      setStatus('error')
    }
  }

  /** 入力変更を受け取り、値が変化していれば遅延保存を予約します。 */
  const onValuesChanged = (valuesSignature: string) => {
    if (!enabled || valuesSignature === lastValuesSignature) {
      return
    }
    lastValuesSignature = valuesSignature
    hasPendingChanges = true
    if (timer) {
      clearTimeout(timer)
    }
    setStatus('saving')
    timer = setTimeout(() => {
      void save()
    }, 1000)
  }

  /** 復元候補を破棄して IndexedDB からも削除します。 */
  const discardPendingDraft = async () => {
    const draft = pendingDraft
    if (draft) {
      try {
        await runRequest('readwrite', (store) => store.delete(draft.key))
      } catch {
        setStatus('error')
        return
      }
    }
    pendingDraft = null
    setStatus('')
  }

  /** 復元候補を返し、コントローラの保留状態から取り除きます。 */
  const restorePendingDraft = () => {
    const draft = pendingDraft
    pendingDraft = null
    if (draft) {
      baseUpdatedAt = draft.baseUpdatedAt
      lastValuesSignature = JSON.stringify(draft.values)
      setStatus('saved')
    }
    return draft
  }

  /** 指定中の記事のローカル下書きを削除します。 */
  const clear = async () => {
    if (!key) {
      return
    }
    try {
      if (timer) {
        clearTimeout(timer)
        timer = undefined
      }
      await runRequest('readwrite', (store) => store.delete(key))
      lastValuesSignature = JSON.stringify(options.getValues())
      hasPendingChanges = false
      setStatus('')
    } catch {
      setStatus('error')
    }
  }

  /** 保留中の保存を取り消し、画面を離れる直前の内容を保存します。 */
  const flush = async () => {
    if (timer) {
      clearTimeout(timer)
      timer = undefined
    }
    if (hasPendingChanges) {
      await save()
    }
  }

  return {
    clear,
    disable,
    discardPendingDraft,
    enable,
    flush,
    onValuesChanged,
    prepare,
    restorePendingDraft,
    save,
  }
}
