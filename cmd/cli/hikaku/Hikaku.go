package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// JSONファイルを読み込んでmapとして返す関数
func loadJSON(filePath string) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// JSONデータを読み込む
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// JSONデータをmapにパース
	var jsonData map[string]interface{}
	if err := json.Unmarshal(bytes, &jsonData); err != nil {
		return nil, err
	}

	return jsonData, nil
}

// スライスの要素を順序に関係なく比較する
func compareSlices(slice1, slice2 []interface{}) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	// スライスをソートして比較するため、文字列化して比較
	stringifiedSlice1 := stringifySlice(slice1)
	stringifiedSlice2 := stringifySlice(slice2)

	// ソート
	sort.Strings(stringifiedSlice1)
	sort.Strings(stringifiedSlice2)

	// ソート後に比較
	return reflect.DeepEqual(stringifiedSlice1, stringifiedSlice2)
}

// スライスを文字列に変換
func stringifySlice(slice []interface{}) []string {
	var result []string
	for _, elem := range slice {
		// 各要素をJSON形式に変換してから文字列として扱う
		jsonElem, _ := json.Marshal(elem)
		result = append(result, string(jsonElem))
	}
	return result
}

// 数値を比較するための関数（誤差を許容しない）
func compareNumbers(devValue, prodValue float64) bool {
	return devValue == prodValue
}

func compareValues(path string, devValue, prodValue interface{}) bool {
	switch devVal := devValue.(type) {
	case float64:
		if prodVal, ok := prodValue.(float64); ok {
			// 数値の比較（指数形式と整数形式の違いを許容）
			if !compareNumbers(devVal, prodVal) {
				fmt.Printf("Difference in '%s':\n  dirA: %.0f\n  dirB: %.0f\n\n", path, devVal, prodVal)
				return true
			}
		} else {
			fmt.Printf("Difference in '%s':\n  dirA: %.0f\n  dirB: %v\n\n", path, devVal, prodValue)
			return true
		}
	case []interface{}:
		if prodVal, ok := prodValue.([]interface{}); ok {
			// スライスを順序に関係なく比較
			if !compareSlices(devVal, prodVal) {
				fmt.Printf("Difference in '%s':\n  dirA: %v\n  dirB: %v\n\n", path, devValue, prodValue)
				return true
			}
		} else {
			fmt.Printf("Difference in '%s':\n  dirA: %v\n  dirB: %v\n\n", path, devValue, prodValue)
			return true
		}
	case map[string]interface{}:
		if prodVal, ok := prodValue.(map[string]interface{}); ok {
			// 再帰的にマップを比較
			if compareJSONContent(devVal, prodVal, path) {
				return true
			}
		} else {
			fmt.Printf("Difference in '%s':\n  dirA: %v\n  dirB: %v\n\n", path, devValue, prodValue)
			return true
		}
	default:
		if !reflect.DeepEqual(devValue, prodValue) {
			fmt.Printf("Difference in '%s':\n  dirA: %v\n  dirB: %v\n\n", path, devValue, prodValue)
			return true
		}
	}
	return false
}

// 2つのJSONの内容を比較し、異なる部分があればtrueを返す
func compareJSONContent(devJSON, prodJSON map[string]interface{}, path string) bool {
	hasDifferences := false

	for key, devValue := range devJSON {
		if prodValue, exists := prodJSON[key]; exists {
			// キーが存在する場合、値を比較
			if compareValues(path+"."+key, devValue, prodValue) {
				hasDifferences = true
			}
		} else {
			// キーが存在しない場合
			fmt.Printf("Key '%s.%s' is missing in dirB.\n\n", path, key)
			hasDifferences = true
		}
	}

	// dirBにあってdirAにないキーもチェック
	for key := range prodJSON {
		if _, exists := devJSON[key]; !exists {
			fmt.Printf("Key '%s.%s' is missing in dirA.\n\n", path, key)
			hasDifferences = true
		}
	}

	return hasDifferences
}

// JSONファイルの比較を行い、異なる場合のみ出力する
func compareJSONFiles(devPath string, prodPath string, relPath string) {
	devJSON, err := loadJSON(devPath)
	if err != nil {
		fmt.Printf("dirAのJSONファイルの読み込みエラー: %s\n", err)
		return
	}

	prodJSON, err := loadJSON(prodPath)
	if err != nil {
		fmt.Printf("dirBのJSONファイルの読み込みエラー: %s\n", err)
		return
	}

	// JSONファイルの比較を行い、異なる部分のみ表示
	if compareJSONContent(devJSON, prodJSON, relPath) {
		fmt.Printf("上記のファイルに違いが見つかりました: %s\n\n", relPath)
	}
}

// 再帰的にディレクトリ内のファイルを探索し、JSONファイルを比較する関数
func compareDirectories(devDir, prodDir string) {
	err := filepath.Walk(devDir, func(devPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// JSONファイルのみを対象とする
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			// dirB側の対応するファイルパスを構成
			relPath, err := filepath.Rel(devDir, devPath)
			if err != nil {
				return err
			}
			prodPath := filepath.Join(prodDir, relPath)

			// dirB側のファイルが存在するか確認
			if _, err := os.Stat(prodPath); os.IsNotExist(err) {
				fmt.Printf("dirBにファイルが存在しません: %s\n\n", prodPath)
				return nil
			}

			// JSONファイルを比較
			compareJSONFiles(devPath, prodPath, relPath)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("dirAディレクトリの走査中のエラー: %s\n", err)
	}
}

func main() {
	// コマンドライン引数としてディレクトリパスを受け取る
	dirA := flag.String("dirA", "", "最初のディレクトリ (dirA) のパス")
	dirB := flag.String("dirB", "", "二つ目のディレクトリ (dirB) のパス")

	// 引数の解析
	flag.Parse()

	// 引数が指定されていない場合はエラーメッセージを表示して終了
	if *dirA == "" {
		fmt.Println("エラー: 最初のディレクトリパス (dirA) が必要です。--dirA でパスを指定してください。")
		flag.Usage() // 使用方法の表示
		os.Exit(1)
	}

	if *dirB == "" {
		fmt.Println("エラー: 二つ目のディレクトリパス (dirB) が必要です。--dirB でパスを指定してください。")
		flag.Usage() // 使用方法の表示
		os.Exit(1)
	}

	// ディレクトリが存在するかチェック
	if _, err := os.Stat(*dirA); os.IsNotExist(err) {
		fmt.Printf("エラー: 最初のディレクトリ '%s' (dirA) が存在しません。\n", *dirA)
		os.Exit(1)
	}

	if _, err := os.Stat(*dirB); os.IsNotExist(err) {
		fmt.Printf("エラー: 二つ目のディレクトリ '%s' (dirB) が存在しません。\n", *dirB)
		os.Exit(1)
	}

	// ディレクトリを再帰的に走査し、JSONファイルを比較
	compareDirectories(*dirA, *dirB)
}
