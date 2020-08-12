package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"net/http"
	"strings"
	"strconv"
	"math/rand"
	"encoding/json"
	"github.com/boltdb/bolt"
	"github.com/satori/go.uuid"
	"github.com/gorilla/mux"
)

var (
	templates = map[string]string{}
	static = map[string]string{}
)

func addUUID(id string) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("adminUUIDs"))
		bucket.Put([]byte(id), []byte("a"))
		return nil
	})
}

func checkUUID(id string) (bool, error) {
	result := false
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("adminUUIDs"))
		result = bucket.Get([]byte(id)) != nil
		return nil
	})
	return result, err
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
	result := true
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokenExpirationTime"))
		result = bucket.Get([]byte(token)) != nil
		return nil
	})

	if err != nil {
		log.Print(err)
		result = true
	}
	return result
}

func isShopIDExists(shop_id string) bool {
	result := true
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("shopIDToken"))
		result = bucket.Get([]byte(shop_id)) != nil
		return nil
	})

	if err != nil {
		log.Print(err)
		result = true
	}
	return result
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

			err = db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte("tokenExpirationTime"))
				bucket.Put([]byte(token), []byte(strconv.FormatInt(expiration_time, 10)))

				bucket = tx.Bucket([]byte("shopIDToken"))
				bucket.Put([]byte(shop_id), []byte(token))

				bucket = tx.Bucket([]byte("tokens"))
				value := tokensItem{[]string{}, []string{}, description, shop_id}
				json_value, err := json.Marshal(value)
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

			err = db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte("tokenExpirationTime"))
				bucket.Put([]byte(token), []byte(strconv.FormatInt(expiration_time, 10)))

				bucket = tx.Bucket([]byte("tokens"))
				json_value := bucket.Get([]byte(token))
				var value tokensItem
				err := json.Unmarshal(json_value, &value)
				if err != nil {
					return err
				}

				old_shop_id := value.ShopID
				value.ShopID = shop_id
				value.Description = description

				json_value, err = json.Marshal(value)
				if err != nil {
					return err
				}
				bucket.Put([]byte(token), json_value)

				bucket = tx.Bucket([]byte("shopIDToken"))
				bucket.Delete([]byte(old_shop_id))
				bucket.Put([]byte(shop_id), []byte(token))

				return nil
			})

			if err != nil {
				log.Print(err)
				http.Error(w, "500 internal server error", 500)
				return
			}

			fmt.Fprint(w, "ok")
			return

		} else if req_v == "delete" {
			token := r.FormValue("token")

			err := db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte("tokens"))
				json_value := bucket.Get([]byte(token))
				if json_value == nil {
					return nil
				}
				var value tokensItem
				err := json.Unmarshal(json_value, &value)
				if err != nil {
					return err
				}

				// Delete token
				err = bucket.Delete([]byte(token))
				if err != nil {
					return err
				}

				bucket = tx.Bucket([]byte("tokenExpirationTime"))
				err = bucket.Delete([]byte(token))
				if err != nil {
					return err
				}

				// Delete shop ID
				bucket = tx.Bucket([]byte("shopIDToken"))
				err = bucket.Delete([]byte(value.ShopID))
				if err != nil {
					return err
				}

				// Delete IDs
				bucket = tx.Bucket([]byte("IDToken"))
				bucket2 := tx.Bucket([]byte("IDImageIDs"))
				for _, id := range value.Ids {
					err = bucket.Delete([]byte(id))
					if err != nil {
						return err
					}
					err = bucket2.Delete([]byte(id))
					if err != nil {
						return err
					}
				}

				// Delete images
				bucket = tx.Bucket([]byte("imageIDToken"))
				for _, id := range value.Image_ids {
					err = bucket.Delete([]byte(id))
					if err != nil {
						return err
					}

					err = os.Remove("./images/" + id + ".jpg")
					if err != nil {
						return err
					}
				}
				return nil
			})

			if err != nil {
				log.Print(err)
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
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		bucketExpTime := tx.Bucket([]byte("tokenExpirationTime"))
		cursor := bucket.Cursor()

		num := 0
		for token, json_value := cursor.First(); token != nil; token, json_value = cursor.Next() {
			var value tokensItem
			err := json.Unmarshal(json_value, &value)
			if err != nil {
				return err
			}

			token_block := templates["token-block"]
			token_block = strings.ReplaceAll(token_block, "{{token}}", string(token))
			token_block = strings.ReplaceAll(token_block, "{{shop_id}}", value.ShopID)
			token_block = strings.ReplaceAll(token_block, "{{num}}", strconv.Itoa(num))

			if value.Description == "" {
				token_block = strings.ReplaceAll(token_block, "{{br}}", "")	
			} else {
				token_block = strings.ReplaceAll(token_block, "{{br}}", "<br/>")
			}
			token_block = strings.ReplaceAll(token_block, "{{description}}", value.Description)
			
			token_block = strings.ReplaceAll(token_block, "{{images_count}}", 
				strconv.Itoa(len(value.Image_ids)))
			token_block = strings.ReplaceAll(token_block, "{{ids_count}}", 
				strconv.Itoa(len(value.Ids)))

			expTimeString := bucketExpTime.Get([]byte(token))
			expTime, err := strconv.ParseInt(string(expTimeString), 10, 64)
			if err != nil {
				return err
			}

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

		return nil
	})

	if err != nil {
		log.Print(err)
		http.Error(w, "500 internal server error", 500)
		return
	}

	html = strings.ReplaceAll(html, "{{container}}", token_blocks)
	fmt.Fprint(w, html)
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