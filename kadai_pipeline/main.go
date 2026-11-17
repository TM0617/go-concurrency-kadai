package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// テスト用のダミーファイルを作成する関数
func generateDummyFiles(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// 5つのファイルを作成
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		// ダミーの単語データ（"word" という単語を i*10 個書き込む）
		content := strings.Repeat("word ", i*10)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func findFiles(paths chan<- string, root string, ctx context.Context) {
	defer close(paths)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// ディレクトリはスキップ
		if d.IsDir() {
			return nil
		}
		// .txtファイルのみ対象
		if filepath.Ext(path) == ".txt" {
			select {
			case <-ctx.Done():
				// キャンセルされたら探索を打ち切る
				return ctx.Err()
			case paths <- path: // パスを送信
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("探索エラー: %v\n", err)
	}
}

var fileCount int64

func countWords(paths <-chan string, results chan<- int, ctx context.Context) {
	for path := range paths {
		file, err := os.Open(path)
		if err != nil {
			fmt.Println("ファイルオープン失敗", err)
			continue
		}

		// 単語数をカウントする
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanWords) // 単語区切り

		count := 0
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				file.Close()
				return
			default:
				count++
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("ファイル読み込み中にエラー", err)
			file.Close()
			continue
		}
		file.Close()
		// 処理したファイル数をカウント
		atomic.AddInt64(&fileCount, 1)

		select {
		case results <- count:
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	dir := "./kadai_pipeline"
	if err := generateDummyFiles(dir); err != nil {
		fmt.Println("ダミーファイル作成失敗:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	paths := make(chan string)
	results := make(chan int)

	go findFiles(paths, dir, ctx)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			countWords(paths, results, ctx)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	total := 0
	for count := range results {
		total += count
	}
	if err := ctx.Err(); err != nil {
		fmt.Println("処理がキャンセルされました:", err)
	} else {
		fmt.Println("処理完了！")
		fmt.Printf("処理ファイル数: %d\n", atomic.LoadInt64(&fileCount))
		fmt.Printf("総単語数は%d\n", total)
	}
}
