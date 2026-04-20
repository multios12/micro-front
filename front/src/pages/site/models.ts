export type SiteTab = {
  id: string
  label: string
  url: string
}

export const siteTabs: readonly SiteTab[] = [
  { id: 'home', label: 'Home', url: '/' },
  { id: 'blogs', label: 'Blogs', url: '/blogs' },
  { id: 'about', label: 'About', url: '/about' },
] as const
