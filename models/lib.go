package models

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/csuhan/csugo/utils"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const LIB_LOGIN_URL = "http://opac.its.csu.edu.cn/NTRdrLogin.aspx"
const LIB_BOOK_URL = "http://opac.its.csu.edu.cn/NTBookLoanRetr.aspx"
const LIB_REBORROW_URL = "http://opac.its.csu.edu.cn/NTBookloanResult.aspx"

type Lib struct {
}

type Book struct {
	BarCode, BookName, BookNo, Author, Place, BorrowedDate, ReturnedDate, Price, BorrowTimes, ReloanRes string
}

//登录系统
func (this *Lib) Login(ID, Pwd string) (*http.Client, error) {
	resp, err := http.Get(LIB_LOGIN_URL)
	if err != nil {
		return nil, utils.ERROR_SERVER
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, utils.ERROR_SERVER
	}
	reqData := url.Values{
		"txtName":     {ID},
		"txtPassWord": {Pwd},
		"Logintype":   {"RbntRecno"},
		"BtnLogin":    {"登 录"},
	}
	reqData.Add("__VIEWSTATE", doc.Find("#__VIEWSTATE").AttrOr("value", ""))
	reqData.Add("__VIEWSTATEGENERATOR", doc.Find("#__VIEWSTATEGENERATOR").AttrOr("value", ""))
	reqData.Add("__EVENTVALIDATION", doc.Find("#__EVENTVALIDATION").AttrOr("value", ""))

	req, err := http.NewRequest("POST", LIB_LOGIN_URL, strings.NewReader(reqData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return nil, utils.ERROR_SERVER
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, utils.ERROR_SERVER
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	//账号密码错误
	if !strings.Contains(string(data), "图书续借") {
		return nil, utils.ERROR_ID_PWD
	}
	//登录成功
	return client, nil
}

//借阅列表
func (this *Lib) List(ID, Pwd string) ([]Book, error) {
	client, err := this.Login(ID, Pwd)
	if err != nil { //登录失败返回
		return []Book{}, err
	}
	//新建请求,获取借书列表
	req, _ := http.NewRequest("GET", LIB_BOOK_URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		return []Book{}, utils.ERROR_SERVER
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return []Book{}, utils.ERROR_SERVER
	}
	book := Book{}
	books := []Book{}
	doc.Find("#flexitable tbody tr").Each(func(i int, s *goquery.Selection) {
		td := s.Find("td")
		book.BarCode = td.Eq(1).Text()
		book.BookName = td.Eq(2).Text()
		book.BookNo = td.Eq(3).Text()
		book.Author = td.Eq(4).Text()
		book.Place = td.Eq(5).Text()
		book.BorrowedDate = td.Eq(6).Text()
		book.ReturnedDate = td.Eq(7).Text()
		book.Price = td.Eq(8).Text()
		book.BorrowTimes = td.Eq(9).Text()

		books = append(books, book)
	})
	return books, nil
}

//图书续借
func (this *Lib) Borrow(ID, Pwd string, BarCodeLists []string) ([]Book, error) {
	//登录获取cookie
	client, err := this.Login(ID, Pwd)
	if err != nil {
		return []Book{}, err
	}
	reqData := "?barno="
	for _, barNo := range BarCodeLists {
		reqData = reqData + barNo + ";"
	}
	req, _ := http.NewRequest("GET", LIB_REBORROW_URL+reqData, nil)
	resp, err := client.Do(req)
	if err != nil {
		return []Book{}, utils.ERROR_SERVER
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return []Book{}, utils.ERROR_SERVER
	}
	//解析页面
	books := []Book{}
	fmt.Println(doc.Html())
	doc.Find("#flexitable tbody tr").Each(func(i int, s *goquery.Selection) {
		td := s.Find("td")
		book := Book{
			BarCode:      td.Eq(1).Text(),
			BookName:     td.Eq(2).Text(),
			BookNo:       td.Eq(3).Text(),
			BorrowedDate: td.Eq(4).Text(),
			ReturnedDate: td.Eq(5).Text(),
			BorrowTimes:  td.Eq(6).Text(),
		}
		book.ReloanRes = strings.Trim(td.Eq(0).Text(), "\n ")
		if strings.Contains(book.ReloanRes, "续借成功,可返回查看结果") {
			book.ReloanRes = "续借成功"
		}
		if strings.Contains(book.ReloanRes, "超过续借次数, 不能续借") {
			book.ReloanRes = "超过续借次数"
		}
		books = append(books, book)
	})
	return books, nil
}
