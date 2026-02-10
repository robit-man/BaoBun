package main

import "github.com/baoswarm/baobun/internal/core"

func main() {
	trackers := []string{"bao.06b77fc89d2b9785433dd37a9b98a3c8fa37f03db2b2cc0e79be76f87b223d21"}
	file, _ := core.CreateFromFile("./BaoBun/downloads/BigBuckBunny_320x180.mp4", trackers)
	file.Save("./BaoBun/test.bao")
}
