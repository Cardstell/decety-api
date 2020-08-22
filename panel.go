package main

import (
	"fmt"
	"log"
	"time"
	"net/http"
	"strings"
	"strconv"
	"math/rand"
	"bytes"
	"encoding/json"
	"github.com/satori/go.uuid"
	"github.com/gorilla/mux"
	"os"
	"io"
)

var (
	templates = map[string]string{}
	static = map[string]string{}
)

func addUUID(id string) error {
	stmt, err := db.Prepare("insert into admin_uuids (uuid) values (?)")
	if err != nil {
		return fmt.Errorf("Error creating stmt: %v\n", err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(id)
	if err != nil {
		return fmt.Errorf("Error request execution: %v\n", err)
	}
	return nil
}

func checkUUID(id string) (bool, error) {
	stmt, err := db.Prepare("select * from admin_uuids where uuid == ?")
	if err != nil {
		return false, fmt.Errorf("Error creating stmt: %v\n", err)
	}
	rows, err := stmt.Query(id)
	if err != nil {
		return false, fmt.Errorf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	return rows.Next(), nil
}

func redirectAuthorized(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("uuid")
	result := false
	if err == nil {
		result, err = checkUUID(cookie.Value)
		if err != nil {
			log.Print(err)
			http.Error(w, "500 internal server error", 500)
			return true
		}
	}

	if result {
		// redirect to main admin panel page
		http.Redirect(w, r, prefix + "/dc-admin-p/tokens", 301)
		return true
	}
	return false
}

func redirectUnauthorized(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("uuid")
	result := false
	if err == nil {
		result, err = checkUUID(cookie.Value)
		if err != nil {
			log.Print(err)
			http.Error(w, "500 internal server error", 500)
			return true
		}
	}

	if !result {
		// redirect to login page
		http.Redirect(w, r, prefix + "/dc-admin-p/", 301)
		return true
	}
	return false
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if (r.Method == http.MethodPost) {
		// login
		login := r.FormValue("login")
		password := r.FormValue("password")

		if login != defaultLogin || password != defaultPassword {
			fmt.Fprintf(w, "incorrect login or password")
			return
		}

		_id, _ := uuid.NewV4()
		id := _id.String()
		addUUID(id)

		http.SetCookie(w, &http.Cookie{
			Name: "uuid", 
			Value: id, 
			MaxAge: 0, 
		})

		fmt.Fprint(w, "ok")
	} else {
		if redirectAuthorized(w, r) {
			return
		}

		// login page
		fmt.Fprint(w, templates["login"])
	}	
}

func getRandomToken() string {
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 12
	result := make([]byte, length)

	for i := 0;i < length;i++ {
		result[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(result)
}

func getRandomShopID() string {
	return strconv.Itoa(rand.Intn(10000))
}

func isTokenExists(token string) bool {
	stmt, err := db.Prepare("select * from tokens where token == ?")
	if err != nil {
		log.Fatalf("Error creating stmt: %v\n", err)
	}
	rows, err := stmt.Query(token)
	if err != nil {
		log.Fatalf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	return rows.Next()
}

func isShopIDExists(shop_id string) bool {
	stmt, err := db.Prepare("select * from tokens where shop_id == ?")
	if err != nil {
		log.Fatalf("Error creating stmt: %v\n", err)
	}
	rows, err := stmt.Query(shop_id)
	if err != nil {
		log.Fatalf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	return rows.Next()
}

func getRandomValidToken() string {
	for {
		token := getRandomToken()
		if !isTokenExists(token) {
			return token
		}
	}
}

func getRandomValidShopID() string {
	for {
		shop_id := getRandomShopID()
		if !isShopIDExists(shop_id) {
			return shop_id
		}
	}
}

func getImagesCount(token string) string {
	stmt, err := db.Prepare("select count(token) from images where token == ?")
	if err != nil {
		log.Printf("Error creating stmt: %v\n", err)
		return "null"
	}
	rows, err := stmt.Query(token)
	if err != nil {
		log.Printf("Error query execution: %v\n", err)
		return "null"
	}
	defer rows.Close()
	rows.Next()

	result := ""
	rows.Scan(&result)
	return result
}

func getItemsCount(token string) string {
	stmt, err := db.Prepare("select count(*) from (select distinct item_id, color, size, description from items where token == ?)")
	if err != nil {
		log.Printf("Error creating stmt: %v\n", err)
		return "null"
	}
	rows, err := stmt.Query(token)
	if err != nil {
		log.Printf("Error query execution: %v\n", err)
		return "null"
	}
	defer rows.Close()
	rows.Next()

	result := ""
	rows.Scan(&result)
	return result
}

func tokensHandler(w http.ResponseWriter, r *http.Request) {
	if redirectUnauthorized(w, r) {
		return
	}

	if (r.Method == http.MethodPost) {
		req_v := r.FormValue("v")

		if req_v == "create" {
			token := r.FormValue("token")
			shop_id := r.FormValue("shop_id")
			description := r.FormValue("description")
			exp_time := r.FormValue("exp_time")

			if token == "" || shop_id == "" || isTokenExists(token) || isShopIDExists(shop_id) {
				fmt.Fprint(w, "invalid_request")
				return	
			}

			expiration_time, err := strconv.ParseInt(exp_time, 10, 64)
			if err != nil {
				fmt.Fprint(w, "invalid_request")
				return
			}

			stmt, err := db.Prepare("insert into tokens (token, exp_time, description, shop_id) values (?, ?, ?, ?)")
			if err != nil {
				log.Printf("Error creating stmt: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}
			defer stmt.Close()

			_, err = stmt.Exec(token, strconv.FormatInt(expiration_time, 10), description, shop_id)
			if err != nil {
				log.Printf("Error request execution: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}

			fmt.Fprint(w, "ok")
			return

		} else if req_v == "edit" {
			token := r.FormValue("token")
			shop_id := r.FormValue("shop_id")
			description := r.FormValue("description")
			exp_time := r.FormValue("exp_time")

			if token == "" || shop_id == "" || !isTokenExists(token) {
				fmt.Fprint(w, "invalid_request")
				return	
			}

			if isShopIDExists(shop_id) {
				token_shop_id, err := getShopID(token)
				if err != nil || token_shop_id != shop_id {
					fmt.Fprint(w, "invalid_request")
					return
				}
			}

			expiration_time, err := strconv.ParseInt(exp_time, 10, 64)
			if err != nil {
				fmt.Fprint(w, "invalid_request")
				return
			}

			stmt, err := db.Prepare("update tokens set exp_time = ?, description = ?, shop_id = ? where token = ?")
			if err != nil {
				log.Printf("Error creating stmt: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}
			defer stmt.Close()
			
			_, err = stmt.Exec(strconv.FormatInt(expiration_time, 10), description, shop_id, token)
			if err != nil {
				log.Printf("Error request execution: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}

			stmt2, err := db.Prepare("update items set shop_id = ? where token = ?")
			if err != nil {
				log.Printf("Error creating stmt: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}
			defer stmt2.Close()
			
			_, err = stmt2.Exec(shop_id, token)
			if err != nil {
				log.Printf("Error request execution: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}

			fmt.Fprint(w, "ok")
			return

		} else if req_v == "delete" {
			token := r.FormValue("token")
			stmt, err := db.Prepare("delete from images where token = ?")
			if err != nil {
				log.Printf("Error creating stmt: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}
			defer stmt.Close()
			
			_, err = stmt.Exec(token)
			if err != nil {
				log.Printf("Error request execution: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}

			stmt2, err := db.Prepare("delete from items where token = ?")
			if err != nil {
				log.Printf("Error creating stmt: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}
			defer stmt2.Close()
			
			_, err = stmt2.Exec(token)
			if err != nil {
				log.Printf("Error request execution: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}

			stmt3, err := db.Prepare("delete from tokens where token = ?")
			if err != nil {
				log.Printf("Error creating stmt: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}
			defer stmt3.Close()
			
			_, err = stmt3.Exec(token)
			if err != nil {
				log.Printf("Error request execution: %v\n", err)
				http.Error(w, "500 internal server error", 500)
				return
			}

			fmt.Fprint(w, "ok")
			return

		} else {
			fmt.Fprint(w, "invalid_request")
		}

		return
	}

	// Create html token list 

	html := templates["tokens"]
	html = strings.ReplaceAll(html, "{{token}}", getRandomValidToken())
	html = strings.ReplaceAll(html, "{{shop_id}}", getRandomValidShopID())

	token_blocks := ""

	rows, err := db.Query("select * from tokens")
	if err != nil {
		log.Printf("Error query execution: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	
	num := 0
	for rows.Next() {
		var token, description, shop_id string
		var expTime int64
		err := rows.Scan(&token, &expTime, &description, &shop_id)
		if err != nil {
			log.Print(err)
			http.Error(w, "500 internal server error", 500)
			return
		}

		token_block := templates["token-block"]
		token_block = strings.ReplaceAll(token_block, "{{token}}", token)
		token_block = strings.ReplaceAll(token_block, "{{shop_id}}", shop_id)
		token_block = strings.ReplaceAll(token_block, "{{num}}", strconv.Itoa(num))

		if description == "" {
			token_block = strings.ReplaceAll(token_block, "{{br}}", "")	
		} else {
			token_block = strings.ReplaceAll(token_block, "{{br}}", "<br/>")
		}
		token_block = strings.ReplaceAll(token_block, "{{description}}", description)
		
		token_block = strings.ReplaceAll(token_block, "{{images_count}}", getImagesCount(token))
		token_block = strings.ReplaceAll(token_block, "{{items_count}}", getItemsCount(token))

		expired := expTime <= time.Now().Unix()
		time_string := time.Unix(expTime, 0).UTC().Format("2006-01-02 15:04:05 UTC")
		time_string_default := time.Unix(expTime, 0).UTC().Format("2006-01-02T15:04:05")
		token_block = strings.ReplaceAll(token_block, "{{exp_time_default}}", time_string_default)

		if expired {
			token_block = strings.ReplaceAll(token_block, "{{exp_time}}",
				"<span class=\"text-danger\">Expired:</span> " + time_string)
		} else {
			token_block = strings.ReplaceAll(token_block, "{{exp_time}}", 
				"Valid through: " + time_string)
		}

		token_blocks += token_block
		num++
	}

	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	html = strings.ReplaceAll(html, "{{container}}", token_blocks)
	fmt.Fprint(w, html)
}

type jsonItem struct {
	Item_id string 			`json:"item_id"`
	Color string 			`json:"color"`
	Size string 			`json:"size"`
	Description string 		`json:"description"`
	Requests_count int 		`json:"requests_count"`
	Items []jsonTypeItem 	`json:"items"`
}

type jsonTypeItem struct {
	type_ string
	params []float64
	requests_count int
	image_list string
}

type keyItem struct {
	Item_id string
	Color string
	Size string
	Description string
}

func (item jsonTypeItem) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	jsonValue, err := json.Marshal(item.type_)
	if err != nil {
		return nil, err
	}
	buffer.WriteString("\"type\":" + string(jsonValue) + ",")

	for i, name := range paramNames {
		jsonValue, err := json.Marshal(item.params[i])
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("\"%s\":%s,", name, string(jsonValue)))
	}

	buffer.WriteString(fmt.Sprintf("\"requests_count\":%d,\"image_list\":[", item.requests_count))
	ids := strings.Split(item.image_list, ",")
	for i, id := range ids {
		jsonValue, err := json.Marshal(id)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(string(jsonValue))
		if i != len(ids) - 1 {
			buffer.WriteString(",")
		}
	}

	buffer.WriteString("]}")
	return buffer.Bytes(), nil
}

func itemsHandler(w http.ResponseWriter, r *http.Request) {
	if redirectUnauthorized(w, r) {
		return
	}

	token := r.FormValue("token")

	stmt, err := db.Prepare(fmt.Sprintf("select item_id, color, size, description, type, image_list, requests_count, %v from items where token == ?", 
		strings.Join(paramNames, ", ")))
	if err != nil {
		log.Printf("Error creating stmt: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(token)
	if err != nil {
		log.Printf("Error query execution: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	defer rows.Close()

	items := make(map[keyItem][]jsonTypeItem)

	for rows.Next() {
		var item_id, color, size, description, type_, image_list string
		var requests_count int
		dest := make([]interface{}, 7 + len(paramNames))
		dest[0] = &item_id
		dest[1] = &color
		dest[2] = &size
		dest[3] = &description
		dest[4] = &type_
		dest[5] = &image_list
		dest[6] = &requests_count
		for i := range paramNames {
			dest[i+7] = new(float64)
		}

		if err = rows.Scan(dest...); err != nil {
			log.Print(err)
			http.Error(w, "500 internal server error", 500)
			return
		}

		params := make([]float64, len(paramNames))
		for i := range paramNames {
			params[i] = *(dest[i+7].(*float64))		
		}

		items[keyItem{item_id, color, size, description}] = append(items[keyItem{item_id, color, size, description}], 
			jsonTypeItem{type_, params, requests_count, image_list})
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error scanning rows: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	result := []jsonItem{}
	for key, typeItems := range items {
		requests_count := 0
		for _, item := range typeItems {
			requests_count += item.requests_count
		}
		result = append(result, jsonItem{key.Item_id, key.Color, key.Size, key.Description, requests_count, typeItems})
	}

	json_result, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error json serializing: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	fmt.Fprint(w, string(json_result))
}

func isLoginStatic(name string) bool {
	return name == "login.css" || name == "login.js"
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if !isLoginStatic(name) {
		if redirectUnauthorized(w, r) {
			return
		}
	}

	file, ok := static[name]
	if !ok {
		http.Error(w, "404 file not found", 404)
		return
	}

	content_type := ""
	if strings.HasSuffix(name, ".css") {
		content_type = "text/css"
	} else if strings.HasSuffix(name, ".js") {
		content_type = "text/javascript"
	}

	w.Header().Set("Content-Type", content_type)
	fmt.Fprint(w, file)
}

func imagePanelHandler(w http.ResponseWriter, r *http.Request) {
	if redirectUnauthorized(w, r) {
		return
	}

	id := mux.Vars(r)["id"]
	file, err := os.Open("./images/" + id + ".jpg")
	if err != nil {
		http.Error(w, "404 file not found", 404)
		return
	}
	defer file.Close()

	fileStat, _ := file.Stat()
	fileSize := strconv.FormatInt(fileStat.Size(), 10)
	w.Header().Set("Content-Type", "image/jpg")
	w.Header().Set("Content-Length", fileSize)
	io.Copy(w, file)
}

func previewHandler(w http.ResponseWriter, r *http.Request) {
	if redirectUnauthorized(w, r) {
		return
	}

	id := mux.Vars(r)["id"]
	file, err := os.Open("./images/previews/" + id + ".jpg")
	if err != nil {
		http.Error(w, "404 file not found", 404)
		return
	}
	defer file.Close()

	fileStat, _ := file.Stat()
	fileSize := strconv.FormatInt(fileStat.Size(), 10)
	w.Header().Set("Content-Type", "image/jpg")
	w.Header().Set("Content-Length", fileSize)
	io.Copy(w, file)
}