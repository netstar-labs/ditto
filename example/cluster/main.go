// Example cluster demonstrates fingerprinting, Hamming distance, and near-
// duplicate clustering over a tiny in-memory corpus.
//
//	go run ./example/cluster
package main

import (
	"fmt"

	"github.com/netstar-labs/ditto"
)

func main() {
	corpus := map[string]string{
		"phish-apple-1": "Your Apple ID has been locked. Please verify your account now.",
		"phish-apple-2": "Your Apple ID has been locked. Please verify your account now!!", // near-dup
		"phish-paypal":  "Your PayPal account has been limited. Confirm your details now.",
		"bulk-news-1":   "This week's newsletter: top stories from around the web.",
		"bulk-news-2":   "This week's newsletter: top stories from around the web.", // exact dup
	}

	f := ditto.Default()

	fmt.Println("fingerprints:")
	fps := map[string]ditto.Fingerprint{}
	for id, text := range corpus {
		fps[id] = f.Of(text)
	}
	for _, id := range []string{"phish-apple-1", "phish-apple-2", "phish-paypal", "bulk-news-1", "bulk-news-2"} {
		fmt.Printf("  %s  %s\n", fps[id], id)
	}

	fmt.Println("\ndistances from phish-apple-1:")
	base := fps["phish-apple-1"]
	for _, id := range []string{"phish-apple-2", "phish-paypal", "bulk-news-1"} {
		fmt.Printf("  %-14s d=%d\n", id, ditto.Distance(base, fps[id]))
	}

	ix := ditto.NewIndex(4)
	for id, fp := range fps {
		ix.Add(id, fp)
	}
	fmt.Println("\nnear-duplicate families (k=4):")
	for i, g := range ix.Clusters(4, 2) {
		fmt.Printf("  [%d] %v\n", i+1, g)
	}
}
