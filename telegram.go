package messeger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
)

type Telegram struct {
	URL      string
	BotToken string
}

type Update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID int `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type GetUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type WebhookInfo struct {
	Ok     bool `json:"ok"`
	Result struct {
		URL                  string `json:"url"`
		HasCustomCertificate bool   `json:"has_custom_certificate"`
		PendingUpdateCount   int    `json:"pending_update_count"`
	} `json:"result"`
}

func (t Telegram) SetWebHook(urlWebHook string) error {
	apiURL := fmt.Sprintf(t.URL+"/bot%s/setWebhook?url=%s", t.BotToken, urlWebHook)
	// Criar buffer para armazenar o body da requisição
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Adicionar URL do webhook
	_ = writer.WriteField("url", urlWebHook)

	certPath := "server.pem"

	// Adicionar certificado como arquivo
	file, err := os.Open(certPath)
	if err != nil {
		log.Fatal("Erro ao abrir o certificado:", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("certificate", certPath)
	if err != nil {
		log.Fatal("Erro ao anexar o certificado:", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		log.Fatal("Erro ao copiar o certificado:", err)
	}

	// Fechar o multipart writer
	writer.Close()

	// Criar requisição HTTP
	req, err := http.NewRequest("POST", apiURL, &body)
	if err != nil {
		log.Fatal("Erro ao criar requisição:", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Enviar requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Erro ao enviar requisição:", err)
	}
	defer resp.Body.Close()

	// Exibir resposta
	fmt.Println("Resposta do Telegram:", resp.Status)

	return nil
}

func (t Telegram) DeleteWebHook() error {
	apiURL := fmt.Sprintf(t.URL+"/bot%s/deleteWebhook", t.BotToken)
	slog.Debug(apiURL)
	// Criando os dados do POST
	data := url.Values{}

	// Fazendo a requisição HTTP
	resp, err := http.PostForm(apiURL, data)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	slog.Info("messeger :: DeleteWebHook :: status code -> " + fmt.Sprint(resp.StatusCode))

	// Verifica se a resposta foi bem-sucedida
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao enviar mensagem, status: %d", resp.StatusCode)
	}

	return nil
}

func (t Telegram) GetWebHookInfo() (*WebhookInfo, error) {
	apiURL := fmt.Sprintf(t.URL+"/bot%s/getWebhookInfo", t.BotToken)
	slog.Debug(apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		slog.Error("Erro ao fazer a requisição: " + err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Erro ao ler resposta:" + err.Error())
		return nil, err
	}

	var info WebhookInfo
	if err := json.Unmarshal(body, &info); err != nil {
		slog.Error("Erro ao decodificar JSON:" + err.Error())
		return nil, err
	}

	fmt.Println(info)

	return &info, nil
}

func (t Telegram) GetUpdates(offset int) ([]Update, error) {
	url := fmt.Sprintf(t.URL+"/bot%s/getUpdates?offset=%d", t.BotToken, offset)
	slog.Debug(url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var updates GetUpdatesResponse
	err = json.Unmarshal(body, &updates)
	if err != nil {
		return nil, err
	}

	slog.Debug(string(body))

	if !updates.Ok {
		return nil, errors.New("Fail to get updades!")
	}

	return updates.Result, nil
}

func (t Telegram) SendMessage(chatID int, message string) error {
	apiURL := fmt.Sprintf(t.URL+"/bot%s/sendMessage", t.BotToken)

	// Criando os dados do POST
	data := url.Values{}
	data.Set("chat_id", fmt.Sprint(chatID))
	data.Set("text", message)

	// Fazendo a requisição HTTP
	resp, err := http.PostForm(apiURL, data)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	slog.Info("telegram :: SendMessage :: status code -> " + fmt.Sprint(resp.StatusCode))

	// Verifica se a resposta foi bem-sucedida
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Fail to send message, status: %d" + fmt.Sprint(resp.StatusCode))
	}

	return nil
}

func (t Telegram) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Fail to read request body!", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		http.Error(w, "Fail to processa json!", http.StatusInternalServerError)
		return
	}

	slog.Info("telegram :: WebhookHandler :: message -> " + update.Message.Text)

	w.WriteHeader(http.StatusOK)
}
