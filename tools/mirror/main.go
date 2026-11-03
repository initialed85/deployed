package main

// mirror serves the pinned k3s release onto the lab's private net (192.168.56.1 is the gateway address of the
// Vagrant private network, in ./env.sh up and in the CI live lane alike), which is what keeps the live tier
// off GitHub: it installs k3s on four guests at once, and four guests at once have been refused by GitHub
// while this, a single flow, has not.
//
// The paths are the ones k3s-install.sh builds - <INSTALL_K3S_ARTIFACT_URL>/<version>/k3s and
// .../sha256sum-amd64.txt - plus the installer itself at /get.k3s.io, so a guest needs no third party at all.
// ./env.sh up starts this; the guest scripts ask for /get.k3s.io here and fall back to GitHub when it is not
// answered.

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func main() {
	root := flag.String("root", filepath.Join(os.TempDir(), "deployed-k3s-mirror-root"), "directory to serve")
	version := flag.String("version", "v1.36.2+k3s1", "the k3s release the live tier pins")
	addr := flag.String("addr", "192.168.56.1:8080", "address to serve on")
	flag.Parse()

	if err := ensure(*root, *version); err != nil {
		log.Fatalf("mirror: %v", err)
	}

	fmt.Println("MIRROR_READY", *addr)
	log.Fatal(http.ListenAndServe(*addr, logRequests(http.FileServer(http.Dir(*root)))))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func ensure(root, version string) error {
	dir := filepath.Join(root, "k3s-io", "k3s", "releases", "download", version)

	for _, want := range []struct{ dir, file, url string }{
		{dir, "k3s", "https://github.com/k3s-io/k3s/releases/download/" + url.QueryEscape(version) + "/k3s"},
		{dir, "sha256sum-amd64.txt", "https://github.com/k3s-io/k3s/releases/download/" + url.QueryEscape(version) + "/sha256sum-amd64.txt"},
		{root, "get.k3s.io", "https://get.k3s.io"},
	} {
		if _, err := os.Stat(filepath.Join(want.dir, want.file)); err == nil {
			continue
		}
		if err := fetch(want.dir, want.file, want.url); err != nil {
			return err
		}
	}

	return nil
}

func fetch(dir, file, url string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := filepath.Join(dir, file+".tmp")

	for attempt := 1; attempt <= 3; attempt++ {
		if err := download(url, tmp); err == nil {
			return os.Rename(tmp, filepath.Join(dir, file))
		} else if attempt == 3 {
			return fmt.Errorf("fetch %s: %v", url, err)
		} else {
			log.Printf("fetch %s failed (%v), retrying", url, err)
			time.Sleep(5 * time.Second)
		}
	}

	return nil
}

func download(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)

	return err
}
