export type SiteFormAction = {
  label: string
  href: string
  variant: 'ghost' | 'primary' | 'secondary' | 'danger'
}

export type SiteTab = {
  tab_label: string
  tab_url: string
}

export const siteHeaderAction = {
  label: 'ダッシュボードへ戻る',
  href: '#/dashboard',
} as const

export function normalizeSiteTabs(items: Array<{ tab_label: string; tab_url: string }>): SiteTab[] {
  return items.map((item) => ({ ...item }))
}

export function createEmptySiteTab(): SiteTab {
  return { tab_label: '', tab_url: '/' }
}

export function addSiteTab(tabs: SiteTab[]): SiteTab[] {
  return [...tabs, createEmptySiteTab()]
}

export function updateSiteTab(tabs: SiteTab[], index: number, key: keyof SiteTab, value: string): SiteTab[] {
  return tabs.map((tab, currentIndex) => (currentIndex === index ? { ...tab, [key]: value } : tab))
}

export function removeSiteTab(tabs: SiteTab[], index: number): SiteTab[] {
  return tabs.filter((_, currentIndex) => currentIndex !== index)
}

export function buildSiteUpdateRequest(args: {
  siteTitle: string
  siteSubtitle: string
  siteDescription: string
  siteUrl: string
  tabs: SiteTab[]
  footInformation: string
  copyright: string
}): {
  site_title: string
  site_subtitle: string
  site_description: string
  site_url: string
  tabs: SiteTab[]
  foot_information: string
  copyright: string
} {
  return {
    site_title: args.siteTitle,
    site_subtitle: args.siteSubtitle,
    site_description: args.siteDescription,
    site_url: args.siteUrl,
    tabs: args.tabs,
    foot_information: args.footInformation,
    copyright: args.copyright,
  }
}
