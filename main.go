package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

func main() {
	tempList := make([]string, 0)
	list, err := getDone("./done.txt")
	if err != nil {
		logrus.Fatal(err)
	}
	firstUrl := "https://hosocongty.vn/thang-02/2022"
	total, err := getPageList(firstUrl)
	if err != nil {
		logrus.Fatal(err)
	}

	firstPageRecords, err := getListCompanyUrl(firstUrl)
	if err != nil {
		logrus.Fatal(err)
	}
	numb := 1
	for _, record := range firstPageRecords {
		fmt.Println(numb, record.Url)
		flag := false
		for i := 0; i < len(list); i++ {
			if list[i] == record.Url {
				flag = true
			}
		}
		if !flag {
			err := getDetail(&record, tempList)
			if err != nil {
				if err.Error() == "exists" {
					fmt.Println(err)
				} else {
					fmt.Println(err)
					return
				}
			}
			tempList = append(tempList, record.Url)
			WriteToFile(record.Url, "./done.txt")
		}
		numb++
	}
	for i := 1; i <= *total; i++ {
		url := fmt.Sprintf(firstUrl+"/page-%d", i)
		items, err := getListCompanyUrl(url)
		if err != nil {
			logrus.Fatal(err)
		}
		for _, record := range items {
			fmt.Println(numb, record.Url)
			flag := false
			for i := 0; i < len(list); i++ {
				if list[i] == record.Url {
					flag = true
				}
			}
			if !flag {
				// go getDetail(&record, tempList)
				err := getDetail(&record, tempList)
				if err != nil {
					if err.Error() == "exists" {
						fmt.Println(err)
					} else {
						fmt.Println(err)
						return
					}
				}
			}
			tempList = append(tempList, record.Url)
			WriteToFile(record.Url, "./done.txt")
			numb++
		}
	}

}

type Record struct {
	Url         string //
	Name        string //
	TaxNumber   string //
	Address     string //
	PhoneNumber string
	Deputy      string
	MainField   string
	LisenceDate string
	Status      string
}

func getPageList(firstUrl string) (*int, error) {
	firstRes, err := resty.New().R().Post(firstUrl)
	if err != nil {
		return nil, err
	}
	if !firstRes.IsSuccess() {
		return nil, fmt.Errorf("status: %s, res: %s", firstRes.Status(), string(firstRes.Body()))
	}
	doc, err := html.Parse(strings.NewReader(string(firstRes.Body())))
	if err != nil {
		return nil, err
	}
	var crawler func(*html.Node)
	value := 0
	crawler = func(node *html.Node) {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Data == "input" {
				flag := false
				for _, attr := range child.Attr {
					if attr.Key == "name" {
						if attr.Val == "total" {
							flag = true
							break
						}
					}
				}
				if flag {
					for _, attr := range child.Attr {
						if attr.Key == "value" {
							value, _ = strconv.Atoi(attr.Val)
							break
						}
					}
					break
				}
			}
			crawler(child)
		}
	}
	crawler(doc)
	return &value, nil
}

func getListCompanyUrl(firstUrl string) ([]Record, error) {
	records := make([]Record, 0)

	firstRes, err := resty.New().R().Post(firstUrl)
	if err != nil {
		return nil, err
	}
	if !firstRes.IsSuccess() {
		return nil, fmt.Errorf("status: %s, res: %s", firstRes.Status(), string(firstRes.Body()))
	}
	r := regexp.MustCompile(`<ul class="hsdn">(.*)</ul>`)
	ulPart := r.FindAllString(string(firstRes.Body()), -1)
	for _, v := range ulPart {
		ul := string(v)
		doc, err := html.Parse(strings.NewReader(ul))
		if err != nil {
			return nil, err
		}
		hrefs, err := ListAnchor(doc)
		if err != nil {
			return nil, err
		}
		for _, href := range hrefs {
			url := "https://hosocongty.vn/" + href
			record := Record{
				Url: url,
			}
			flag := false
			for _, existRecord := range records {
				if existRecord.Url == record.Url {
					flag = true
					break
				}
			}
			if !flag {
				records = append(records, record)
			}
		}
	}
	return records, nil
}

func ListAnchor(doc *html.Node) ([]string, error) {
	result := make([]string, 0)
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Data == "a" {
				for _, attr := range child.Attr {
					if attr.Key == "href" {
						flag := false
						for i := 0; i < len(result); i++ {
							if result[i] == attr.Val {
								flag = true
							}
						}
						if !flag {
							result = append(result, attr.Val)
						}
					}
				}
			}
			crawler(child)
		}
	}
	crawler(doc)
	return result, nil
}

func getDetail(record *Record, tempList []string) error {
	list, err := getDone("./done.txt")
	if err != nil {
		logrus.Fatal(err)
	}
	for i := 0; i < len(list); i++ {
		if list[i] == record.Url {
			return fmt.Errorf("exists")
		}
	}
	// for i := 0; i < len(tempList); i++ {
	// 	if list[i] == record.Url {
	// 		return fmt.Errorf("exists")
	// 	}
	// }
	firstRes, err := resty.New().R().Post(record.Url)
	if err != nil {
		return err
	}
	if !firstRes.IsSuccess() {
		return fmt.Errorf("status: %s, res: %s", firstRes.Status(), string(firstRes.Body()))
	}
	// fmt.Println(string(firstRes.Body()))
	r1 := regexp.MustCompile(`<ul class='hsct'><li><h1>(.*)</h1></li><li>`)
	companyNamePart := r1.FindAllString(string(firstRes.Body()), -1)
	for _, v := range companyNamePart {
		v := strings.Replace(v, "<ul class='hsct'><li><h1>", "", -1)
		v = strings.Replace(v, "</h1></li><li>", "", -1)
		record.Name = v
		break
	}
	// <li><label><i class="fa fa-map-marker"></i> ?????a ch??? thu???:</label> <span>
	r2 := regexp.MustCompile(`<li><label><i class="fa fa-map-marker"></i> ?????a ch??? thu???:</label> <span>(.*)</span></li></ul><ul class='hsct'>`)
	addressPart := r2.FindAllString(string(firstRes.Body()), -1)
	for _, v := range addressPart {
		v := strings.Replace(v, `<li><label><i class="fa fa-map-marker"></i> ?????a ch??? thu???:</label> <span>`, "", -1)
		v = strings.Replace(v, "</span></li></ul><ul class='hsct'>", "", -1)
		record.Address = v
		break
	}
	// <li><label><i class="fa fa-hashtag"></i> M?? s??? thu???:</label> <span>
	r3 := regexp.MustCompile(`<li><label><i class="fa fa-hashtag"></i> M?? s??? thu???:</label> <span>(.*)</span></li><li><label><i class="fa fa-map-marker"></i> ?????a ch??? thu???:`)
	taxNumbPart := r3.FindAllString(string(firstRes.Body()), -1)
	for _, v := range taxNumbPart {
		v := strings.Replace(v, `<li><label><i class="fa fa-hashtag"></i> M?? s??? thu???:</label> <span>`, "", -1)
		v = strings.Replace(v, `</span></li><li><label><i class="fa fa-map-marker"></i> ?????a ch??? thu???:`, "", -1)
		record.TaxNumber = v
		break
	}
	// <ul class='hsct'><li><label><i class="fa fa-user-o"></i> ?????i di???n ph??p lu???t:</label><span><a href="search?key=
	r4 := regexp.MustCompile(`<li><label><i class="fa fa-user-o"></i> ?????i di???n ph??p lu???t:</label>(.*)</a></span></li><li><label><i class="`)
	ownerPart := r4.FindAllString(string(firstRes.Body()), -1)
	for _, v := range ownerPart {
		v := strings.Replace(v, `<li><label><i class="fa fa-user-o"></i> ?????i di???n ph??p lu???t:</label>`, "", -1)
		v = strings.Replace(v, `</a></span></li><li><label><i class="`, "", -1)
		tempArr := strings.Split(v, `&opt=1" title="`)
		tempArr[0] = strings.ReplaceAll(tempArr[0], `<span><a href="search?key=`, "")
		record.Deputy = tempArr[0]
		break
	}
	r5 := regexp.MustCompile(`<i class="fa fa fa-phone"></i> ??i???n tho???i:</label><span class='highlight'>(.*)</span></li><li><label><i class="fa fa-calendar"></i>`)
	phonePart := r5.FindAllString(string(firstRes.Body()), -1)
	for _, v := range phonePart {
		v := strings.Replace(v, `<i class="fa fa fa-phone"></i> ??i???n tho???i:</label><span class='highlight'>`, "", -1)
		v = strings.Replace(v, `</span></li><li><label><i class="fa fa-calendar"></i>`, "", -1)
		tempArr := strings.Split(v, `</span></li><li><label><i class="fa fa-envelope-o"></i>`)
		record.PhoneNumber = tempArr[0]
		break
	}
	r6 := regexp.MustCompile(`<li><label><i class="fa fa-calendar"></i> Ng??y c???p:</label><span> <a href="ngay-(.*)" title="Danh s??ch c??ng ty th??nh l???p`)
	datePart := r6.FindAllString(string(firstRes.Body()), -1)
	for _, v := range datePart {
		v := strings.Replace(v, `<li><label><i class="fa fa-calendar"></i> Ng??y c???p:</label><span> <a href="ngay-`, "", -1)
		v = strings.Replace(v, `" title="Danh s??ch c??ng ty th??nh l???p`, "", -1)
		record.LisenceDate = v
		break
	}
	r7 := regexp.MustCompile(`<label><i class="fa fa-info"></i> Tr???ng th??i:</label><span>(.*)</span></li><li><i class="fa fa-question-circle"></i>`)
	statusPart := r7.FindAllString(string(firstRes.Body()), -1)
	for _, v := range statusPart {
		v := strings.Replace(v, `<label><i class="fa fa-info"></i> Tr???ng th??i:</label><span>`, "", -1)
		v = strings.Replace(v, `</span></li><li><i class="fa fa-question-circle"></i>`, "", -1)
		record.Status = v
		break
	}
	r8 := regexp.MustCompile(`<li><label><i class="fa fa-anchor"></i> Ng??nh ngh??? ch??nh:</label><span>(.*)</a></span></li><li><label><i class=`)
	mainFieldPart := r8.FindAllString(string(firstRes.Body()), -1)
	for _, v := range mainFieldPart {
		v := strings.Replace(v, `<li><label><i class="fa fa-anchor"></i> Ng??nh ngh??? ch??nh:</label><span>`, "", -1)
		v = strings.Replace(v, `</a></span></li><li><label><i class=`, "", -1)
		tempArr := strings.Split(v, `">`)
		record.MainField = tempArr[len(tempArr)-1]
		break
	}
	return WriteToFile(BuildString(record), "./data.csv")
}

func BuildString(record *Record) string {
	str := fmt.Sprintf(`%s,%s,"%s","%s",%s,"%s",%s,%s,"%s"`, record.TaxNumber, record.PhoneNumber, record.Name, record.Deputy, record.LisenceDate, record.MainField, record.Status, record.Url, record.Address)
	return str
}

func WriteToFile(str, path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(str + "\n")
	return err
}

func getDone(file string) ([]string, error) {
	// method open is read only
	res := make([]string, 0)
	f, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		res = append(res, string(scanner.Bytes()))
	}
	if err = scanner.Err(); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return res, nil
}
