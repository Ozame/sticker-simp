package function

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ozame/sticker-simp/imaging"
)

var chat = Chat{Id: 1}

func TestParseUpdateMessage(t *testing.T) {

	requestBody, err := os.Open("test-update.json")
	if err != nil {
		t.Errorf("Failed to marshal update in json, got %s", err.Error())
	}
	req := httptest.NewRequest("POST", "http://myTelegramWebHookHandler.com/secretToken", requestBody)

	var updateToTest, errParse = parseTelegramRequest(req)

	if updateToTest.Message.Text != "test message" {
		t.Errorf("Different Text content")
	}

	if len(updateToTest.Message.Photos) != 2 {
		t.Errorf("Photo count not matching")
	}
	if errParse != nil {
		t.Errorf("Expected a <nil> error, got %s", errParse.Error())
	}

}

func TestSendPhoto(t *testing.T) {

	reader, _ := os.Open("brocc.jpg")
	defer reader.Close()

	tmp, _ := os.CreateTemp(".", "tmp")
	defer tmp.Close()
	defer os.Remove("tmp")

	imaging.RecodeAndScale(reader, tmp)
	tmp.Seek(0, 0)

	var id int64 = 0 // Replace this with the chat id to test functionality

	err := sendPhotoToTelegramChat(id, tmp)

	if err != nil {
		t.Errorf("failed upload %v", err)
	}
}
