export function formatCategory(category: string, parentCategory?: string): string {
  return [parentCategory?.trim(), category.trim()].filter(Boolean).join(' > ') || '未分類'
}
