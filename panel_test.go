package main

import (
	"testing"
	"time"
	"fmt"
)

var (
	uuidValid string
)

func createToken(t *testing.T, uuid, token, shop_id, description, exp_time string) bool {
	resp, body := request(baseURL + "dc-admin-p/tokens", "POST", map[string]string{
		"v": "create",
		"token": token,
		"shop_id": shop_id,
		"description": description,
		"exp_time": exp_time,
	}, map[string]string{"uuid": uuid})

	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("Status code doesn't equal 200")
	}
	return string(body) == "ok"
}

func login(t *testing.T) (uuid string) {
	resp, body := request(baseURL + "dc-admin-p/", "POST", map[string]string{
		"login": defaultLogin,
		"password": defaultPassword,
	}, nil)


	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("Status code doesn't equal 200")
	}

	if string(body) != "ok" {
		t.Fatalf("Body doesn't equal \"ok\"")	
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "uuid" {
			uuid = cookie.Value
			return
		}
	}
	t.Fatalf("No uuid cookie in server response")
	return
}

func testCreateToken(t *testing.T) {
	fmt.Println("try with empty uuid")
	if createToken(t, "", "123", "123", "123", "123") {
		t.Fatalf("Successful createToken with invalid data")
	}
	fmt.Println("try with empty all fields")
	if createToken(t, "", "", "", "", "") {
		t.Fatalf("Successful createToken with invalid data")
	}
	fmt.Println("try with sample uuid")
	if createToken(t, "1234-9182", "test_token", "123", "123", "123") {
		t.Fatalf("Successful createToken with invalid data")
	}

	fmt.Println("login")
	uuidValid = login(t)

	fmt.Println("try create a token")
	if !createToken(t, uuidValid, "t_1", "1234", "description", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Failed createToken with valid data")
	}
	fmt.Println("try create the same token")
	if createToken(t, uuidValid, "t_1", "9876", "description", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Successful createToken with invalid data")
	}
	fmt.Println("try create a token with existing shop_id")
	if createToken(t, uuidValid, "t_2", "1234", "description", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Successful createToken with invalid data")
	}
	fmt.Println("try create expired token")
	if !createToken(t, uuidValid, "t_2", "9876", "description", getUnixTime(-10 * time.Minute)) {
		t.Fatalf("Failed createToken with valid data")
	}

	t.Run("Test uploading with valid token", testUploadSuccess("t_1"))
	t.Run("Test uploading with valid token", testUploadSuccess("t_1"))
	t.Run("Test uploading with valid token", testUploadSuccess("t_1"))
	t.Run("Test uploading with valid token", testUploadSuccess("t_1"))
	t.Run("Test uploading with valid token", testUploadSuccess("t_1"))
	t.Run("Test uploading with invalid token", testUploadFail("t_2"))

	checkImagesSuccess(t, imageIDsByToken["t_1"])
}

func editToken(t *testing.T, uuid, token, shop_id, description, exp_time string) bool {
	resp, body := request(baseURL + "dc-admin-p/tokens", "POST", map[string]string{
		"v": "edit",
		"token": token,
		"shop_id": shop_id,
		"description": description,
		"exp_time": exp_time,
	}, map[string]string{"uuid": uuid})

	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("Status code doesn't equal 200")
	}
	return string(body) == "ok"
}

func testEditToken(t *testing.T) {
	fmt.Println("try with empty uuid")
	if editToken(t, "", "t_1", "2345", "", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Successful editToken with invalid data")
	}
	fmt.Println("try edit nonexistent token")
	if editToken(t, uuidValid, "t_123", "2345", "", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Successful editToken with invalid data")
	}
	fmt.Println("try change shop_id to already existing")
	if editToken(t, uuidValid, "t_1", "9876", "", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Successful editToken with invalid data")
	}
	fmt.Println("try not change shop_id")
	if !editToken(t, uuidValid, "t_1", "1234", "", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Failed editToken with valid data")
	}
	fmt.Println("try change shop_id")
	if !editToken(t, uuidValid, "t_1", "2345", "", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Failed editToken with valid data")
	}

	checkImagesSuccess(t, imageIDsByToken["t_1"])

	fmt.Println("try change exp_time")
	if !editToken(t, uuidValid, "t_1", "1234", "", getUnixTime(-10 * time.Minute)) {
		t.Fatalf("Failed editToken with valid data")
	}

	checkImagesFail(t, imageIDsByToken["t_1"])

	fmt.Println("try change exp_time")
	if !editToken(t, uuidValid, "t_1", "1234", "", getUnixTime(10 * time.Minute)) {
		t.Fatalf("Failed editToken with valid data")
	}

	checkImagesSuccess(t, imageIDsByToken["t_1"])
}

func deleteToken(t *testing.T, uuid, token string) bool {
	resp, body := request(baseURL + "dc-admin-p/tokens", "POST", map[string]string{
		"v": "delete",
		"token": token,
	}, map[string]string{"uuid": uuid})

	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("Status code doesn't equal 200")
	}
	return string(body) == "ok"
}

func testDeleteToken(t *testing.T) {
	fmt.Println("try with empty uuid")
	if deleteToken(t, "", "t_1") {
		t.Fatalf("Successful deleteToken with invalid data")
	}
	fmt.Println("try delete nonexistent token")
	if !deleteToken(t, uuidValid, "t_123") {
		t.Fatalf("Failed deleteToken with valid data")
	}

	checkImagesSuccess(t, imageIDsByToken["t_1"])

	fmt.Println("try delete the first token")
	if !deleteToken(t, uuidValid, "t_1") {
		t.Fatalf("Failed deleteToken with valid data")
	}

	checkImagesFail(t, imageIDsByToken["t_1"])

	fmt.Println("try delete the second token")
	if !deleteToken(t, uuidValid, "t_2") {
		t.Fatalf("Failed deleteToken with valid data")
	}
	fmt.Println("try delete the second token again")
	if !deleteToken(t, uuidValid, "t_2") {
		t.Fatalf("Failed deleteToken with valid data")
	}
}