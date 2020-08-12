package main

import (
	"github.com/boltdb/bolt"
	"log"
	"os"
	"fmt"
	"strings"
	"bufio"
)

func main() {
	path := strings.Split(os.Args[0], "/")
	using_command := fmt.Sprintf("go run %s.go [path]", path[len(path)-1])
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("Using:", using_command)
		return
	}
	filename := args[0]

	db, err := bolt.Open(filename, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	fmt.Println("Database successfully opened")


	getAllBuckets := func() []string {
		var result []string
		err = db.View(func(tx *bolt.Tx) error {
			return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
				result = append(result, string(name))
				return nil
			})
		})

		if err != nil {
			db.Close()
			log.Fatal(err)
		}
		return result
	}

	createBucket := func(name string) {
		err := db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucket([]byte(name))
			return err
		})

		if err != nil {
			db.Close()
			log.Fatal(err)
		}
	}

	createBucketIfNotExists := func(name string) {
		err := db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(name))
			return err
		})

		if err != nil {
			db.Close()
			log.Fatal(err)
		}
	}

	deleteBucket := func(name string) {
		err := db.Update(func(tx *bolt.Tx) error {
			err := tx.DeleteBucket([]byte(name))
			return err
		})

		if err != nil {
			db.Close()
			log.Fatal(err)
		}
	}

	getAllValues := func(bucketName string) ([]string, []string) {
		var keys, values []string
		err := db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketName))
			cursor := bucket.Cursor()

			for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
				keys = append(keys, string(key))
				values = append(values, string(value))
			}
			return nil
		})

		if err != nil {
			db.Close()
			log.Fatal(err)
		}
		return keys, values
	}

	setValue := func(bucketName string, key string, value string) {
		err := db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketName))
			err := bucket.Put([]byte(key), []byte(value))
			return err
		})

		if err != nil {
			db.Close()
			log.Fatal()
		}
	}

	getValue := func(bucketName string, key string) string  {
		var result string
		err := db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketName))
			value := bucket.Get([]byte(key))
			result = string(value)
			return nil
		})

		if err != nil {
			db.Close()
			log.Fatal()
		}
		return result
	}

	deleteKey := func(bucketName string, key string) {
		err := db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketName))
			err := bucket.Delete([]byte(key))
			return err
		})

		if err != nil {
			db.Close()
			log.Fatal(err)
		}
	}


	fmt.Println("List of available commands:")
	fmt.Println("list - list of available buckets")
	fmt.Println("create [name] - create bucket")
	fmt.Println("delete [name] - delete bucket")
	fmt.Println("select [name] - select bucket")
	fmt.Println("exit - close database")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		var command string
		if scanner.Scan() {
			command = scanner.Text()
		}
		commands := strings.Split(command, " ")
		switch (commands[0]) {
		case "list":
			list := getAllBuckets()
			fmt.Println("Available buckets:")
			for _, i := range list {
				fmt.Println(i)
			}
		case "create":
			if len(commands) < 2 {
				fmt.Println("Using: create [name]")
				break
			}
			createBucket(commands[1])
		case "delete":
			if len(commands) < 2 {
				fmt.Println("Using: delete [name]")
				break
			}
			deleteBucket(commands[1])
		case "select":
			if len(commands) < 2 {
				fmt.Println("Using: select [name]")
				break
			}

			bucket := commands[1]
			createBucketIfNotExists(bucket)
			fmt.Println("Selected bucket " + bucket)
			fmt.Println("List of available commands:")
			fmt.Println("list - list of all keys with values")
			fmt.Println("set [key] [value] - set value")
			fmt.Println("get [key] - get value by key")
			fmt.Println("delete [key] - delete key")
			fmt.Println("exit - return to selection buckets")

			for {
				fmt.Print("> ")
				var command string
				if scanner.Scan() {
					command = scanner.Text()
				}
				commands := strings.Split(command, " ")
				exitFlag := false
				switch (commands[0]) {
				case "list":
					keys, values := getAllValues(bucket)
					fmt.Println("All values in bucket " + bucket + ":")
					for i := range keys {
						fmt.Println(keys[i] + ": " + values[i])
					}
				case "set":
					if len(commands) < 3 {
						fmt.Println("Using: set [key] [value]")
						break
					}
					setValue(bucket, commands[1], commands[2])
				case "get":
					if len(commands) < 2 {
						fmt.Println("Using: get [key]")
						break
					}
					fmt.Println(getValue(bucket, commands[1]))
				case "delete":
					if len(commands) < 2 {
						fmt.Println("Using: delete [key]")
						break
					}
					deleteKey(bucket, commands[1])
				case "exit":
					exitFlag = true
				default:
					fmt.Println("unknown command")
				}
				if exitFlag {
					break
				}
			}

			list := getAllBuckets()
			fmt.Println("Available buckets:")
			for _, i := range list {
				fmt.Println(i)
			}

		case "exit":
			return
		default:
			fmt.Println("unknown command")
		}
	}
}