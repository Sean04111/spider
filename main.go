package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
)

const (
	BaseSelector   = "#base-info > div > div.sc-1buquy1-1.devTPk > p"
	BaseUrl        = "https://baike.sogou.com/m/fullLemma?key="
	EventUrl       = ""
	EmptyStr       = ""
	ColumnSeletor  = "#bkcard-level1"
	TitleSelector  = "div.sc-gtx4fj-0.fhWDTV.top-title.primary-title > div.special-title-wrap > h3"
	ContentSeletor = "div.bkcard-lv1-content > p"
)

var (
	NeedMap = map[string]func(*Data, string){
		// "别名": func(data *Data, content string) {
		// 	data.Alias = content
		// },
		// "主要成就": func(data *Data, content string) {
		// 	data.Achivements = content
		// },

		// "出生日期": func(data *Data, content string) {
		// 	data.Born_in = content
		// },
		// "逝世日期": func(data *Data, content string) {
		// 	data.Died_time = content
		// },
		// // "政党": func(data *Data, content string) {
		// // 	data.Belongs_to = data.Belongs_to + "," + content
		// // },
		// // "国籍": func(data *Data, content []string) {
		// // 	data.Belongs_to = append(data.Belongs_to, content...)
		// // },
		// // "开始日期": func(d *Data, s []string) {
		// // 	d.Time_happen = append(d.Time_happen, s...)
		// // },
		// // "时间": func(d *Data, s []string) {
		// // 	d.Time_happen = append(d.Time_happen, s...)
		// // },
		// // "爆发时间": func(d *Data, s []string) {
		// // 	d.Time_happen = append(d.Time_happen, s...)
		// // },
		// // "地点": func(d *Data, s []string) {
		// // 	d.Place_happen = append(d.Place_happen, s...)
		// // },
		// // "主要人物": func(d *Data, s []string) {
		// // 	d.Participate_in = append(d.Participate_in, s...)
		// // },
		// // "领导人": func(d *Data, s []string) {
		// // 	d.Participate_in = append(d.Participate_in, s...)
		// // },
		// "出生地": func(d *Data, s string) {
		// 	d.Origin_in = s
		// },
		// "代表作品": func(d *Data, s string) {
		// 	d.Opus = s
		// },
		// "朝代": func(d *Data, s string) {
		// 	if d.Dynasty_of==EmptyStr{
		// 		d.Dynasty_of = s
		// 	}
		// },
		// "所处时代": func(d *Data, s string) {
		// 	if d.Dynasty_of==EmptyStr{
		// 		d.Dynasty_of = s
		// 	}
		// },
		// "民族": func(d *Data, s string) {
		// 	d.Ethnic_of = s
		// },
		// "庙号": func(d *Data, s string) {
		// 	d.Templename = s
		// },
		// // "陵墓位置":func(d *Data, s string) {
		// // 	d.Died_in = s
		// // },
		// "字": func(d *Data, s string) {
		// 	d.Courtesy_name = s
		// },
		"作者": func(d *Data, s string) {
			d.Author_is = s
		},
		"创作年代": func(d *Data, s string) {
			d.Dynasty_is = s
		},
		"作品出处": func(d *Data, s string) {
			d.From_is = s
		},
		"文学体裁": func(d *Data, s string) {
			d.Typs_is = s
		},
		"别名": func(d *Data, s string) {
			d.Alias_is = s
		},
		"作品原文": func(d *Data, s string) {
			d.Content = s
		},
		"作品译文": func(d *Data, s string) {
			d.Translation = s
		},
	}
)

type Data struct {
	// Name        string `json:"name"`
	// Achivements string `json:"achivements"`
	// Alias       string `json:"alias"`
	// Templename  string `json:"templename"`
	// Opus        string `json:"opus"`
	// Dynasty_of  string `json:"dynaty_of"`
	// Born_in     string `json:"born_in"`
	// Died_time   string `json:"died_time"`
	// Died_in     string `json:"died_in"`
	// Origin_in     string `json:"origin_in"`
	// Ethnic_of     string `json:"ethnic_of"`
	// Courtesy_name string `json:"courtesy_name"`

	// ++++
	Name        string `json:"name"`
	Content     string `json:"content"`
	Translation string `json:"translation"`

	Author_is  string `json:"author_is"`
	Dynasty_is string `json:"time_is"`
	From_is    string `json:"from_is"`
	Alias_is   string `json:"alias_is"`
	Typs_is    string `json:"type_is"`
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type Service struct {
	sp       *Spider
	writer   *Writer
	dataCh   chan *Data
	baseUrl  string
	keywords []string
}

func NewService(baseUrl string, filepath string) *Service {
	svr := &Service{}
	svr.sp = NewSpider()
	svr.baseUrl = baseUrl
	svr.writer = NewWriter(filepath)
	svr.dataCh = make(chan *Data, 100)
	svr.LoadCeles()
	return svr
}

func (s *Service) WorkOnKeys(keywords []string) {
	wg := sync.WaitGroup{}
	for _, key := range keywords {
		wg.Add(1)
		url := s.baseUrl + url.QueryEscape(key)
		go func() {
			fmt.Println("正在收集:", key, "...")
			data, err := s.sp.Spide(url)
			if err != nil {
				panic(err)
			}
			data.Name = key
			s.dataCh <- data
		}()
	}
	go func() {
		for {
			select {
			case data := <-s.dataCh:
				s.writer.Write(data)
				wg.Done()

			default:
			}
		}
	}()
	wg.Wait()
}

func (s *Service) LoadCeles() {
	f, err := os.Open("./cele.txt")
	if err != nil {
		panic(err)
	}
	tripm := map[string]bool{}
	reader := bufio.NewReader(f)
	for {
		lineRaw, err := reader.ReadString('\n')
		line := strings.Trim(strings.Trim(lineRaw, "\n"), " ")
		if err == io.EOF {
			if _, ok := tripm[line]; !ok {
				s.keywords = append(s.keywords, line)
				tripm[line] = true
			}
			break
		}
		if err != nil {
			panic(err)
		}
		if _, ok := tripm[line]; !ok {
			s.keywords = append(s.keywords, line)
			tripm[line] = true
		}
	}
}

func (s *Service) Work() {
	if len(s.keywords) == 0 {
		s.LoadCeles()
	}
	s.WorkOnKeys(s.keywords)
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type Spider struct {
	c         *http.Client
	nodeCount atomic.Int32
	nodeMap   sync.Map
}

func (spider *Spider) Parse(resp *http.Response) *Data {
	retData := &Data{}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		panic(err)
	}
	doc.Find(BaseSelector).Each(func(i int, s *goquery.Selection) {
		colName := s.Find("strong").Text()
		if f, ok := NeedMap[colName]; ok {
			s.Find("span").Each(func(i int, s *goquery.Selection) {
				done := false
				html, err := s.Html()
				if err != nil {
					panic(err)
				}
				if strings.Contains(html, "<br/>") {
					strs := strings.Split(html, "<br/>")
					appendin := []string{}
					for _, v := range strs {
						if strings.Contains(v, "class") {
							continue
						}
						appendin = append(appendin, v)
					}
					for _, v := range appendin {
						if _, ok := spider.nodeMap.Load(v); !ok {
							spider.nodeCount.Add(1)
							spider.nodeMap.Store(v, true)
						}
					}
					pushin := strings.Join(appendin, ",")
					f(retData, pushin)
					done = true
				}
				if !done {
					RawcolContent := s.Text()
					content := strings.Split(RawcolContent, "、")
					pushin := strings.Join(content, ",")
					for _, v := range content {
						if _, ok := spider.nodeMap.Load(v); !ok {
							spider.nodeCount.Add(1)
							spider.nodeMap.Store(v, true)
						}
					}
					f(retData, pushin)
				}

			})

		}
	})
	doc.Find(ColumnSeletor).Each(func(i int, s *goquery.Selection) {
		fmt.Println(s.Find(TitleSelector).Text())
		if f, ok := NeedMap[s.Find(TitleSelector).Text()]; ok {
			fmt.Println("ok1")
			var data string
			s.Find(ContentSeletor).Each(func(i int, small *goquery.Selection) {
				data += small.Text()
			})
			f(retData, data)
		}
	})
	return retData
}

func (s *Spider) Spide(url string) (*Data, error) {
	resp, err := s.c.Get(url)
	if err != nil {
		return nil, err
	}
	data := s.Parse(resp)
	return data, nil
}

func NewSpider() *Spider {
	spider := &Spider{}
	spider.c = http.DefaultClient
	return spider
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
type Writer struct {
	filePath string
	f        *os.File
}

func NewWriter(filepath string) *Writer {
	w := &Writer{}
	w.filePath = filepath
	var err error
	w.f, err = os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return w
}

func (w *Writer) Write(data *Data) {

	// if data.Achivements == EmptyStr && data.Alias == EmptyStr && data.Born_in == EmptyStr && data.Courtesy_name == EmptyStr && data.Died_time == EmptyStr && data.Dynasty_of == EmptyStr && data.Ethnic_of == EmptyStr && data.Opus == EmptyStr && data.Origin_in == EmptyStr && data.Templename == EmptyStr {
	// 	return
	// }

	code, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	_, err = w.f.Write(code)
	w.f.WriteString("\n")
	if err != nil {
		panic(err)
	}
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func main() {
	s := NewService(BaseUrl, "./history.json")
	s.WorkOnKeys([]string{"静夜思"})
}
