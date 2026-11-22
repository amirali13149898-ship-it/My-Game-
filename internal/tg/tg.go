// Package tg is a tiny Telegram Bot API client (net/http only, no dependencies).
package tg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"` // private, group, supergroup, channel
}

type PhotoSize struct {
	FileID string `json:"file_id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Message struct {
	Photo          []PhotoSize `json:"photo"`
	Caption        string      `json:"caption"`
	MessageID      int         `json:"message_id"`
	From           *User       `json:"from"`
	Chat           Chat        `json:"chat"`
	Text           string      `json:"text"`
	ReplyToMessage *Message    `json:"reply_to_message"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// Button is an inline keyboard button. Style ("primary" = blue, "success" =
// green, "danger" = red) is the colour field of newer Bot API versions;
// clients/servers that do not know it simply ignore it.
type Button struct {
	Text  string `json:"text"`
	Data  string `json:"callback_data,omitempty"`
	URL   string `json:"url,omitempty"`
	Style string `json:"style,omitempty"`
}

type Keyboard [][]Button

func (k Keyboard) markup() any {
	if len(k) == 0 {
		return nil
	}
	return map[string]any{"inline_keyboard": k}
}

type Client struct {
	token string
	hc    *http.Client
	Me    User
}

func New(token string) *Client {
	return &Client{token: token, hc: &http.Client{Timeout: 75 * time.Second}}
}

type apiResp struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func (c *Client) do(method string, body io.Reader, ctype string, out any) error {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest("POST", "https://api.telegram.org/bot"+c.token+"/"+method, body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", ctype)
		resp, err := c.hc.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %w", method, scrub(err, c.token))
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var ar apiResp
		if err := json.Unmarshal(raw, &ar); err != nil {
			return fmt.Errorf("%s: bad response (%d)", method, resp.StatusCode)
		}
		if !ar.OK {
			if ar.ErrorCode == 429 && attempt < 3 && ar.Parameters.RetryAfter > 0 {
				time.Sleep(time.Duration(ar.Parameters.RetryAfter) * time.Second)
				if b, ok := body.(*bytes.Reader); ok {
					b.Seek(0, io.SeekStart)
					continue
				}
			}
			return &APIError{Method: method, Code: ar.ErrorCode, Desc: ar.Description}
		}
		if out != nil {
			return json.Unmarshal(ar.Result, out)
		}
		return nil
	}
}

func scrub(err error, token string) error {
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), token, "<token>"))
}

type APIError struct {
	Method string
	Code   int
	Desc   string
}

func (e *APIError) Error() string { return fmt.Sprintf("telegram %s: %d %s", e.Method, e.Code, e.Desc) }

// NotModified reports the harmless "message is not modified" error.
func NotModified(err error) bool {
	e, ok := err.(*APIError)
	return ok && strings.Contains(e.Desc, "not modified")
}

func (c *Client) Call(method string, params any, out any) error {
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.do(method, bytes.NewReader(b), "application/json", out)
}

func (c *Client) Init() error { return c.Call("getMe", map[string]any{}, &c.Me) }

func (c *Client) GetUpdates(offset, timeout int) ([]Update, error) {
	var ups []Update
	err := c.Call("getUpdates", map[string]any{
		"offset": offset, "timeout": timeout,
		"allowed_updates": []string{"message", "callback_query"},
	}, &ups)
	return ups, err
}

type SendOpts struct {
	ReplyTo  int
	Keyboard Keyboard
	Silent   bool
}

func (c *Client) Send(chatID int64, text string, o SendOpts) (*Message, error) {
	p := map[string]any{"chat_id": chatID, "text": text, "parse_mode": "HTML",
		"link_preview_options": map[string]any{"is_disabled": true}}
	if o.ReplyTo != 0 {
		p["reply_parameters"] = map[string]any{"message_id": o.ReplyTo, "allow_sending_without_reply": true}
	}
	if m := o.Keyboard.markup(); m != nil {
		p["reply_markup"] = m
	}
	if o.Silent {
		p["disable_notification"] = true
	}
	var msg Message
	err := c.Call("sendMessage", p, &msg)
	return &msg, err
}

func (c *Client) Edit(chatID int64, msgID int, text string, kb Keyboard) error {
	p := map[string]any{"chat_id": chatID, "message_id": msgID, "text": text, "parse_mode": "HTML",
		"link_preview_options": map[string]any{"is_disabled": true}}
	p["reply_markup"] = map[string]any{"inline_keyboard": kb}
	if len(kb) == 0 {
		p["reply_markup"] = map[string]any{"inline_keyboard": [][]Button{}}
	}
	err := c.Call("editMessageText", p, nil)
	if NotModified(err) {
		return nil
	}
	return err
}

func (c *Client) Answer(cbID, text string, alert bool) error {
	p := map[string]any{"callback_query_id": cbID}
	if text != "" {
		p["text"] = text
		p["show_alert"] = alert
	}
	return c.Call("answerCallbackQuery", p, nil)
}

// SendDocument uploads a local file (used for database backups).
func (c *Client) SendDocument(chatID int64, path, caption string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("chat_id", fmt.Sprint(chatID))
	w.WriteField("caption", caption)
	part, err := w.CreateFormFile("document", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	w.Close()
	return c.do("sendDocument", &buf, w.FormDataContentType(), nil)
}

// BestPhoto returns the file_id of the largest size of a photo message.
func (m *Message) BestPhoto() string {
	best, area := "", -1
	for _, p := range m.Photo {
		if a := p.Width * p.Height; a > area {
			best, area = p.FileID, a
		}
	}
	return best
}

// SendPhoto sends a photo by file_id (path == "") or by uploading a local file.
func (c *Client) SendPhoto(chatID int64, fileID, path, caption string, kb Keyboard) (*Message, error) {
	var msg Message
	if path == "" {
		p := map[string]any{"chat_id": chatID, "photo": fileID, "caption": caption, "parse_mode": "HTML"}
		if m := kb.markup(); m != nil {
			p["reply_markup"] = m
		}
		return &msg, c.Call("sendPhoto", p, &msg)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("chat_id", fmt.Sprint(chatID))
	w.WriteField("caption", caption)
	w.WriteField("parse_mode", "HTML")
	if m := kb.markup(); m != nil {
		j, _ := json.Marshal(m)
		w.WriteField("reply_markup", string(j))
	}
	part, err := w.CreateFormFile("photo", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	w.Close()
	return &msg, c.do("sendPhoto", &buf, w.FormDataContentType(), &msg)
}

// EditPhoto swaps the photo (by file_id), caption and keyboard of a photo message.
func (c *Client) EditPhoto(chatID int64, msgID int, fileID, caption string, kb Keyboard) error {
	p := map[string]any{"chat_id": chatID, "message_id": msgID,
		"media":        map[string]any{"type": "photo", "media": fileID, "caption": caption, "parse_mode": "HTML"},
		"reply_markup": map[string]any{"inline_keyboard": kb}}
	if len(kb) == 0 {
		p["reply_markup"] = map[string]any{"inline_keyboard": [][]Button{}}
	}
	err := c.Call("editMessageMedia", p, nil)
	if NotModified(err) {
		return nil
	}
	return err
}

func (c *Client) Delete(chatID int64, msgID int) error {
	return c.Call("deleteMessage", map[string]any{"chat_id": chatID, "message_id": msgID}, nil)
}

// SendPhotoBytes uploads a freshly generated JPEG.
func (c *Client) SendPhotoBytes(chatID int64, data []byte, caption string, kb Keyboard) (*Message, error) {
	var msg Message
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("chat_id", fmt.Sprint(chatID))
	w.WriteField("caption", caption)
	w.WriteField("parse_mode", "HTML")
	if m := kb.markup(); m != nil {
		j, _ := json.Marshal(m)
		w.WriteField("reply_markup", string(j))
	}
	part, err := w.CreateFormFile("photo", "card.jpg")
	if err != nil {
		return nil, err
	}
	part.Write(data)
	w.Close()
	return &msg, c.do("sendPhoto", &buf, w.FormDataContentType(), &msg)
}

// EditPhotoBytes replaces the picture of a photo message with a generated JPEG.
func (c *Client) EditPhotoBytes(chatID int64, msgID int, data []byte, caption string, kb Keyboard) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("chat_id", fmt.Sprint(chatID))
	w.WriteField("message_id", fmt.Sprint(msgID))
	media, _ := json.Marshal(map[string]any{"type": "photo", "media": "attach://card", "caption": caption, "parse_mode": "HTML"})
	w.WriteField("media", string(media))
	markup := map[string]any{"inline_keyboard": kb}
	if len(kb) == 0 {
		markup = map[string]any{"inline_keyboard": [][]Button{}}
	}
	j, _ := json.Marshal(markup)
	w.WriteField("reply_markup", string(j))
	part, err := w.CreateFormFile("card", "card.jpg")
	if err != nil {
		return err
	}
	part.Write(data)
	w.Close()
	err = c.do("editMessageMedia", &buf, w.FormDataContentType(), nil)
	if NotModified(err) {
		return nil
	}
	return err
}
