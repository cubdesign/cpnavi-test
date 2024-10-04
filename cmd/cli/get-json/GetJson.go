package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func getURLsForUniversity(host string, csvFile string) ([]string, []string, error) {
	file, err := os.Open(csvFile)
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
		url := fmt.Sprintf("%s/%s/%s", host, "university", slug)
		urls = append(urls, url)
		slugs = append(slugs, slug)
	}

	return urls, slugs, nil
}

func getURLsForMajor(host string, csvFile string) ([]string, []string, error) {
	file, err := os.Open(csvFile)
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
		url := fmt.Sprintf("%s/university/%s/major/%s", host, universitySlug, majorCode)
		urls = append(urls, url)

		// 大学とメジャーのパスを構成
		path := filepath.Join(universitySlug, majorCode)
		paths = append(paths, path)
	}

	return urls, paths, nil
}

func getURLs(api string, apiHost string, csvFile string) ([]string, []string) {
	var urls, paths []string
	var err error

	if api == "university" {
		urls, paths, err = getURLsForUniversity(apiHost, csvFile)
	} else if api == "major" {
		urls, paths, err = getURLsForMajor(apiHost, csvFile)
	} else {
		fmt.Println("Error: Unknown api")
		return nil, nil
	}

	if err != nil {
		fmt.Println("Error:", err)
		return nil, nil
	}

	return urls, paths
}

func fetchAndSaveJSON(label string, api string, accessToken string, exportFolder string, url string, universitySlug string, majorSlug string) error {

	// 1. HTTPリクエストを作成
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

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
	saveFolder := filepath.Join(exportFolder, label, api)

	// フォルダが存在しない場合は作成
	if err := os.MkdirAll(saveFolder, os.ModePerm); err != nil {
		return err
	}

	// ファイル名の構成
	var fileName string
	if api == "university" {
		// universityの場合、ファイル名は universitySlug.json
		fileName = filepath.Join(saveFolder, fmt.Sprintf("%s.json", universitySlug))
	} else if api == "major" {
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

func removeFile(exportFolder string, label string, api string) {
	// 出力を削除
	dir := filepath.Join(exportFolder, label, api)
	// ディレクトリを再帰的に削除
	err := os.RemoveAll(dir)
	if err != nil {
		fmt.Printf("ディレクトリの削除中にエラーが発生しました: %v\n", err)
	} else {
		fmt.Printf("ディレクトリ '%s' を削除しました。\n", dir)
	}

}

func getJson(apiHost string, csvFile string, label string, api string, accessToken string, exportFolder string) {

	// getURLsから universitySlug と (必要なら) majorSlug を取得
	urls, paths := getURLs(api, apiHost, csvFile)

	// 出力フォルダを削除
	removeFile(exportFolder, label, api)

	for i, url := range urls {
		fmt.Printf("Fetching URL %d: %s\n", i+1, url)

		if api == "university" {
			// universityの場合は paths[i] = "universitySlug"
			universitySlug := paths[i]
			err := fetchAndSaveJSON(label, api, accessToken, exportFolder, url, universitySlug, "")
			if err != nil {
				fmt.Printf("Error fetching URL %d: %s\n", i+1, err)
			}
		} else if api == "major" {
			// majorの場合は paths[i] = "universitySlug/majorSlug"
			pathParts := strings.Split(paths[i], "/")
			if len(pathParts) == 2 {
				universitySlug := pathParts[0]
				majorSlug := pathParts[1]
				err := fetchAndSaveJSON(label, api, accessToken, exportFolder, url, universitySlug, majorSlug)
				if err != nil {
					fmt.Printf("Error fetching URL %d: %s\n", i+1, err)
				}
			} else {
				fmt.Printf("Error: Invalid path format for major at index %d: %s\n", i+1, paths[i])
			}
		} else {
			fmt.Printf("Error: Unknown api %s\n", api)
		}
	}
}

func main() {

	// コマンドライン引数としてディレクトリパスを受け取る
	label := flag.String("label", "", "環境ラベル (local, production, development)")
	api := flag.String("api", "", "API (university, major)")
	apiHost := flag.String("apiHost", "", "APIホスト")
	csvFile := flag.String("csv", "", "CSVファイルのパス")
	accessToken := flag.String("accessToken", "", "アクセストークン")
	exportFolder := flag.String("exportFolder", "", "エクスポートフォルダ")

	// 引数の解析
	flag.Parse()

	//label := "local"
	//label := "production"
	//label := "development"

	//api := "university"
	//api := "major"

	//apiHost := "http://localhost:8081"
	//apiHost := "https://cpnavi-api.l-interface.co.jp"
	//apiHost := "https://navi-api.collegepathway.jp"

	//csvFile := "import/university_slug.csv"
	//csvFile := "import/university_slug_and_major_code.csv"

	//exportFolder := "export"

	//accessToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjgwMTI1MDEsImlhdCI6MTcyODAwODkwMSwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDozMDAwIiwicGFyZW50IjowLCJyb2xlIjoic3R1ZGVudCIsInVpZCI6MTUzfQ.0w0edgIykzXjC156lXl9wpjbsEZNrgFVhZEnhGOKDjqTnk4NGPXU7BC9CE-TipESLbRwFpqpmUKXEBtrcr7lh218v2A81JchIAMHcgpksmCHYa557NTsmAu12H54KbL8y6oCm6_tYZyOokcP-MnGZw5SrjewqLHDt3To5gbGMhb9S4lII1qoHUc0kkmnDkocTxxfYz9x5tWBWMcGIFVUE79IWOHKpYQGhOCJ2s2l2eJoVZvxY_QvYby23g16RhOVAJ8sFOm7hZED91rIBnNp_r7TnIDIu3YGjKBNJTzKP1x_OVNfHvwzcOF0XLQyG_uzjhDAwsx2eRCZskL1g55YIA"

	// 引数が指定されていない場合はエラーメッセージを表示して終了
	if *label == "" {
		fmt.Println("Error: -label が指定されていません")
		flag.Usage() // 使用方法の表示
		os.Exit(1)
	}
	if *api == "" {
		fmt.Println("Error: -api が指定されていません")
		flag.Usage()
		os.Exit(1)
	}
	if *apiHost == "" {
		fmt.Println("Error: -apiHost が指定されていません")
		flag.Usage()
		os.Exit(1)

	}

	if *csvFile == "" {
		fmt.Println("Error: -csv が指定されていません")
		flag.Usage()
		os.Exit(1)
	}

	if *accessToken == "" {
		fmt.Println("Error: -accessToken が指定されていません")
		flag.Usage()
		os.Exit(1)
	}

	if *exportFolder == "" {
		fmt.Println("Error: -exportFolder が指定されていません")
		flag.Usage()
		os.Exit(1)
	}

	// csvFileが存在するかチェック
	if _, err := os.Stat(*csvFile); os.IsNotExist(err) {
		fmt.Printf("Error: CSVファイルが存在しません: %s\n", *csvFile)
		os.Exit(1)
	}

	// ディレクトリが存在するかチェック
	if _, err := os.Stat(*exportFolder); os.IsNotExist(err) {
		fmt.Printf("Error: エクスポートフォルダが存在しません: %s\n", *exportFolder)
		os.Exit(1)
	}

	getJson(*apiHost, *csvFile, *label, *api, *accessToken, *exportFolder)
}
