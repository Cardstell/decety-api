package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
	"math/rand"
	"time"
	"github.com/disintegration/imaging"
	"image"
	"image/jpeg"
	"path/filepath"
	"os"
	"io"
	"io/ioutil"
	"mime/multipart"
	"strconv"
	"strings"
)

var (
	templateNames = []string{"login", "tokens", "token-block"}
	staticNames = []string{"login.css", "login.js", "tokens.css", "tokens.js"}
	limiter = rate.NewLimiter(1, 1000)
	db *sql.DB
	server *http.Server
)

func getRandomID() string {
	const N = 12
	s := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, N)
	for i := range result {
		result[i] = s[rand.Intn(len(s))]
	}
	return string(result)
}

func printError(w http.ResponseWriter, error string) {
	fmt.Fprintf(w, "{\"error\":\"" + error + "\"}")
}

func printResult(w http.ResponseWriter, result string) {
	fmt.Fprintf(w, "{\"error\":\"\",\"result\":" + result + "}")
}

func getRequestFile(r *http.Request) (multipart.File, bool) {
	for _, field := range []string{"file", "data", "image"} {
		reqfile, _, err := r.FormFile(field)
		if err == nil {
			return reqfile, true
		}
	}
	return multipart.File(nil), false
}

func isValidToken(token string) (bool, error) {
	stmt, err := db.Prepare("select exp_time from tokens where token == ?")
	if err != nil {
		return false, fmt.Errorf("Error creating stmt: %v\n", err)
	}
	rows, err := stmt.Query(token)
	if err != nil {
		return false, fmt.Errorf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return false, nil
	}

	var exp_time int64
	if err = rows.Scan(&exp_time); err != nil {
		return false, err
	}
	if err = rows.Err(); err != nil {
		return false, err
	}

	return exp_time > time.Now().Unix(), nil
}

func addImageIDtoDB(token, image_id string) error {
	stmt, err := db.Prepare("insert into images (token, image_id) values (?, ?)")
	if err != nil {
		return fmt.Errorf("Error creating stmt: %v\n", err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(token, image_id)
	if err != nil {
		return fmt.Errorf("Error request execution: %v\n", err)
	}
	return nil
}

func isImageIDExists(image_id string) bool {
	stmt, err := db.Prepare("select * from images where image_id == ?")
	if err != nil {
		log.Fatalf("Error creating stmt: %v\n", err)
	}
	rows, err := stmt.Query(image_id)
	if err != nil {
		log.Fatalf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	return rows.Next()
}

func generatePreview(file multipart.File, image_id string) error {
	fullImage, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	previewImage := imaging.Thumbnail(fullImage, 80, 80, imaging.CatmullRom)
	newImage, err := os.Create("images/previews/" + image_id + ".jpg")
	if err != nil {
		return err
	}
	defer newImage.Close()

	if jpeg.Encode(newImage, previewImage, &jpeg.Options{jpeg.DefaultQuality}) != nil {
		return err
	}
	return nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if !limiter.Allow() {
		printError(w, "flood_limit")
		return
	}

	r.ParseMultipartForm(1 << 23)
	token := r.FormValue("token")
	valid, err := isValidToken(token)
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	if !valid {
		printError(w, "invalid_token")
		return
	}

	reqfile, ok := getRequestFile(r)
	if !ok {
		printError(w, "invalid_request")
		return
	}
	defer reqfile.Close()

	image_id := getRandomID()
	for ;isImageIDExists(image_id); {
		image_id = getRandomID()
	}
	file, err := os.Create("images/" + image_id + ".jpg")
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	io.Copy(file, reqfile)
	file.Close()

	if _, err := reqfile.Seek(0, 0); err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	if err = generatePreview(reqfile, image_id); err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	err = addImageIDtoDB(token, image_id)
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	printResult(w, "\"" + image_id + "\"")
}

func isValidImageIDs(image_ids string) (bool, error) {
	ids := strings.Split(image_ids, ",")
	if len(ids) > maxImagesPerID || image_ids == "" {
		return false, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("Error creating database transaction: %v\n", err)
	}
	defer tx.Commit()

	stmt, err := tx.Prepare("select image_id from images where image_id == ?")
	if err != nil {
		return false, fmt.Errorf("Error creating stmt: %v\n", err)
	}
	defer stmt.Close()
	for _, image_id := range ids {
		rows, err := stmt.Query(image_id)
		if err != nil {
			return false, fmt.Errorf("Error query execution: %v\n", err)
		}
		if !rows.Next() {
			rows.Close()
			return false, nil
		}	
		rows.Close()
	}
	return true, nil
}

func imageIDsToJSON(image_ids string) string {
	result := ""
	ids := strings.Split(image_ids, ",")
	for _, id := range ids {
		result += "\"" + id + "\","
	}
	return "[" + result[:len(result)-1] + "]"
}

func getShopID(token string) (string, error) {
	stmt, err := db.Prepare("select shop_id from tokens where token == ?")
	if err != nil {
		return "", fmt.Errorf("Error creating stmt: %v\n", err)
	}
	rows, err := stmt.Query(token)
	if err != nil {
		return "", fmt.Errorf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return "", fmt.Errorf("Token %v\n not exists", token)
	}

	var shop_id string
	if rows.Scan(&shop_id) != nil {
		return "", err
	}
	if rows.Err() != nil {
		return "", err
	}
	return shop_id, nil
}

func isItemExists(id, color, size, type_ string) bool {
	stmt, err := db.Prepare("select * from items where item_id == ? AND color == ? AND size == ? AND type == ?")
	if err != nil {
		log.Fatalf("Error creating stmt: %v\n", err)
	}
	rows, err := stmt.Query(id, color, size, type_)
	if err != nil {
		log.Fatalf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	return rows.Next()
} 

func getL2Norm(a, b []float64) float64 {
	result := 0.0
	for i := range a {
		// Z-scaling
		dt := (a[i] - b[i]) * paramWeights[i]
		result += dt * dt
	}
	return result
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	color := r.FormValue("color")
	size := r.FormValue("size")
	type_ := r.FormValue("type")
	image_ids := r.FormValue("image_ids")
	token := r.FormValue("token")
	params := make([]string, len(paramNames))
	for i, name := range paramNames {
		floatValue, err := strconv.ParseFloat(r.FormValue(name), 10)
		if err != nil || r.FormValue(name) == "" {
			printError(w, "invalid_request")
			return
		}
		params[i] = fmt.Sprintf("'%f'", floatValue)
	}

	if !limiter.Allow() {
		printError(w, "flood_limit")
		return
	}
	valid, err := isValidToken(token)
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	if !valid {
		printError(w, "invalid_token")
		return
	}

	shop_id, err := getShopID(token)
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	valid, err = isValidImageIDs(image_ids)
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	if !valid {
		printError(w, "invalid_request")
		return
	}

	if isItemExists(id, color, size, type_) || id == "" {
		printError(w, "invalid_id")
		return
	}

	stmt, err := db.Prepare(fmt.Sprintf(`insert into items (token, shop_id, item_id, color, size, type, image_list, %s) 
		values (?, ?, ?, ?, ?, ?, ?, %s)`, 
		strings.Join(paramNames[:], ", "), 
		strings.Join(params, ", ")))

	if err != nil {
		log.Printf("Error creating stmt: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(token, shop_id, id, color, size, type_, image_ids)
	if err != nil {
		log.Printf("Error request execution: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	printResult(w, "\"\"")
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	shop_id := r.FormValue("shop_id")
	id := r.FormValue("id")
	color := r.FormValue("color")
	size := r.FormValue("size")
	params := make([]float64, len(paramNames))
	for i, name := range paramNames {
		var err error
		params[i], err = strconv.ParseFloat(r.FormValue(name), 10)
		if err != nil || r.FormValue(name) == "" {
			printError(w, "invalid_request")
			return
		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error creating database transaction: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	defer tx.Commit()

	stmt, err := tx.Prepare(fmt.Sprintf(`select token, image_list, type, %v from items where 
		shop_id == ? AND item_id == ? AND color == ? AND size == ?`, strings.Join(paramNames, ", ")))
	if err != nil {
		log.Printf("Error creating stmt: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query(shop_id, id, color, size)
	if err != nil {
		log.Printf("Error query execution: %v\n", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	defer rows.Close()

	success := false
	var bestType, bestParams, resultImageList string
	var minL2Norm float64

	for rows.Next() {
		var token, image_list, type_ string
		dest := make([]interface{}, 3 + len(paramNames))
		dest[0] = &token
		dest[1] = &image_list
		dest[2] = &type_
		for i := range paramNames {
			dest[i+3] = new(float64)
		}

		if err = rows.Scan(dest...); err != nil {
			log.Print(err)
			http.Error(w, "500 internal server error", 500)
			return
		}

		currentParams := make([]float64, len(paramNames))
		for i := range paramNames {
			currentParams[i] = *(dest[i+3].(*float64))		
		}

		valid, err := isValidToken(token)
		if err != nil {
			log.Print(err)
			http.Error(w, "500 internal server error", 500)
			return
		}
		if !valid {
			continue
		}

		l2norm := getL2Norm(params, currentParams)
		if !success {
			success = true
			bestType = type_
			bestParams = strings.ReplaceAll(fmt.Sprint(currentParams), " ", ",")
			resultImageList = image_list
			minL2Norm = l2norm
		} else if l2norm < minL2Norm {
			bestType = type_
			bestParams = strings.ReplaceAll(fmt.Sprint(currentParams), " ", ",")
			resultImageList = image_list
			minL2Norm = l2norm
		}

	}
	if rows.Err() != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	if success {
		fmt.Fprintf(w, `{"error":"","result":"[%v]","type":"%v","params":%v}`, resultImageList, bestType, bestParams)
	} else {
		printError(w, "invalid_id")
	}
}

func isValidImageID(id string) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("Error creating database transaction: %v\n", err)
	}
	defer tx.Commit()

	stmt, err := tx.Prepare("select token from images where image_id == ?")
	if err != nil {
		return false, fmt.Errorf("Error creating stmt: %v\n", err)
	}
	defer stmt.Close()
	rows, err := stmt.Query(id)
	if err != nil {
		return false, fmt.Errorf("Error query execution: %v\n", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return false, nil
	}

	var token string
	if err = rows.Scan(&token); err != nil {
		return false, err
	}
	if err = rows.Err(); err != nil {
		return false, err
	}

	stmt2, err := tx.Prepare("select exp_time from tokens where token == ?")
	if err != nil {
		return false, fmt.Errorf("Error creating stmt: %v\n", err)
	}
	defer stmt2.Close()
	rows2, err := stmt2.Query(token)
	if err != nil {
		return false, fmt.Errorf("Error query execution: %v\n", err)
	}
	defer rows2.Close()
	if !rows2.Next() {
		return false, nil
	}

	var exp_time int64
	if err = rows2.Scan(&exp_time); err != nil {
		return false, err
	}
	if err = rows2.Err(); err != nil {
		return false, err
	}

	return exp_time > time.Now().Unix(), nil
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	valid, err := isValidImageID(id)
	if err != nil {
		log.Print("Database error:", err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	if !valid {
		http.Error(w, "404 file not found", 404)
		return
	}
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

func createTablesIfNotExists() {
	sqlStmt := fmt.Sprintf(`
	create table if not exists images (
		id integer not null primary key autoincrement, 
		token text not null, 
		image_id text not null
	);

	create table if not exists items (
		id integer not null primary key autoincrement, 
		token text not null,
		shop_id text not null,
		item_id text not null, 
		color text,
		size text,
		type integer not null,
		%s,
		image_list text
	);

	create table if not exists tokens (
		token text not null primary key,
		exp_time time not null,
		description text,
		shop_id text
	);

	create table if not exists admin_uuids (
		uuid text not null primary key
	);

	`, strings.Join(paramNames, " float,\n		") + " float")

	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatal("Error creating tables:", err)
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	os.MkdirAll(filepath.Join(".", "images/previews"), os.ModePerm)

	var err error
	db, err = sql.Open("sqlite3", "sqlite3.db")
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()
	createTablesIfNotExists()

	for _, name := range templateNames {
		file, err := os.Open("templates/" + name + ".html")
		if err != nil {
			log.Fatal("Error opening template: ", name)
		}

		template, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal("Error reading template: ", name)
		}
		templates[name] = string(template)
		file.Close()
	}

	for _, name := range staticNames {
		file, err := os.Open("static/" + name)
		if err != nil {
			log.Fatal("Error opening file: ", name)
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal("Error reading file: ", name)
		}
		static[name] = string(content)
		file.Close()
	}
	
	r := mux.NewRouter()
	r.HandleFunc(prefix + "/upload", uploadHandler).Methods("POST")
	r.HandleFunc(prefix + "/update", updateHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/get", getHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/image/{id}", imageHandler).Methods("GET")
	r.HandleFunc(prefix + "/dc-admin-p/", loginHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/dc-admin-p/tokens", tokensHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/dc-admin-p/items", itemsHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/dc-admin-p/static/{name}", staticHandler).Methods("GET")
	r.HandleFunc(prefix + "/dc-admin-p/image/{id}", imagePanelHandler).Methods("GET")
	r.HandleFunc(prefix + "/dc-admin-p/preview/{id}", previewHandler).Methods("GET")
	server = &http.Server{
		Handler: r,
		Addr: ":" + port,
	}
	server.ListenAndServe()
}