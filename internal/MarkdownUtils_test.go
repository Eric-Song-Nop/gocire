package internal

import (
	"reflect"
	"strings"
	"testing"
)

func TestMarkdownPageRendererAssignsUniqueUnicodeHeadingIDsAcrossFragments(t *testing.T) {
	renderer := NewMarkdownPageRenderer()

	firstHTML := renderer.RenderFragment("## 核心链路\n\n第一段。")
	secondHTML := renderer.RenderFragment("## 核心链路\n\n第二段。")
	thirdHTML := renderer.RenderFragment("## 这套架构为什么适合本地 AI Agent\n\n第三段。")

	for _, want := range []string{
		`<h2 id="核心链路">核心链路</h2>`,
		`<h2 id="核心链路-1">核心链路</h2>`,
		`<h2 id="这套架构为什么适合本地-ai-agent">这套架构为什么适合本地 AI Agent</h2>`,
	} {
		html := firstHTML + secondHTML + thirdHTML
		if !strings.Contains(html, want) {
			t.Fatalf("rendered HTML missing %q\nGot:\n%s", want, html)
		}
	}

	got := renderer.Headings()
	want := []MarkdownHeading{
		{Level: 2, ID: "核心链路", Title: "核心链路"},
		{Level: 2, ID: "核心链路-1", Title: "核心链路"},
		{Level: 2, ID: "这套架构为什么适合本地-ai-agent", Title: "这套架构为什么适合本地 AI Agent"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Headings() = %#v, want %#v", got, want)
	}
}
