package main

import (
	"encoding/json"
	"http"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
)

const TOKEN string = "read from environment"
const REQUESTURL string = "https://api.telegram.org/" + TOKEN + "/"

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
	Id int `json:"id"`
}

// parseTelegramRequest handles incoming update from the Telegram web hook
func parseTelegramRequest(r *http.Request) (*Update, error) {
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("could not decode incoming update %s", err.Error())
		return nil, err
	}
	return &update, nil
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

	var bodyBytes, errRead = ioutil.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("error in parsing telegram answer %s", errRead.Error())
		return "", err
	}
	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)

	return bodyString, nil
}

// HandleTelegramWebHook sends a message back to the chat with a punchline starting by the message provided by the user.
func HandleTelegramWebHook(w http.ResponseWriter, r *http.Request) {

	// Parse incoming request
	var update, err = parseTelegramRequest(r)
	if err != nil {
		log.Printf("error parsing update, %s", err.Error())
		return
	}

	bigPic := update.Message.Photos[len(update.Message.Photos)-1]
	// TODO: Now get the photo file itself
	getFileUrl := REQUESTURL + "getFile" + "?file_id=" + bigPic.FileID

	//TODO: download the file
	fileUrl := "https://api.telegram.org/file/" + TOKEN + "/<file_path>"

	// Send the punchline back to Telegram
	var telegramResponseBody, errTelegram = sendTextToTelegramChat(update.Message.Chat.Id, "something")
	if errTelegram != nil {
		log.Printf("got error %s from telegram, reponse body is %s", errTelegram.Error(), telegramResponseBody)
	} else {
		log.Printf("punchline %s successfuly distributed to chat id %d", "something", update.Message.Chat.Id)
	}
}

func main() {
	// err := http.ListenAndServe(":8080", http.HandlerFunc(HandleTelegramWebHook))
	// if err != nil {
	//     log.Fatal(err)
	//     return
	// }
}
