package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sqweek/dialog"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func main() {

	var (
		// 選擇要轉的新注音的詞彙文字檔
		inputPath string
		// 輸入 sqlite3 檔名
		outputPath string
	)

	// 使用者輸入
	{
		var err error
		// 選擇要轉的新注音的詞彙文字檔
		inputPath, err = dialog.File().Title("選擇檔案").Load()
		printError(err)

		// 輸入 sqlite3 檔名
		outputPath, err = dialog.File().Title("儲存檔案").Save()
		printError(err)

		// 自動補上副檔名（如果使用者沒輸入）
		if !strings.HasSuffix(outputPath, ".sqlite3") {
			outputPath += ".sqlite3"
		}
	}

	//　主程式

	{
		// 讀取文字檔案 詞彙和注音符號部份
		lines := ReadDatas(inputPath)

		var dbDatas [][]any
		for _, line := range lines {
			phrase, phone := line[0], line[1]
			// 注音符號 空白隔開
			phonesWord := strings.Split(phone, ` `)
			// 根據轉換表計算加總
			var phones []int
			for _, word := range phonesWord {
				var sum int
				for _, p := range word {
					phone := string(p)
					sum += phoneticToCode[phone]
				}
				phones = append(phones, sum)
			}

			// 注音符號長度
			length := len(phonesWord)

			db := dbData(phrase, length, phones)
			dbDatas = append(dbDatas, db)
		}

		fmt.Println("即將匯入筆數：", len(dbDatas))
		// 插入資料
		dbImport(dbDatas, outputPath)

		fmt.Println("已經匯出到", outputPath)
	}

}

// 讀取文字檔案
func ReadDatas(path string) [][]string {
	// 開啟檔案並轉為 UTF-8
	file, err := os.Open(path)
	printError(err)
	defer file.Close()

	// 建立 UTF-16 LE 解碼器
	utf16bom := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM)
	reader := transform.NewReader(file, utf16bom.NewDecoder())

	// 讀取並轉成 UTF-8 字串
	utf8Bytes, err := io.ReadAll(reader)
	printError(err)

	var contents [][]string
	for line := range strings.SplitSeq(string(utf8Bytes), "\n") {
		// 取代全形空白
		line = strings.ReplaceAll(line, `　`, ``)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 有詞彙和注音符號部份
		data := strings.Split(line, `  `)
		if len(data) < 2 {
			continue // 防止格式錯誤的行
		}
		// 詞彙
		phrase := data[0]
		// 注音符號
		phone := strings.Join(data[1:], ` `)

		contents = append(contents, []string{phrase, phone})
	}

	return contents
}

// 建立資料
func dbData(phrase string, length int, phones []int) []any {
	values := make([]any, 17)
	values[0] = 926    // time
	values[1] = 1      // user_freq
	values[2] = 1      // max_freq
	values[3] = 1      // orig_freq
	values[4] = length // 注音音節數

	for i := range 11 {
		if i < len(phones) {
			values[5+i] = phones[i]
		} else {
			values[5+i] = 0
		}
	}
	values[16] = phrase
	return values
}

// 插入資料
func dbImport(dbDatas [][]any, outputPath string) {

	// 先產生一個空白db檔案
	db, err := sql.Open("sqlite3", outputPath)
	printError(err)
	defer db.Close()

	createUserPhrase := `CREATE TABLE userphrase_v1 (time INTEGER,user_freq INTEGER,max_freq INTEGER,orig_freq INTEGER,length INTEGER,phone_0 INTEGER,phone_1 INTEGER,phone_2 INTEGER,phone_3 INTEGER,phone_4 INTEGER,phone_5 INTEGER,phone_6 INTEGER,phone_7 INTEGER,phone_8 INTEGER,phone_9 INTEGER,phone_10 INTEGER,phrase TEXT,PRIMARY KEY (phone_0,phone_1,phone_2,phone_3,phone_4,phone_5,phone_6,phone_7,phone_8,phone_9,phone_10,phrase));`
	createConfig := `CREATE TABLE config_v1 (id INTEGER,value INTEGER,PRIMARY KEY (id));`

	_, err = db.Exec(createUserPhrase)
	printError(err)
	_, err = db.Exec(createConfig)
	printError(err)

	// SQL 插入語句
	sqlStmt := `INSERT INTO userphrase_v1 (	time, user_freq, max_freq, orig_freq, length, phone_0, phone_1, phone_2, phone_3, phone_4, phone_5, phone_6, phone_7, phone_8, phone_9, phone_10, phrase) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	tx, err := db.Begin()
	printError(err)

	stmt, err := tx.Prepare(sqlStmt)
	printError(err)
	defer stmt.Close()

	for _, data := range dbDatas {
		_, err := stmt.Exec(data...)
		printError(err)
	}

	err = tx.Commit()
	printError(err)
}

var phoneticToCode = map[string]int{
	// 聲母
	"ㄅ": 512,
	"ㄆ": 1024,
	"ㄇ": 1536,
	"ㄈ": 2048,
	"ㄉ": 2560,
	"ㄊ": 3072,
	"ㄋ": 3584,
	"ㄌ": 4096,
	"ㄍ": 4608,
	"ㄎ": 5120,
	"ㄏ": 5632,
	"ㄐ": 6144,
	"ㄑ": 6656,
	"ㄒ": 7168,
	"ㄓ": 7680,
	"ㄔ": 8192,
	"ㄕ": 8704,
	"ㄖ": 9216,
	"ㄗ": 9728,
	"ㄘ": 10240,
	"ㄙ": 10752,

	// 介音
	"ㄧ": 128,
	"ㄨ": 256,
	"ㄩ": 384,

	// 韻母
	"ㄚ": 8,
	"ㄛ": 16,
	"ㄜ": 24,
	"ㄝ": 32,
	"ㄞ": 40,
	"ㄟ": 48,
	"ㄠ": 56,
	"ㄡ": 64,
	"ㄢ": 72,
	"ㄣ": 80,
	"ㄤ": 88,
	"ㄥ": 96,
	"ㄦ": 104,

	// 聲調
	"ˉ": 0, // 輕聲或無聲調
	"˙": 1,
	"ˊ": 2, // 第二聲
	"ˇ": 3, // 第三聲
	"ˋ": 4, // 第四聲
}

func printError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
