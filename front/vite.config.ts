import { defineConfig, loadEnv, Plugin } from 'vite'
import { OutputChunk, OutputAsset } from "rollup"
import tailwindcss from "@tailwindcss/vite"
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const html = env.HTML || "index.html"
  const base = env.BASE_URL || "./"
  const basePath = base === "./" || base === "/" ? "" : `/${base.replace(/^\/+|\/+$/g, "")}`
  const escapedBasePath = escapeRegExp(basePath)
  const apiTarget = "http://localhost:3001"
  const stripBasePath = (path: string) => (basePath ? path.replace(new RegExp(`^${escapedBasePath}`), "") : path)
  return {
    plugins: [tailwindcss(), svelte(), singleFilePlugin(base)],
    build: {
      rollupOptions: {
        input: html,
      },
    },
    base,
    server: {
      watch: { usePolling: true },
      host: "0.0.0.0",
      port: 3000,
      proxy: {
        [`^${escapedBasePath}/admin/api(?:/.*)?$`]: {
          target: apiTarget,
          rewrite: stripBasePath,
        },
        [`^${escapedBasePath}/admin/preview(?:/.*)?$`]: {
          target: apiTarget,
          rewrite: stripBasePath,
        },
        [`^${escapedBasePath}/admin/images(?:/.*)?$`]: {
          target: apiTarget,
          rewrite: stripBasePath,
        },
        [`^${escapedBasePath}/settings$`]: {
          target: apiTarget,
          rewrite: stripBasePath,
        },
      },
    },
  }
})

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
}

function singleFilePlugin(base: string): Plugin {
  return {
    name: 'vite:singleFile',
    enforce: 'post',
    async generateBundle(_options, bundle) {
      const htmlNames = Object.keys(bundle).filter(key => key.endsWith('.html'));
      if (!htmlNames || htmlNames.length != 1) {
        console.log("必ず、1つのHTMLファイルを指定する必要があります。複数のHTMLファイルは指定できません。")
        return
      }

      const deleteTarget = [] as string[]
      const htmlAsset = bundle[htmlNames[0]] as OutputAsset
      let filter = htmlNames[0].replace(".html", "")
      let body = htmlAsset.source as string
      const normalizedBase = base.endsWith("/") ? base : `${base}/`

      let re = new RegExp(`^assets/${filter}.*js$`)
      const jsNames = Object.keys(bundle).filter(key => re.test(key));

      for (const jsName of jsNames) {
        const target = `<script type="module" crossorigin src="${normalizedBase}${jsName}"></script>`
        re = new RegExp(escapeRegExp(target))
        if (re.test(body)) {
          const jsChunk = bundle[jsName] as OutputChunk
          const replaced = `<script type="module" crossorigin>\n${jsChunk.code}\n</script>`
          const targets = body.split(target)
          body = targets[0] + replaced + targets[1]
          htmlAsset.source = body
          deleteTarget.push(jsName)
        }
      }
      re = new RegExp(`^assets/${filter}.*css$`)
      const cssNames = Object.keys(bundle).filter(key => re.test(key));

      for (const css of cssNames) {
        const target = `<link rel="stylesheet" crossorigin href="${normalizedBase}${css}">`
        re = new RegExp(escapeRegExp(target))
        if (re.test(body)) {
          const replaced = `<style type="text/css">\n${(bundle[css] as any).source}\n</style>`
          body = body.replace(target, replaced);
          htmlAsset.source = body
          deleteTarget.push(css)
        }
      }
      for (const key of deleteTarget) {
        delete bundle[key]
      }
    }
  }
}
