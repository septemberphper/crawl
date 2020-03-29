package main

import (
	"fmt"
	"mark/crawl"
	"strconv"
	"time"
)

func main() {
	start := time.Now()

	for i:=0; i<10; i++ {
		s := crawl.NewSpider("https://movie.douban.com/top250?start=" + strconv.Itoa(25 * i))
		go s.Fetch()
		go s.Parse()
		s.SaveData()
	}

	fmt.Printf("任务耗时：%f秒\n", time.Since(start).Seconds())
}
