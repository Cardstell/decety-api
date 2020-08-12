package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
	"github.com/boltdb/bolt"
	"encoding/json"
	"math/rand"
	"time"
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
	db *bolt.DB
	server *http.Server
)

type tokensItem struct {
	Image_ids, Ids []string
	Description, ShopID string
}

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
	result := false
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokenExpirationTime"))
		expTimeString := bucket.Get([]byte(token))
		if expTimeString == nil {
			return nil
		}
		expTime, err := strconv.ParseInt(string(expTimeString), 10, 64)
		if err != nil {
			return err
		}

		result = expTime > time.Now().Unix()
		return nil
	})
	return result, err
}

func addImageIDtoDB(token, image_id string) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		json_value := bucket.Get([]byte(token))
		var value tokensItem
		err := json.Unmarshal(json_value, &value)
		if err != nil {
			return err
		}

		value.Image_ids = append(value.Image_ids, image_id)
		json_value, err = json.Marshal(value)
		if err != nil {
			return err
		}
		bucket.Put([]byte(token), json_value)

		bucket = tx.Bucket([]byte("imageIDToken"))
		bucket.Put([]byte(image_id), []byte(token))
		return nil
	})
}

func isImageIDExists(image_id string) bool {
	var result bool
	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("imageIDToken"))
		token := bucket.Get([]byte(image_id))
		result = token != nil
		return nil
	})
	return result
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
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

	if !limiter.Allow() {
		printError(w, "flood_limit")
		return
	}

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

	err = addImageIDtoDB(token, image_id)
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	printResult(w, "\"" + image_id + "\"")
}

func isIDExists(id string) bool {
	var result bool
	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("IDToken"))
		token := bucket.Get([]byte(id))
		result = token != nil
		return nil
	})
	return result
}

func isValidImageIDs(image_ids string) (bool, error) {
	ids := strings.Split(image_ids, ",")
	if len(ids) > maxImagesPerID || image_ids == "" {
		return false, nil
	}

	result := true
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("imageIDToken"))
		for _, id := range ids {
			token := bucket.Get([]byte(id))
			if token == nil {
				result = false
				return nil
			}
		}
		return nil			
	})

	if err != nil {
		return result, err
	}
	return result, nil
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
	var result string
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		json_value := bucket.Get([]byte(token))
		var value tokensItem
		err := json.Unmarshal(json_value, &value)
		if err != nil {
			return err
		}
		
		result = value.ShopID
		return nil
	})
	return result, err
}

func packID(shop_id, id, color, size string) string {
	return shop_id + ";" + id + ";" + color + ";" + size
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	color := r.FormValue("color")
	size := r.FormValue("size")
	image_ids := r.FormValue("image_ids")
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

	shop_id, err := getShopID(token)
	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	id = packID(shop_id, id, color, size)

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

	if isIDExists(id) {
		printError(w, "invalid_request")
		return
	}
	if !limiter.Allow() {
		printError(w, "flood_limit")
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("IDImageIDs"))
		bucket.Put([]byte(id), []byte(imageIDsToJSON(image_ids)))

		bucket = tx.Bucket([]byte("IDToken"))
		bucket.Put([]byte(id), []byte(token))

		bucket = tx.Bucket([]byte("tokens"))
		json_value := bucket.Get([]byte(token))
		var value tokensItem
		err := json.Unmarshal(json_value, &value)
		if err != nil {
			return err
		}

		value.Ids = append(value.Ids, id)
		json_value, err = json.Marshal(value)
		if err != nil {
			return err
		}
		bucket.Put([]byte(token), json_value)
		return nil
	})

	if err != nil {
		log.Print(err)
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
	id = packID(shop_id, id, color, size)

	valid := true
	var result string
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("IDToken"))
		token := bucket.Get([]byte(id))
		if token == nil {
			valid = false
			return nil
		}

		bucket = tx.Bucket([]byte("tokenExpirationTime"))
		expTime, err := strconv.ParseInt(string(bucket.Get([]byte(token))), 10, 64)
		if err != nil {
			 return err
		}
		if expTime <= time.Now().Unix() {
			valid = false
			return nil
		}

		bucket = tx.Bucket([]byte("IDImageIDs"))
		result = string(bucket.Get([]byte(id)))
		return nil
	})

	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}
	if !valid {
		printError(w, "invalid_id")
		return
	}

	printResult(w, result)
}

func isValidImageID(id string) (bool, error) {
	result := false
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("imageIDToken"))
		token := bucket.Get([]byte(id))
		if token == nil {
			return nil
		}

		bucket = tx.Bucket([]byte("tokenExpirationTime"))
		expTime, err := strconv.ParseInt(string(bucket.Get([]byte(token))), 10, 64)
		if err != nil {
			 return err
		}

		result = expTime > time.Now().Unix()
		return nil
	})

	return result, err
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	valid, err := isValidImageID(id)
	if err != nil {
		log.Print(err)
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

// func createToken(token string, duration time.Duration) {
// 	expTime := time.Now().Add(duration).Unix()
// 	err := db.Update(func(tx *bolt.Tx) error {
// 		bucket := tx.Bucket([]byte("tokenExpirationTime"))
// 		bucket.Put([]byte(token), []byte(strconv.FormatInt(expTime, 10)))

// 		bucket = tx.Bucket([]byte("tokens"))
// 		value := tokensItem{}
// 		json_value, err := json.Marshal(value)
// 		if err != nil {
// 			return err
// 		}
// 		bucket.Put([]byte(token), []byte(json_value))
// 		return nil
// 	})

// 	if err != nil {
// 		db.Close()
// 		log.Fatal(err)
// 	}
// }

func createNewDatabase() {
	err := db.Update(func(tx *bolt.Tx) error {
		var result []string
		err := tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			result = append(result, string(name))
			return nil
		})
		if err != nil {
			 return err
		}

		for _, bucket := range result {
			err := tx.DeleteBucket([]byte(bucket))
			if err != nil {
				return err
			}
		}

		buckets := []string{"IDImageIDs", "IDToken", "imageIDToken", "tokenExpirationTime", "tokens", "adminUUIDs", "shopIDToken"}
		for _, bucket := range buckets {
			_, err := tx.CreateBucket([]byte(bucket))
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		db.Close()
		log.Fatal(err)
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	var err error
	db, err = bolt.Open("database.db", 0600, nil)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()

	
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

	// for i := 0;i<=24;i++ {
	// 	createToken(strconv.Itoa(i) + "m", time.Minute * time.Duration(i))
	// }
	
	r := mux.NewRouter()
	r.HandleFunc(prefix + "/upload", uploadHandler).Methods("POST")
	r.HandleFunc(prefix + "/update", updateHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/get", getHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/image/{id}", imageHandler).Methods("GET")
	r.HandleFunc(prefix + "/dc-admin-p/", loginHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/dc-admin-p/tokens", tokensHandler).Methods("GET", "POST")
	r.HandleFunc(prefix + "/dc-admin-p/static/{name}", staticHandler).Methods("GET")
	server = &http.Server{
		Handler: r,
		Addr: ":" + port,
	}
	server.ListenAndServe()
}