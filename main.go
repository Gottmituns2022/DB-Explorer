// тут лежит тестовый код
// менять вам может потребоваться только коннект к базе
package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
)

type ItemRecord struct {
	title       string
	description string
	updated     string
}

var (
	// DSN это соединение с базой
	// вы можете изменить этот на тот который вам нужен
	// docker run -p 3306:3306 -v $(PWD):/docker-entrypoint-initdb.d -e MYSQL_ROOT_PASSWORD=1234 -e MYSQL_DATABASE=golang -d mysql
	// DSN = "root@tcp(localhost:3306)/golang2017?charset=utf8"
	DSN = "root:Danetda8383@@tcp(localhost:3306)/maindata?charset=utf8"
)

func main() {
	db, err := sql.Open("mysql", DSN)
	err = db.Ping() // вот тут будет первое подключение к базе
	if err != nil {
		panic(err)
	}

	handler, err := NewDbExplorer(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("starting server at :8082")
	http.ListenAndServe(":8082", handler)
	//waitChan := make(chan int, 0)

	/*data := url.Values{}
	data.Set("title", "Ademar")
	data.Set("description", "Bishop of Baden")
	data.Set("updated", "Heinrich VIII")

	req, _ := http.NewRequest("PUT", "http://localhost:8082/items", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: time.Second}
	client.Do(req)

	dataPost := url.Values{}
	dataPost.Set("title", "Reinhard")
	dataPost.Set("description", "Bishop of Hamburg")
	dataPost.Set("updated", "Wilhelm II")

	req, _ := http.NewRequest("POST", "http://localhost:8082/items/10", strings.NewReader(dataPost.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client.Do(req)

	req, _ := http.NewRequest("DELETE", "http://localhost:8082/items/10", nil)
	client.Do(req)*/

	//waitChan <- 1
}
