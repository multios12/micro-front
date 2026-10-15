import type { CategoryApiItem } from '../../lib/admin-api'

export type CategoryRow = {
  key: string
  category: CategoryApiItem | null
  isHeading: boolean
  canMoveUp: boolean
  canMoveDown: boolean
}

// 最上位は種別を問わず表示順に並べ、子は親の直後に配置する。
export function buildCategoryRows(categories: CategoryApiItem[]): CategoryRow[] {
  const parentIDs = new Set(categories.map((category) => category.parent_id).filter((id) => id !== null))
  const ordered = [...categories].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
  const rows: CategoryRow[] = []
  for (const root of ordered.filter((category) => category.parent_id === null)) {
    for (const category of [root, ...ordered.filter((child) => child.parent_id === root.id)]) {
      rows.push({ key: `category-${category.id}`, category, isHeading: category.is_heading || parentIDs.has(category.id), canMoveUp: false, canMoveDown: false })
    }
  }
  const groups = new Map<string, CategoryRow[]>()
  for (const row of rows) {
    if (!row.category) continue
    const groupKey = `siblings-${row.category.parent_id ?? 'root'}`
    const siblings = groups.get(groupKey) ?? []
    siblings.push(row)
    groups.set(groupKey, siblings)
  }
  for (const siblings of groups.values()) {
    siblings.forEach((row, index) => {
      row.canMoveUp = index > 0
      row.canMoveDown = index < siblings.length - 1
    })
  }
  return rows
}
