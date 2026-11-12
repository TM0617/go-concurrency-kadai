package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type SafeUrlMap struct {
	visited map[string]bool
	mu      sync.Mutex
}

func (s *SafeUrlMap) Visit(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.visited[url] {
		return false
	}
	s.visited[url] = true
	return true
}

// URLを受け取り、そのページに含まれるリンク（文字列のスライス）を返します
func fetch(url string) []string {
	// 処理時間をランダムにシミュレーション（0〜1.3秒）
	// ※タイムアウト(1秒)の挙動を確認するため
	time.Sleep(time.Duration(rand.Intn(1300)) * time.Millisecond)

	// 擬似的なリンクデータ
	links := map[string][]string{
		"https://example.com":   {"https://example.com/a", "https://example.com/b"},
		"https://example.com/a": {"https://example.com/c"},
		"https://example.com/b": {"https://example.com/a"}, // 重複リンク
		"https://example.com/c": {},
	}
	return links[url]
}

func Crawl(url string, wg *sync.WaitGroup, urls *SafeUrlMap) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		// 訪問済みURLなら終了
		if !urls.Visit(url) {
			return
		}

		ch := make(chan []string, 1)

		go func() {
			ch <- fetch(url)
		}()

		var res []string
		select {
		case res = <-ch:
			fmt.Println("成功:", url, res)
		case <-time.After(1 * time.Second):
			fmt.Println("エラー:タイムアウトしました。", url)
			return
		}

		// 取得したリンクを再帰的にクロール
		for _, link := range res {
			Crawl(link, wg, urls)
		}
	}()
}

func main() {
	var wg sync.WaitGroup
	urls := SafeUrlMap{
		visited: make(map[string]bool),
	}
	url := "https://example.com"

	Crawl(url, &wg, &urls)

	wg.Wait()
	fmt.Println("クロール終了")

}
