import './app.css'
import App from './App.svelte'
import { mount } from 'svelte'

const pathname = location.pathname
const lastSegment = pathname.split('/').pop() ?? ''
const shouldNormalizePath = pathname !== '/' && !pathname.endsWith('/') && !lastSegment.includes('.')

if (shouldNormalizePath) {
  location.replace(`${pathname}/${location.search}${location.hash}`)
}

const app = shouldNormalizePath
  ? undefined
  : mount(App, {
      target: document.getElementById('app')!,
    })

export default app
