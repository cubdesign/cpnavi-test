# cpnavi-test


## フォルダ比較

Tasksを使う場合

```sh 
DIRA="export/development" DIRB="export/production" task hikaku > "export/hikaku.txt"
```

Tasksを使わない場合

```sh
go run cmd/cli/hikaku/Hikaku.go --dirA=export/development --dirB=export/production  > "export/hikaku.txt"
```



## APIからのデータ取得

	//label := "local"
	//label := "production"
	//label := "development"


LABEL: local, production, development
API: university, major 


Tasksを使う場合

```sh
LABEL="local" \
API="university" \
APIHOST="http://localhost:8081" \
CSVFILE="import/university_slug.csv" \
ACCESSTOKEN="eyJhbGdgWVNfHvwzcOF0XLQyG_uzjhDAwsx2eRCZskL1g55YIA" \
EXPORTFOLDER="export" \
task getjson
```

Tasksを使わない場合
```sh
go run cmd/cli/get-json/GetJson.go \
  --label=local \
  --api=university \
  --apiHost=http://localhost:8081 \
  --csv=import/university_slug.csv \
  --accessToken=eyJhbGdgWVNfHvwzcOF0XLQyG_uzjhDAwsx2eRCZskL1g55YIA \
  --exportFolder=export
```
