package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type APIResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var CP_ENV = "production" // "production" or "development"
var CP_TYPE = "major"     // "university" or "major"
var CP_TOKEN = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mjc5NTM0NTksImlhdCI6MTcyNzk0OTg1OSwiaXNzIjoiaHR0cHM6Ly9uYXZpLmNvbGxlZ2VwYXRod2F5LmpwIiwicGFyZW50IjowLCJyb2xlIjoic3R1ZGVudCIsInVpZCI6ODg1fQ.x1VGY-HD6Xk-vv2EyRQAUUskrG0svTFwQmXPjfQV7PpZYjIIdEtu1Z0v0gdd2IwM4EMHmFHqHa73tK_CeMWkZF_E5t31-EVDGVWaUHUS2y6_esn__y5f2xnbXZRu7k54_oyGvVznfRH2OLnOJZ_V8WjtFmq_hd1rlLhQzN09-B9tBidPt2juAvsvdQtIW8tGigQZ6sHNaWb9ipmWV2igIC_4ZfS5fFmAd7Ue5f3TBhNAm4TpvBZWlfaYjAWDlrkoR-XzzUAscOZ1ADdY_KAHVN2asNJ_k3-yMLU03L_Y4otIFc_sqJkUcQkkEHeZHlg223hHTUSTsNvbwLCBKX1WHg"

var CP_HOST_DEVELOPMENT = "https://cpnavi-api.l-interface.co.jp"
var CP_HOST_PRODUCTION = "https://navi-api.collegepathway.jp"

var CP_HOST = getHost()

func getHost() string {
	if CP_ENV == "production" {
		return CP_HOST_PRODUCTION
	}
	return CP_HOST_DEVELOPMENT
}

func getURLsForUniversity() ([]string, []string, error) {
	file, err := os.Open("import/university_slug.csv")
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	var urls []string
	var slugs []string

	for _, record := range records {
		slug := record[0]
		url := fmt.Sprintf("%s/%s/%s", CP_HOST, CP_TYPE, slug)
		urls = append(urls, url)
		slugs = append(slugs, slug)
	}

	return urls, slugs, nil
}

func getURLsForMajor() ([]string, []string, error) {
	file, err := os.Open("import/university_slug_and_major_code.csv")
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	var urls []string
	var paths []string

	for _, record := range records {
		if len(record) < 2 {
			continue
		}
		universitySlug := record[0]
		majorCode := record[1]
		url := fmt.Sprintf("%s/university/%s/major/%s", CP_HOST, universitySlug, majorCode)
		urls = append(urls, url)

		// 大学とメジャーのパスを構成
		path := filepath.Join(universitySlug, majorCode)
		paths = append(paths, path)
	}

	return urls, paths, nil
}

func getURLs() ([]string, []string) {
	var urls, paths []string
	var err error

	if CP_TYPE == "university" {
		urls, paths, err = getURLsForUniversity()
	} else if CP_TYPE == "major" {
		urls, paths, err = getURLsForMajor()
	} else {
		fmt.Println("Error: Unknown CP_TYPE")
		return nil, nil
	}

	if err != nil {
		fmt.Println("Error:", err)
		return nil, nil
	}

	return urls, paths
}

func fetchAndSaveJSON(url string, universitySlug string, majorSlug string) error {
	// 1. HTTPリクエストを作成
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+CP_TOKEN)

	// 2. HTTPクライアントを作成してリクエストを送信
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 3. レスポンスボディを読み取る
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 4. 汎用的にJSONをパース（interface{}を使用）
	var apiResponse interface{}
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return err
	}

	// フォルダパスは "export/CP_ENV/CP_TYPE"
	saveFolder := filepath.Join("export", CP_ENV, CP_TYPE)

	// フォルダが存在しない場合は作成
	if err := os.MkdirAll(saveFolder, os.ModePerm); err != nil {
		return err
	}

	// ファイル名の構成
	var fileName string
	if CP_TYPE == "university" {
		// universityの場合、ファイル名は universitySlug.json
		fileName = filepath.Join(saveFolder, fmt.Sprintf("%s.json", universitySlug))
	} else if CP_TYPE == "major" {
		// majorの場合、ファイル名は universitySlug/majorSlug.json
		saveFolder = filepath.Join(saveFolder, universitySlug)
		if err := os.MkdirAll(saveFolder, os.ModePerm); err != nil {
			return err
		}
		fileName = filepath.Join(saveFolder, fmt.Sprintf("%s.json", majorSlug))
	}

	// 5. JSONデータをファイルに保存
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // 整形して保存
	err = encoder.Encode(apiResponse)
	if err != nil {
		return err
	}

	fmt.Printf("JSONデータがファイルに保存されました: %s\n", fileName)
	return nil
}

func main() {
	urls, paths := getURLs() // getURLsから universitySlug と (必要なら) majorSlug を取得

	for i, url := range urls {
		fmt.Printf("Fetching URL %d: %s\n", i+1, url)

		if CP_TYPE == "university" {
			// universityの場合は paths[i] = "universitySlug"
			universitySlug := paths[i]
			err := fetchAndSaveJSON(url, universitySlug, "")
			if err != nil {
				fmt.Printf("Error fetching URL %d: %s\n", i+1, err)
			}
		} else if CP_TYPE == "major" {
			// majorの場合は paths[i] = "universitySlug/majorSlug"
			pathParts := strings.Split(paths[i], "/")
			if len(pathParts) == 2 {
				universitySlug := pathParts[0]
				majorSlug := pathParts[1]
				err := fetchAndSaveJSON(url, universitySlug, majorSlug)
				if err != nil {
					fmt.Printf("Error fetching URL %d: %s\n", i+1, err)
				}
			} else {
				fmt.Printf("Error: Invalid path format for major at index %d: %s\n", i+1, paths[i])
			}
		} else {
			fmt.Printf("Error: Unknown CP_TYPE %s\n", CP_TYPE)
		}
	}
}
