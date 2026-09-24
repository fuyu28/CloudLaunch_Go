// パスを OS 標準のファイルマネージャで開くための共通ヘルパ。
package app

import (
	"os/exec"
	"sync"
)

// openPathStarter はテストで差し替え、Start 失敗を実バイナリなしで検証できるようにする。
// 並列テストと本番呼び出しの交差を避けるため、読み書きは mu 経由にする。
var (
	openPathStarterMu sync.Mutex
	openPathStarter   = startOpenPathCommand
)

// openPathCommand はシェルを介さず、実行ファイル名とパスを独立引数で組み立てる。
func openPathCommand(name, path string) *exec.Cmd {
	return exec.Command(name, path)
}

func startOpenPathCommand(name, path string) error {
	return openPathCommand(name, path).Start()
}

func runOpenPath(name, path string) error {
	openPathStarterMu.Lock()
	starter := openPathStarter
	openPathStarterMu.Unlock()
	return starter(name, path)
}

// setOpenPathStarterForTest はテスト用に starter を差し替え、復元関数を返す。
func setOpenPathStarterForTest(starter func(name, path string) error) (restore func()) {
	openPathStarterMu.Lock()
	previous := openPathStarter
	openPathStarter = starter
	openPathStarterMu.Unlock()
	return func() {
		openPathStarterMu.Lock()
		openPathStarter = previous
		openPathStarterMu.Unlock()
	}
}
