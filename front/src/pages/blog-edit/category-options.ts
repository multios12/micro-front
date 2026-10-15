import type { CategoryApiItem } from '../../lib/admin-api'

export type BlogCategoryOption = {
  value: string
  label: string
  parentCategory: string
  category: string
}

export function categorySelectionValue(parentCategory: string, category: string): string {
  return parentCategory || category ? JSON.stringify([parentCategory, category]) : ''
}

export function categoryOptionLabel(parentCategory: string, category: string): string {
  return parentCategory && category ? `${parentCategory} > ${category}` : category || parentCategory
}

export function buildBlogCategoryOptions(items: readonly CategoryApiItem[]): BlogCategoryOption[] {
  const byID = new Map(items.map((item) => [item.id, item]))
  const parentIDs = new Set(items.map((item) => item.parent_id))
  const options: BlogCategoryOption[] = []
  const sorted = [...items].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
  const ordered = sorted
    .filter((item) => item.parent_id === null)
    .flatMap((root) => [root, ...sorted.filter((item) => item.parent_id === root.id)])
  for (const item of ordered) {
    const heading = item.is_heading || parentIDs.has(item.id)
    if (heading) continue
    const parentCategory = byID.get(item.parent_id ?? -1)?.name ?? ''
    const category = item.name
    const option = {
      value: categorySelectionValue(parentCategory, category),
      label: categoryOptionLabel(parentCategory, category),
      parentCategory,
      category,
    }
    options.push(option)
  }
  return options
}
