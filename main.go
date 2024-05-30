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

	"github.com/PuerkitoBio/goquery"
)

const (
	BaseSelector = "#base-info > div > div.sc-1buquy1-1.devTPk > p"
	BaseUrl      = "https://baike.sogou.com/m/fullLemma?key="
	EventUrl     = ""
)

var (
	NeedMap = map[string]func(*Data, []string){
		"别名": func(data *Data, content []string) {
			data.Alias = append(data.Alias, content...)
		},
		"主要成就": func(data *Data, content []string) {
			data.Achivements = append(data.Achivements, content...)
		},
		// "字": func(data *Data, content []string) {
		// 	data.Pseudonym = append(data.Pseudonym, content...)
		// },
		// "生平": func(data *Data, content []string) {
		// 	data.Deeds = append(data.Deeds, content...)
		// },
		// "领导": func(data *Data, content []string) {
		// 	data.Participate_in = append(data.Participate_in, content...)
		// },
		// "主要指挥官": func(data *Data, content []string) {
		// 	data.Participate_in = append(data.Participate_in, content...)
		// },
		"出生日期": func(data *Data, content []string) {
			data.Born_in = append(data.Born_in, content...)
		},
		"逝世日期": func(data *Data, content []string) {
			data.Died_time = append(data.Died_time, content...)
		},
		"政党": func(data *Data, content []string) {
			data.Belongs_to = append(data.Belongs_to, content...)
		},
		"国籍": func(data *Data, content []string) {
			data.Belongs_to = append(data.Belongs_to, content...)
		},
		// "开始日期": func(d *Data, s []string) {
		// 	d.Time_happen = append(d.Time_happen, s...)
		// },
		// "时间": func(d *Data, s []string) {
		// 	d.Time_happen = append(d.Time_happen, s...)
		// },
		// "爆发时间": func(d *Data, s []string) {
		// 	d.Time_happen = append(d.Time_happen, s...)
		// },
		// "地点": func(d *Data, s []string) {
		// 	d.Place_happen = append(d.Place_happen, s...)
		// },
		// "主要人物": func(d *Data, s []string) {
		// 	d.Participate_in = append(d.Participate_in, s...)
		// },
		// "领导人": func(d *Data, s []string) {
		// 	d.Participate_in = append(d.Participate_in, s...)
		// },
		"出生地": func(d *Data, s []string) {
			d.Origin_in = append(d.Origin_in, s...)
		},
		"代表作品": func(d *Data, s []string) {
			d.Opus = append(d.Opus, s...)
		},
		"毕业院校": func(d *Data, s []string) {
			d.Graduated_from = append(d.Graduated_from, s...)
		},
	}
)

type Data struct {
	Name string `json:"name"`
	// Deeds       []string `json:deeds`
	Alias []string `json:"alias"`
	// Pseudonym   []string `json:pseudonym`
	Achivements []string `json:"achivements"`
	// Lead_to        []string `json:lead_to`
	// Participate_in []string `json:"participate_in"`  暂时没有想到办法解决
	Born_in   []string `json:"born_in"`
	Died_time []string `json:"died_time"`
	Origin_in []string `json:"origin_in"`
	// Time_happen    []string `json:time_happen`
	// Place_happen   []string `json:place_happen`
	Belongs_to     []string `json:"belongs_to"`
	Opus           []string `json:"opus"`
	Graduated_from []string `json:"graduated_from"`
}

func (d *Data) Format() {
	// if len(d.Lead_to) != 0 {
	// 	d.Lead_to = append(d.Lead_to, []string{"领导", d.Name}...)
	// }
	if len(d.Alias) != 0 {
		d.Alias = append([]string{d.Name, "别名"}, d.Alias...)
	}
	// if len(d.Participate_in) != 0 {
	// 	d.Participate_in = append([]string{d.Name, "参与"}, d.Participate_in...)
	// }
	if len(d.Born_in) != 0 {
		d.Born_in = append([]string{d.Name, "出生于"}, d.Born_in...)
	}
	if len(d.Died_time) != 0 {
		d.Died_time = append([]string{d.Name, "卒于"}, d.Died_time...)
	}
	if len(d.Origin_in) != 0 {
		d.Origin_in = append([]string{d.Name, "籍贯"}, d.Origin_in...)
	}
	// if len(d.Time_happen) != 0 {
	// 	d.Time_happen = append([]string{d.Name, "发生自"}, d.Time_happen...)
	// }
	// if len(d.Place_happen) != 0 {
	// 	d.Place_happen = append([]string{d.Name, "发生在"}, d.Place_happen...)
	// }
	if len(d.Belongs_to) != 0 {
		d.Belongs_to = append([]string{d.Name, "从属"}, d.Belongs_to...)
	}
	if len(d.Opus) != 0 {
		d.Opus = append([]string{d.Name, "主要作品"}, d.Opus...)
	}
	if len(d.Achivements) != 0 {
		d.Achivements = append([]string{d.Name, "主要成就"}, d.Achivements...)
	}
	if len(d.Graduated_from) != 0 {
		d.Graduated_from = append([]string{d.Name, "毕业于"}, d.Graduated_from...)
	}
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
			data.Format()
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
	c *http.Client
}

func (s *Spider) Parse(resp *http.Response) *Data {
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
					strs:=strings.Split(html, "<br/>")
					appendin:=[]string{}
					for _,v:=range strs{
						if strings.Contains(v,"class"){
							continue
						}
						appendin = append(appendin,v)
					}
					f(retData,appendin)
					done = true
				}
				if !done {
					RawcolContent := s.Text()
					content := strings.Split(RawcolContent, "、")
					f(retData, content)
				}

			})

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
	s.Work()
}
