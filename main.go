package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

type UploadURLResponse struct {
	Ok        bool   `json:"ok"`
	FileID    string `json:"file_id"`
	UploadURL string `json:"upload_url"`
}

func main() {
	os.Setenv("CHANNEL_ID", "C07MDHHLCHK")
	os.Setenv("TOKEN", "xoxb-7723840493030-7759198065952-6gEE05CZv1muXgCDl9tLEUrg")
	// token := "xoxb-7723840493030-7759198065952-6gEE05CZv1muXgCDl9tLEUrg"
	filename := "resume_.pdf"

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Cannot open file:", err)
		return
	}
	defer file.Close()

	fileinfo, err := file.Stat()
	if err != nil {
		fmt.Println("Cannot get fileinfo:", err)
		return
	}
	filesize := fileinfo.Size()

	// Create the form data
	payload := fmt.Sprintf("filename=%s&length=%d", filename, filesize)

	req, err := http.NewRequest("POST", "https://slack.com/api/files.getUploadURLExternal", strings.NewReader(payload))
	if err != nil {
		fmt.Println("Could not create request:", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("TOKEN"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to make request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response:", err)
		return
	}

	var uploadurlrep UploadURLResponse
	err = json.Unmarshal(body, &uploadurlrep)
	if err != nil {
		fmt.Println("Failed to parse response:", err)
		return
	}

	if uploadurlrep.Ok {
		fmt.Println("Upload URL:", uploadurlrep.UploadURL)
		fmt.Println("File ID:", uploadurlrep.FileID)
	} else {
		fmt.Println("Failed to get upload URL. Error:")
		fmt.Println("Full response:", string(body))
	}


	fmt.Println("Upload URL:", uploadurlrep.UploadURL)
	fmt.Println("File ID to complete upload:", uploadurlrep.FileID)


	err=uploadFile(uploadurlrep.UploadURL,filename)
	if err!=nil{
		fmt.Println("failed to upload file:",err)
	}

	err= completeUpload(uploadurlrep.FileID,os.Getenv("CHANNEL_ID"))
	if err!=nil{
		fmt.Println("Failed to complete upload: ",err)
	}
}

func uploadFile(uploadURL, filename string) error{
	file,err:=os.Open(filename)
	if err!=nil{
		fmt.Println("Could not open file! :%w",err) 
	}
    defer file.Close()

	body:=&bytes.Buffer{}
	writer:= multipart.NewWriter(body)


	part,err:=writer.CreateFormFile("file",filename)
	if err != nil {
		return fmt.Errorf("could not create form file: %w", err)
	}

	_, err=io.Copy(part,file)
	if err!=nil{
		return fmt.Errorf("could not copy file: %w", err)
	}

	err= writer.Close()
	if err!=nil{
		return fmt.Errorf("could not close writer: %w", err)
	}

	req,err:=http.NewRequest("POST",uploadURL,body)
	if err!=nil{
		return fmt.Errorf("could not create request: %w", err)
	}
    
	req.Header.Set("Authorization", "Bearer "+os.Getenv("TOKEN"))
	req.Header.Set("Content-Type",writer.FormDataContentType())

	client:=&http.Client{}
    resp,err:=client.Do(req)
	if err!=nil{
		fmt.Println("could not upload file",err)
	}
    defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK{
		body,_:=io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d : %s",resp.StatusCode,body)
	}

	fmt.Println("File uploaded succesfully!")
	return nil
}
func completeUpload(fileID, channelID string) error {
    // Prepare the payload with correct format
    payload := fmt.Sprintf(`{
        "files": [
            {"id": "%s"}
        ],
        "channel_id": "%s"
    }`, fileID, channelID)
    
    // Debugging: Log the payload before sending
    fmt.Println("Payload being sent to Slack:")
    fmt.Println(payload)

    // Create the HTTP request
    req, err := http.NewRequest("POST", "https://slack.com/api/files.completeUploadExternal", strings.NewReader(payload))
    if err != nil {
        return fmt.Errorf("could not create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+os.Getenv("TOKEN"))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to complete upload: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }

    // Print the full response for debugging
    fmt.Println("Response from Slack completeUpload API:", string(body))

    var completionResponse struct {
        Ok    bool   `json:"ok"`
        Files []struct {
            ID       string   `json:"id"`
            Channels []string `json:"channels"`
        } `json:"files"`
        Error string `json:"error,omitempty"`
    }

    err = json.Unmarshal(body, &completionResponse)
    if err != nil {
        return fmt.Errorf("failed to parse response: %w", err)
    }

    if !completionResponse.Ok {
        return fmt.Errorf("completion failed: %s", completionResponse.Error)
    }

    if len(completionResponse.Files[0].Channels) == 0 {
        return fmt.Errorf("file was uploaded but not associated with any channel")
    }

    fmt.Println("File successfully shared in the channel!")
    return nil
}
