//go:build !windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

var instanceLockFile *os.File

// mustSingleInstance не даёт запустить второй процесс с тем же токеном (один lock на машину в /tmp).
// Два экземпляра с одним токеном дают Conflict на getUpdates и хаотичные ответы.
func mustSingleInstance(token string) {
	if token == "" {
		return
	}
	sum := sha256.Sum256([]byte(token))
	name := hex.EncodeToString(sum[:8])
	path := filepath.Join(os.TempDir(), "simple_vpn_bot_"+name+".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("lock-файл: %v", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		log.Fatalf("уже запущен другой экземпляр этого бота. Остановите: pkill -f simple_vpn_bot — %v", err)
	}
	instanceLockFile = f
}
