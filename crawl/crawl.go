package crawl

import (
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"mark/database"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

func init() {
	log.Println("crawl running...")
}

type spider struct {
	lock *sync.Mutex
	url string
	content chan string				// body内容
	movies chan []map[string]interface{}		// 电影列表内容
	hotComments chan []map[string]interface{}  // 热门评价内容
}

func NewSpider(url string) *spider {
	return &spider{
		lock:new(sync.Mutex),
		url:url,
		content:make(chan string, 1024),
		movies:make(chan []map[string]interface{}, 1024),
		hotComments:make(chan []map[string]interface{}, 1024),
	}
}

// 发起请求
func (s *spider) Fetch() {
	log.Println("fetch url", s.url)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, reqErr := http.NewRequest("GET", s.url, nil)
	if reqErr != nil {
		log.Fatalln("请求失败 :", reqErr.Error())
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	resp, respErr := client.Do(req)
	if respErr != nil {
		log.Fatalln(respErr.Error())
	}

	if resp.StatusCode != 200 {
		log.Fatalln("http status code:", resp.StatusCode)
	}

	defer resp.Body.Close()
	contents, bErr := ioutil.ReadAll(resp.Body)
	if bErr != nil {
		log.Fatalln(bErr.Error())
	}

	s.content <- string(contents)

	//close(s.content)
}

// 解析内容
func (s *spider) Parse() {
	body := <- s.content
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		log.Fatalln("解析html失败", err.Error())
	}

	var movieData []map[string]interface{}		// 电影列表
	var hotCommentData []map[string]interface{}	// 热门评论列表

	dom.Find(".grid_view").Find(".item").Each(func(i int, selection *goquery.Selection) {
		title := selection.Find(".hd").Find("span").Eq(0).Text()
		var bannerUrl string
		if imgUrl, ok := selection.Find(".pic").Find("img").Attr("src"); ok{
			bannerUrl = imgUrl
		} else {
			bannerUrl = ""
		}

		quote := selection.Find(".quote").Find(".inq").Text()
		star := selection.Find(".star").Find(".rating_num").Text()

		// 详情url
		detailUrl, _ := selection.Find(".hd").Find("a").Attr("href");

		// 评论人数
		commentPeopleNum := selection.Find(".star").Find("span").Last().Text()
		re := regexp.MustCompile(`\d+`)
		comment_num := re.FindString(commentPeopleNum)

		// 电影ID
		movieId := re.FindString(detailUrl)

		s.url = detailUrl
		s.Fetch()		// 请求
		detailDom, detailErr := goquery.NewDocumentFromReader(strings.NewReader(<- s.content))

		if detailErr != nil {
			log.Println("解析电影详情失败", detailErr.Error())
		} else {
			hotCommentEle := detailDom.Find("#hot-comments").Find(".comment-item")
			hotCommentEle.Each(func(n int, selection *goquery.Selection) {
				userName := selection.Find(".comment-info").Find("a").Text()
				commentText := selection.Find("p").Find(".short").Text()

				//格式化时间
				commentTime, _ := selection.Find(".comment-time").Attr("title")
				var commentTimeUnix int64
				if commentTime != "" {
					commentTimeUnix = TimeFormTimestamp(commentTime)
				} else {
					commentTimeUnix = 0
				}

				followNum := selection.Find(".comment-vote").Find(".votes").Text()
				followNumInt,_ := strconv.Atoi(followNum)

				commentList := map[string]interface{}{
					"movie_id": movieId,
					"user_name": CheckUtf8(userName),
					"comment_text": CheckUtf8(commentText),
					"comment_time": commentTimeUnix,
					"follow_num": followNumInt,
				}

				hotCommentData = append(hotCommentData, commentList)
			})
		}

		//fmt.Println(title, bannerUrl, quote, star, comment_num)
		item := map[string]interface{}{
			"movie_id" : movieId,
			"movie_name" : title,
			"banner_url": bannerUrl,
			"quote": quote,
			"star": star,
			"comment_num": comment_num,
		}

		movieData = append(movieData, item)
	})

	s.movies <- movieData
	s.hotComments <- hotCommentData

	close(s.movies)
	close(s.hotComments)
}

// 数据存储
func (s *spider) SaveData() {
	DB := database.NewConnect()

	for {
		select {
		case moveList,ok := <- s.movies:
			if !ok {
				log.Println("电影列表接收完毕")
			}
			for _, item := range moveList {
				_, err := DB.Exec("insert INTO movies(movie_id,movie_name,banner_url,quote,star,comment_num,create_time,update_time) values(?,?,?,?,?,?,?,?)",item["movie_id"], item["movie_name"], item["banner_url"], item["quote"], item["star"], item["comment_num"], time.Now().Unix(), time.Now().Unix())
				if err != nil {
					log.Println(err.Error())
				}
			}
		case commentList,ok := <- s.hotComments:
			if !ok {
				log.Fatalln("评论列表接收完毕")
			}
			for _,item := range commentList {
				_, err := DB.Exec("insert INTO hot_comment(movie_id,user_name,comment_text,comment_time,follow_num,create_time,update_time) values(?,?,?,?,?,?,?)",item["movie_id"],item["user_name"],item["comment_text"],item["comment_time"],item["follow_num"], time.Now().Unix(), time.Now().Unix())
				if err != nil {
					log.Println(err.Error())
				}
			}
		}
	}
}

// 格式化时间字符串为时间戳
func TimeFormTimestamp(t string) int64{
	// go语言固定日期模版
	timeLayout := "2006-01-02 15:04:05"

	timeFormat, err := time.Parse(timeLayout, t)
	if err != nil {
		log.Fatalln(err.Error())
	}

	return timeFormat.Unix()
}

// 对字符串进行UTF-8编码
func CheckUtf8(str string) string {
	if !utf8.ValidString(str) {
		return ""
	}

	return str
}



