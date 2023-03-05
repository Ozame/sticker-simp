package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/ozame/sticker-simp/imaging"
)

var TOKEN string = os.Getenv("TELEGRAM_BOT_TOKEN")

var REQUESTURL string = "https://api.telegram.org/bot" + TOKEN + "/"

type Update struct {
	UpdateId int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	Text     string   `json:"text"`
	Chat     Chat     `json:"chat"`
	Document Document `json:"document"`
	Photos   []Photo  `json:"photo"`
}

type Photo struct {
	FileID string `json:"file_id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
}

// A Telegram Chat indicates the conversation to which the message belongs.
type Chat struct {
	Id int64 `json:"id"`
}

type FileResult struct {
	Ok   bool `json:"ok"`
	File File `json:"result"`
}

type File struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
}

func init() {
	functions.HTTP("sticker-bot", RunStickerCreation)
}

// parseTelegramRequest handles incoming update from the Telegram web hook
func parseTelegramRequest(r *http.Request) (*Update, error) {
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("could not decode incoming update %s", err.Error())
		return nil, err
	}
	r.Body.Close()
	return &update, nil
}

// sendPhotoToTelegramChat sends a text message to the Telegram chat identified by its chat Id
func sendPhotoToTelegramChat(chatId int64, reader *os.File) error {

	log.Printf("Sending photo to chat_id: %d", chatId)
	var telegramApi string = REQUESTURL + "sendPhoto"

	values := map[string]io.Reader{
		"chat_id": strings.NewReader(fmt.Sprintf("%d", chatId)),
		"photo":   reader,
	}

	err := Upload(chatId, telegramApi, values)

	if err != nil {
		log.Printf("Sending photo failed")
	}
	return err

}

// sendTextToTelegramChat sends a text message to the Telegram chat identified by its chat Id
func sendTextToTelegramChat(chatId int, text string) (string, error) {

	log.Printf("Sending %s to chat_id: %d", text, chatId)
	var telegramApi string = "https://api.telegram.org/bot" + TOKEN + "/sendMessage"
	response, err := http.PostForm(
		telegramApi,
		url.Values{
			"chat_id": {strconv.Itoa(chatId)},
			"text":    {text},
		})

	if err != nil {
		log.Printf("error when posting text to the chat: %s", err.Error())
		return "", err
	}
	defer response.Body.Close()

	var bodyBytes, errRead = io.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("error in parsing telegram answer %s", errRead.Error())
		return "", err
	}
	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)

	return bodyString, nil
}

// RunStickerCreation edits the received image and sends the result back to the chat
func RunStickerCreation(w http.ResponseWriter, r *http.Request) {

	// Parse incoming request
	var update, err = parseTelegramRequest(r)
	if err != nil {
		log.Printf("error parsing update, %s", err.Error())
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("%v", update)
	// Getting the photo file uri
	if (len(update.Message.Photos)) <= 0 {
		log.Printf("Warn - No photos included in the message")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	go handlePhotoFromUpdate(update)
}

func handlePhotoFromUpdate(update *Update) {
	bigPic := update.Message.Photos[len(update.Message.Photos)-1]
	getFileUrl := REQUESTURL + "getFile" + "?file_id=" + bigPic.FileID
	log.Printf("Getting file info from %s", getFileUrl)
	fileResp, err := http.Get(getFileUrl)
	if err != nil {
		log.Printf("Getting file failed")
	}
	var fileResult FileResult

	if err := json.NewDecoder(fileResp.Body).Decode(&fileResult); err != nil {
		log.Printf("could not decode incoming file %s", err.Error())
	}

	//Download the file
	fileUrl := "https://api.telegram.org/file/bot" + TOKEN + "/" + fileResult.File.FilePath
	log.Printf("Downloading file from %s", fileUrl)

	fileDownloadResp, err := http.Get(fileUrl)
	if err != nil {
		log.Printf("Downloading file failed")
	} else {
		log.Printf("Photo was downloaded successfully")
	}

	tmp, _ := os.CreateTemp(".", "tmp*.png")
	// not closing tmp file since it's needed later in goroutine
	imaging.RecodeAndScale(fileDownloadResp.Body, tmp)
	tmp.Seek(0, 0)
	log.Printf("Scaled the image")

	// Send the edited picture image back to Telegram
	var errTelegram = sendPhotoToTelegramChat(update.Message.Chat.Id, tmp)
	if errTelegram != nil {
		log.Printf("got error %s from telegram server", errTelegram.Error())
	} else {
		log.Printf("Edited picture was successfully shared to chat id %d", update.Message.Chat.Id)
	}
}

func Upload(chatId int64, url string, values map[string]io.Reader) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			defer os.Remove(x.Name())
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {

				return
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return err
		}

	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	log.Printf("making request to address %s", url)
	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %v", res)
	}
	return
}
