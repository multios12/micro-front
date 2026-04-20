package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToHTML_ConvertsSampleMarkdown(t *testing.T) {
	root := filepath.Join("..", "..")
	body, err := os.ReadFile(filepath.Join(root, "docs", "sample-md.md"))
	if err != nil {
		t.Fatalf("read sample-md.md: %v", err)
	}

	got := ToHTML(string(body))

	expected := strings.Join([]string{
		"<h1>見出し１</h1>",
		"<h2>見出し２</h2>",
		"<h3>見出し３</h3>",
		"<p>ノーマル</p>",
		"<ol>",
		"<li>数字付きリスト</li>",
		"<li>数字付きリスト２</li>",
		"</ol>",
		"<ul>",
		"<li>リスト１</li>",
		"<li>リスト２</li>",
		"</ul>",
		"<pre><code>コード\n</code></pre>",
		"<blockquote>",
		"<p>引用単体行</p>",
		"</blockquote>",
		"<blockquote>",
		"<p>引用複数行<br>引用２行目</p>",
		"</blockquote>",
		"<p>文字装飾「<strong>太字</strong>」「<em>斜線</em>」「<u>下線</u>」「<del>取り消し線</del>」「<a href=\"https://google.co.jp\">リンク</a>」</p>",
		"<p><img src=\"/api/diary/2026-04-03/images/001.png\" alt=\"イメージ\"></p>",
		"<p><img src=\"/api/diary/2026-04-03/images/001.png\" alt=\"横幅指定\" width=\"320\"></p>",
		"<p><img src=\"/api/diary/2026-04-03/images/001.png\" alt=\"サイズ指定\" width=\"320\" height=\"180\"></p>",
		"<p><img src=\"/api/diary/2026-04-03/images/001.png\" alt=\"横幅率指定\" style=\"width: 50%;\"></p>",
		"<p><img src=\"/api/diary/2026-04-03/images/001.png\" alt=\"横幅率と高さ指定\" style=\"width: 50%;\" height=\"180\"></p>",
		"<p>画像サイズ指定は Obsidian 風の記法に対応</p>",
		"<ul>",
		"<li><code>![代替テキスト|320](画像URL)</code> : 横幅だけ指定</li>",
		"<li><code>![代替テキスト|320x180](画像URL)</code> : 横幅と高さを指定</li>",
		"<li><code>![代替テキスト|50%](画像URL)</code> : 本文幅に対する比率で横幅を指定</li>",
		"<li><code>![代替テキスト|50%x180](画像URL)</code> : 本文幅に対する比率で横幅、高さを数値で指定</li>",
		"</ul>",
		"<hr>",
		"<ul>",
		"<li>親リスト１",
		"<ul>",
		"<li>子リスト１</li>",
		"<li>子リスト２</li>",
		"</ul></li>",
		"<li>親リスト２</li>",
		"</ul>",
		"<table>",
		"<thead>",
		"<tr><th>名前</th><th>説明</th></tr>",
		"</thead>",
		"<tbody>",
		"<tr><td>サンプル1</td><td>テーブル１行目</td></tr>",
		"<tr><td>サンプル2</td><td>テーブル２行目</td></tr>",
		"</tbody>",
		"</table>",
	}, "\n")

	if got != expected {
		t.Fatalf("unexpected html\nexpected:\n%s\n\ngot:\n%s", expected, got)
	}
}

func TestToHTML_SupportsTablesNestedListsAndDecorations(t *testing.T) {
	input := strings.Join([]string{
		"- 親1",
		"  - 子1",
		"  - 子2",
		"- 親2",
		"",
		"| 名前 | 説明 |",
		"| --- | --- |",
		"| a | `code` |",
		"| b | [link](https://example.com) |",
		"",
		"---",
		"",
		"__下線__ と ~~取り消し~~",
	}, "\n")

	got := ToHTML(input)
	expected := strings.Join([]string{
		"<ul>",
		"<li>親1",
		"<ul>",
		"<li>子1</li>",
		"<li>子2</li>",
		"</ul></li>",
		"<li>親2</li>",
		"</ul>",
		"<table>",
		"<thead>",
		"<tr><th>名前</th><th>説明</th></tr>",
		"</thead>",
		"<tbody>",
		"<tr><td>a</td><td><code>code</code></td></tr>",
		"<tr><td>b</td><td><a href=\"https://example.com\">link</a></td></tr>",
		"</tbody>",
		"</table>",
		"<hr>",
		"<p><u>下線</u> と <del>取り消し</del></p>",
	}, "\n")

	if got != expected {
		t.Fatalf("unexpected html\nexpected:\n%s\n\ngot:\n%s", expected, got)
	}
}

func TestToHTML_HidesCarryOverMarkerLine(t *testing.T) {
	input := strings.Join([]string{
		"## お店",
		"テストデータ",
		"",
		"----ここまで前回内容で置換",
		"## お話",
		"----",
	}, "\n")

	got := ToHTML(input)
	expected := strings.Join([]string{
		"<h2>お店</h2>",
		"<p>テストデータ</p>",
		"<hr>",
		"<h2>お話</h2>",
		"<hr>",
	}, "\n")

	if got != expected {
		t.Fatalf("unexpected html\nexpected:\n%s\n\ngot:\n%s", expected, got)
	}
}

func TestToHTML_SupportsObsidianStyleImageSize(t *testing.T) {
	input := strings.Join([]string{
		"![sample|320](https://example.com/sample.png)",
		"",
		"![sample size|320x180](https://example.com/sample.png)",
		"",
		"![sample percent|50%](https://example.com/sample.png)",
		"",
		"![sample percent size|50%x180](https://example.com/sample.png)",
		"",
		"![plain](https://example.com/sample.png)",
	}, "\n")

	got := ToHTML(input)
	expected := strings.Join([]string{
		"<p><img src=\"https://example.com/sample.png\" alt=\"sample\" width=\"320\"></p>",
		"<p><img src=\"https://example.com/sample.png\" alt=\"sample size\" width=\"320\" height=\"180\"></p>",
		"<p><img src=\"https://example.com/sample.png\" alt=\"sample percent\" style=\"width: 50%;\"></p>",
		"<p><img src=\"https://example.com/sample.png\" alt=\"sample percent size\" style=\"width: 50%;\" height=\"180\"></p>",
		"<p><img src=\"https://example.com/sample.png\" alt=\"plain\"></p>",
	}, "\n")

	if got != expected {
		t.Fatalf("unexpected html\nexpected:\n%s\n\ngot:\n%s", expected, got)
	}
}
